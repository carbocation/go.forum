/*
Entry methods and functions that access a database are placed here.
*/
package forum

import (
	"database/sql"
	"errors"
)

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
func DescendantEntries(root int64, user User) (*Entry, error) {
	return getEntries(root, "AllDescendants", user)
}

// Retrieves entries that are immediate descendants of the ancestral entry, including the ancestral entry itself
func DepthOneDescendantEntries(root int64, user User) (*Entry, error) {
	return getEntries(root, "DepthOneDescendants", user)
}

func getEntries(root int64, flag string, user User) (*Entry, error) {
	// Store output in a map initially. Get it all in here before you try to build the tree.
	entries := map[int64]*Entry{}                //k: id => v: Entry
	relationships := make([]map[string]int64, 0) //A slice of maps with k: parentId in entries map => v: childId in entries map

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
	rows, err := stmt.Query(root, user.GetId())
	if err != nil {
		return &Entry{}, err
	}

	// Iterate over the rows
	for rows.Next() {
		var e *Entry = &Entry{UserVote: &Vote{}}
		var body, url sql.NullString
		var ancestor int64
		err = rows.Scan(&ancestor, &e.Id, &e.Title, &body, &url, &e.Created, &e.AuthorId, &e.Forum, &e.AuthorHandle, &e.Seconds, &e.Upvotes, &e.Downvotes, &e.UserVote.Upvote, &e.UserVote.Downvote)
		if err != nil {
			return e, err
		}

		//Only the body or the url will be set; they are mutually exclusive
		if body.Valid {
			e.Body = body.String
		} else if url.Valid {
			e.Url = url.String
		}

		entries[e.Id] = e
		relationships = append(relationships, map[string]int64{"Parent": ancestor, "Child": e.Id})
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
