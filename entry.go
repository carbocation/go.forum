/*
Entry is the fundamental unit of a threaded discussion. An entry can represent
a forum, a post, or a comment, depending on how it is annotated. There is nothing
fundamentally distinct about these things, and their similarities (including
hierarchical nesting) are abundant.
*/
package forum

import (
	"database/sql"
	"errors"
	"time"
)

// Put ModifiedBy, ModifiedAuthor in a separate table. A post can only be
// created once but modified an infinite number of times.
type Entry struct {
	Id       int64     "The ID of the post"
	Title    string    "Title of the post. Will be empty for entries that are really intended to be comments."
	Body     string    "Contents of the post. Will be empty for entries that are intended to be links."
	Url      string    //Used if the post is just a link
	Created  time.Time "Time at which the post was created."
	AuthorId int64     "ID of the author of the post"
	Forum    bool      `schema:"-"` //Is this Entry actually a forum instead?

	//These are not stored in the DB and are just generated fields
	AuthorHandle string  //Name of the author
	Seconds      float64 //Seconds since creation
	Upvotes      int64
	Downvotes    int64
	
	parent, child, sibling *Entry //Mandatory pointer-holders for Tree-ness
}

// Stores an entry to the database and correctly builds its ancestry based
// on its parent's ID.
func (e *Entry) Persist(parentId int64) error {
	//Wrap in a transaction
	tx, err := Config.DB.Begin()

	EntryCreateStmt, err := tx.Prepare(queries.EntryCreate)
	if err != nil {
		_ = tx.Rollback()
		return errors.New("Error: We had a database problem trying to create your entry.")
	}
	defer EntryCreateStmt.Close()

	//Note: because pq handles LastInsertId oddly (or not at all?), instead of
	//calling .Exec() then .LastInsertId, we prepare a statement that ends in
	//`RETURNING id` and we .QueryRow().Select() the result
	err = EntryCreateStmt.QueryRow(e.Title, e.Body, e.Url, e.AuthorId).Scan(&e.Id)
	if err != nil {
		tx.Rollback()
		return errors.New("Error: your username or email address was already found in the database. Please choose differently.")
	}

	EntryClosureTableCreateStmt, err := tx.Prepare(queries.EntryClosureTableCreate)
	if err != nil {
		tx.Rollback()
		// return err
		return errors.New("Error: We had a database problem trying to create ancestry information.")
	}
	defer EntryClosureTableCreateStmt.Close()

	_, err = EntryClosureTableCreateStmt.Exec(e.Id, parentId)
	if err != nil {
		tx.Rollback()
		return errors.New("Error: We couldn't save the relationship between your comment and its parent comment.")
	}

	tx.Commit()

	return nil
}

func (e Entry) Points() int64 {
	return e.Upvotes - e.Downvotes
}

func (e *Entry) Child() *Entry { return e.child }
func (e *Entry) Sibling() *Entry { return e.sibling }
func (e *Entry) Parent() *Entry { return e.parent }

func (e *Entry) AddChild(newE *Entry) {
	if e.child == nil {
		//Slot is available, directly add the child
		e.child, newE.parent = newE, e
	} else {
		//Slot is unavailable, figure out where the child belongs among peer siblings
		e.child.addSibling(newE)
	}
	
	return
}

func (e *Entry) addSibling(newE *Entry) {
	if newE == nil {
		return
	}

	if newE.Points() <= e.Points() {
		// The new element belongs BELOW the old one
		if e.sibling == nil {
			// The old element has no sibling so insertion below it is trivial
			newE.parent, e.sibling = e, newE
			
			return
		} else {
			// The old element already has a sibling
			// Try to add the new element as a sibling of the sibling
			e.sibling.addSibling(newE)
			
			return 
		}
	} else {
		// The new element belongs ABOVE the old one
		
		// New element may or may not have a sibling, but we will pop it off and then add it back 
		// at the end to cover our bases in case it does
		newESib := newE.sibling
		
		if e == e.parent.child {
			// Old element was a child of its parent
			e.parent.child, newE.parent, newE.sibling, e.parent = newE, e.parent, e, newE
		} else {
			// Old element was presumptively a sibling of its parent
			e.parent.sibling, newE.parent, newE.sibling, e.parent = newE, e.parent, e, newE
		}
		
		// Add back sibling of new element (if any)
		newE.addSibling(newESib)
		
		return
	}
}

//Retrieve one entry by its ID, if it exists. Error if not.
func OneEntry(id int64) (*Entry, error) {
	e := new(Entry)
	var err error = nil

	stmt, err := Config.DB.Prepare(queries.OneEntry)
	if err != nil {
		return e, err
	}
	defer stmt.Close()

	var body, url sql.NullString
	err = stmt.QueryRow(id).Scan(&e.Id, &e.Title, &body, &url, &e.Created, &e.AuthorId, &e.Forum, &e.AuthorHandle, &e.Seconds, &e.Upvotes, &e.Downvotes)
	if err != nil {
		e = new(Entry)
		return e, err
	}

	//Only the body or the url will be set; they are mutually exclusive
	if body.Valid {
		e.Body = body.String
	} else if url.Valid {
		e.Url = url.String
	}

	return e, err
}

// Retrieves all entries that are descendants of the ancestral entry, including the ancestral entry itself
func DescendantEntries(root int64) (*Entry, error) {
	return getEntries(root, "AllDescendants")
}

// Retrieves entries that are immediate descendants of the ancestral entry, including the ancestral entry itself
func DepthOneDescendantEntries(root int64) (*Entry, error) {
	return getEntries(root, "DepthOneDescendants")
}

func getEntries(root int64, flag string) (*Entry, error) {
	// Store output in a map initially. Get it all in here before you try to build the tree.
	entries := map[int64]*Entry{} //k: id => v: Entry
	relationships := make([]map[string]int64,0) //A slice of maps with k: parentId in entries map => v: childId in entries map
	
	var stmt *sql.Stmt
	var err error
	switch flag {
	case "AllDescendants":
		stmt, err = Config.DB.Prepare(queries.DescendantEntriesChildParent)
	case "DepthOneDescendants":
		stmt, err = Config.DB.Prepare(queries.DepthOneDescendantEntriesChildParent)
	}
	if err != nil {
		return &Entry{}, err
	}
	defer stmt.Close()
	
	// Query from that prepared statement
	rows, err := stmt.Query(root)
	if err != nil {
		return &Entry{}, err
	}
	
	// Iterate over the rows
	for rows.Next() {
		var e Entry
		var body, url sql.NullString
		var ancestor int64
		err = rows.Scan(&ancestor, &e.Id, &e.Title, &body, &url, &e.Created, &e.AuthorId, &e.Forum, &e.AuthorHandle, &e.Seconds, &e.Upvotes, &e.Downvotes)
		if err != nil {
			return &e, err
		}

		//Only the body or the url will be set; they are mutually exclusive
		if body.Valid {
			e.Body = body.String
		} else if url.Valid {
			e.Url = url.String
		}

		entries[e.Id] = &e
		relationships = append(relationships, map[string]int64{ "Parent": ancestor, "Child": e.Id})
	}
	
	//Construct the full Entry:
	for _, rel := range relationships {
		if rel["Parent"] == rel["Child"] {
			continue
		}
		entries[int64(rel["Parent"])].AddChild(entries[int64(rel["Child"])])
	}
	
	return entries[root], nil
}
