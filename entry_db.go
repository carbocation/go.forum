/*
Entry methods and functions that access a database are placed here.
*/
package forum

import (
	"database/sql"
	"errors"
	"strings"
)

// Stores an entry to the database and correctly builds its ancestry based
// on its parent's ID.
func (e *Entry) Persist(parentId int64) error {
	//Trim
	e.Title = strings.TrimSpace(e.Title)
	e.Body = strings.TrimSpace(e.Body)

	//Validate
	if e.Body == "" {
		return errors.New("The Body must not be empty or consist solely of whitespace.")
	}

	//Wrap in a transaction
	tx, err := Config.DB.Begin()

	EntryCreateStmt, err := tx.Prepare(queries.EntryCreate)
	defer EntryCreateStmt.Close()
	if err != nil {
		_ = tx.Rollback()
		return errors.New("Error: We had a database problem trying to create your entry.")
	}

	//Note: because pq handles LastInsertId oddly (or not at all?), instead of
	//calling .Exec() then .LastInsertId, we prepare a statement that ends in
	//`RETURNING id` and we .QueryRow().Select() the result
	err = EntryCreateStmt.QueryRow(e.Title, e.Body, e.Url, e.AuthorId).Scan(&e.Id)
	defer EntryCreateStmt.Close()
	if err != nil {
		tx.Rollback()
		return errors.New("Error: your username or email address was already found in the database. Please choose differently.")
	}

	EntryClosureTableCreateStmt, err := tx.Prepare(queries.EntryClosureTableCreate)
	defer EntryClosureTableCreateStmt.Close()
	if err != nil {
		tx.Rollback()
		// return err
		return errors.New("Error: We had a database problem trying to create ancestry information.")
	}

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
	defer stmt.Close()
	if err != nil {
		return e, err
	}

	err = stmt.QueryRow(id).Scan(&e.Id, &e.Title, &e.Body, &e.Url, &e.Created, &e.AuthorId, &e.Forum, &e.AuthorHandle, &e.Seconds, &e.Upvotes, &e.Downvotes)
	if err != nil {
		e = new(Entry)
		return e, err
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
		defer stmt.Close()
	case "DepthOneDescendants":
		stmt, err = Config.DB.Prepare(queries.DepthOneDescendantEntriesChildParent)
		defer stmt.Close()
	}
	if err != nil {
		return New(), err
	}

	// Query from that prepared statement
	rows, err := stmt.Query(root, user.GetId())
	if err != nil {
		return New(), err
	}

	// Iterate over the rows
	for rows.Next() {
		var e *Entry = New()
		var ancestor int64
		err = rows.Scan(&ancestor, &e.Id, &e.Title, &e.Body, &e.Url, &e.Created, &e.AuthorId, &e.Forum, &e.AuthorHandle, &e.Seconds, &e.Upvotes, &e.Downvotes, &e.UserVote.Upvote, &e.UserVote.Downvote)
		if err != nil {
			return e, err
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

	return Arrange(entries[root]), nil
}
