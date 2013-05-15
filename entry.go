/*
Entry is the fundamental unit of a threaded discussion. An entry can represent
a forum, a post, or a comment, depending on how it is annotated. There is nothing
fundamentally distinct about these things, and their similarities (including
hierarchical nesting) are abundant.

For entry methods that manipulate the database in some way, see entry_db.go
*/
package forum

import (
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

	//Fields beneath this line are not persisted to the Entry table

	AuthorHandle string  //Name of the author
	Seconds      float64 //Seconds since creation
	Upvotes      int64
	Downvotes    int64

	UserVote *Vote //A Vote representing how the current user has voted on this Entry

	parent, child, sibling *Entry //Mandatory pointer-holders for Tree-ness
}

func (e *Entry) Child() *Entry           { return e.child }
func (e *Entry) Sibling() *Entry         { return e.sibling }
func (e *Entry) Parent() *Entry          { return e.parent }
func (e *Entry) Greater(cmp *Entry) bool { return e.Points() > cmp.Points() }

func (e *Entry) Points() int64 {
	if e == nil {
		return 0
	}

	/*
		if e.Parent() == nil || e.ParentIsForum() {
			//If the parent is a forum, or if we have no parent
			//For top-level forum entries, i.e. 'STORIES', define points
			return e.recursivePoints()
		}
	*/

	//Default (for comments)
	return e.Upvotes - e.Downvotes
}

func (e *Entry) recursivePoints() int64 {
	if e == nil {
		return 0
	}

	if e.Child() == nil {
		return (e.Upvotes - e.Downvotes) + e.Child().recursivePoints()
	} else {
		return (e.Upvotes - e.Downvotes) + e.Child().recursivePoints() + e.Child().Sibling().recursivePoints()
	}
}

func (e *Entry) ParentIsForum() bool {
	//Because of the LCRS layout of the tree, this is less trivial and merits a method
	if e.Parent() == nil {
		return false
	}

	if e.Parent().Forum {
		return true
	}

	if e.Parent().Sibling() == e {
		//Ascend the sibling tree but not the child tree
		return e.Parent().ParentIsForum()
	}

	return false
}

func (e *Entry) Sort() {
	if e == nil {
		return
	}

	if e.Child() != nil {
		//Sort the children
		e.Child().Sort()
	}

	if e.Sibling() != nil && e.Sibling().Greater(e) {
		//Swap: This node has a sibling that deserves to be above it

		if e.Parent() != nil && e.Parent().Child() == e {
			//e was a direct child of its parent
			e.parent, e.Parent().child, e.Sibling().parent = e.Sibling(), e.Sibling(), e.Parent()
		} else if e.Parent() != nil && e.Parent().Sibling() == e {
			//e was a sibling of its parent
			e.parent, e.Parent().sibling, e.Sibling().parent = e.Sibling(), e.Sibling(), e.Parent()
		} else {
			//e has no parent
			e.parent, e.Sibling().parent = e.Sibling(), nil
		}
	}

	if e.Sibling() != nil {
		//Now go on down the chain to sort the rest of the siblings
		e.Sibling().Sort()
	}

	return
}

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

	if !newE.Greater(e) {
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
