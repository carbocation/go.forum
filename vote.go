/*
Entry is the fundamental unit of a threaded discussion. An entry can represent
a forum, a post, or a comment, depending on how it is annotated. There is nothing
fundamentally distinct about these things, and their similarities (including
hierarchical nesting) are abundant.
*/
package forum

import (
	"errors"
	"fmt"
	"time"
)

type Vote struct {
	//Id       int64     //The ID of this vote
	EntryId  int64     //The ID of the post
	UserId   int64     //The ID of the user who cast this vote
	Upvote   bool      //Is this an upvote?
	Downvote bool      //Is this a downvote?
	Created  time.Time //Time at which the vote was cast
}

// Stores an entry to the database and correctly builds its ancestry based
// on its parent's ID.
func (v *Vote) Persist() error {
	//Wrap in a transaction
	tx, err := Config.DB.Begin()

	VoteCreateStmt, err := tx.Prepare(queries.VoteUpsert)
	if err != nil {
		_ = tx.Rollback()
		fmt.Println(err)
		return errors.New("Error: We had a database problem trying to create your vote.")
	}
	defer VoteCreateStmt.Close()

	_, err = VoteCreateStmt.Exec(v.EntryId, v.UserId, v.Upvote, v.Downvote)
	if err != nil {
		tx.Rollback()
		fmt.Println(err)
		return errors.New("Error: Your vote could not be stored.")
	}

	tx.Commit()

	return nil
}

//Retrieve one vote based on entry ID and user ID.
func FindVote(entryId, userId int64) (*Vote, bool) {
	v := new(Vote)

	stmt, err := Config.DB.Prepare(queries.FindVote)
	if err != nil {
		return v, false
	}
	defer stmt.Close()

	err = stmt.QueryRow(entryId, userId).Scan(&v.EntryId, &v.UserId, &v.Upvote, &v.Downvote, &v.Created)
	if err != nil {
		v = new(Vote)
		return v, false
	}

	return v, true
}
