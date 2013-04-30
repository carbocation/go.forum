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
	"fmt"

	"github.com/carbocation/go.util/datatypes/closuretable"
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
	AuthorHandle string //Name of the author
	Points       int64  //Number of points the post has
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
	if err != nil{
		tx.Rollback()
		return errors.New("Error: We couldn't save the relationship between your comment and its parent comment.")
	}

	tx.Commit()

	return nil
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
	err = stmt.QueryRow(id).Scan(&e.Id, &e.Title, &body, &url, &e.Created, &e.AuthorId, &e.Forum, &e.AuthorHandle)
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
func DescendantEntries(root int64) (entries map[int64]Entry, err error) {
	entries, err = getEntries(root, "AllDescendants")
	return
}

// Retrieves entries that are immediate descendants of the ancestral entry, including the ancestral entry itself
func DepthOneDescendantEntries(root int64) (entries map[int64]Entry, err error) {
	entries, err = getEntries(root, "DepthOneDescendants")
	return
}

func getEntries(root int64, flag string) (entries map[int64]Entry, err error) {
	entries = map[int64]Entry{}

	var stmt *sql.Stmt

	switch flag {
	case "AllDescendants":
		stmt, err = Config.DB.Prepare(queries.DescendantEntries)
	case "DepthOneDescendants":
		stmt, err = Config.DB.Prepare(queries.DepthOneDescendantEntries)
	}
	if err != nil {
		return
	}
	defer stmt.Close()

	// Query from that prepared statement
	rows, err := stmt.Query(root)
	if err != nil {
		return
	}

	// Iterate over the rows
	for rows.Next() {
		var e Entry
		var body, url sql.NullString
		err = rows.Scan(&e.Id, &e.Title, &body, &url, &e.Created, &e.AuthorId, &e.Forum, &e.AuthorHandle)
		if err != nil {
			return
		}

		//Only the body or the url will be set; they are mutually exclusive
		if body.Valid {
			e.Body = body.String
		} else if url.Valid {
			e.Url = url.String
		}

		entries[e.Id] = e
	}

	return
}

//Returns a closure table of IDs that are descendants of a given ID (or the ID itself)
func ClosureTable(id int64) (ct *closuretable.ClosureTable, err error) {
	ct, err = getClosureTable(id, "AllDescendants")
	if err != nil {
		err = errors.New("forum: Error in AllDescendants: "+err.Error()+" ID was " + fmt.Sprintf("%d", id))
	}
	return
}

//Returns a closure table keeping only IDs that are direct descendants of a given ID (or the ID itself)
func DepthOneClosureTable(id int64) (ct *closuretable.ClosureTable, err error) {
	ct, err = getClosureTable(id, "DepthOneDescendants")
	if err != nil {
		err = errors.New("forum: Error in DepthOneDescendants: "+err.Error())
	}
	return
}

func getClosureTable(id int64, flag string) (ct *closuretable.ClosureTable, err error) {
	ct = new(closuretable.ClosureTable)

	var stmt *sql.Stmt

	// Pull down the remaining elements in the closure table that are descendants of this node
	switch flag {
	case "AllDescendants":
		stmt, err = Config.DB.Prepare(queries.DescendantClosureTable)
	case "DepthOneDescendants":
		stmt, err = Config.DB.Prepare(queries.DepthOneClosureTable)
	}
	if err != nil {
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query(id)
	if err != nil {
		//fmt.Printf("Query Error: %v", err)
		return
	}

	//Populate the closuretable
	rel := new(closuretable.Relationship)
	for rows.Next() {
		err = rows.Scan(&rel.Ancestor, &rel.Descendant, &rel.Depth)
		if err != nil {
			//fmt.Printf("Rowscan error: %s\n", err)
			return
		}
		
		err = ct.AddRelationship(*rel)
		if err != nil {
			//fmt.Printf("AddRelationship error: %s\n", err)
			return
		}
	}

	return
}
