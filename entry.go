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
	Id       int64     //The ID of the post
	Title    string    //Title of the post. Will be empty for entries that are really intended to be comments.
	Body     string    //Contents of the post. Will be empty for entries that are intended to be links.
	Created  time.Time //Time at which the post was created.
	AuthorId int64     `schema:"-"` //ID of the author of the post
	Forum    bool      `schema:"-"` //Is this Entry actually a forum instead?
	Url      bool      `schema:"-"` //Is this Entry just a link?

	//Fields beneath this line are not persisted to the Entry table

	AuthorHandle string  //Name of the author
	Seconds      float64 //Seconds since creation
	Upvotes      int64
	Downvotes    int64

	UserVote *Vote //A Vote representing how the current user has voted on this Entry

	parent, child, sibling *Entry //Mandatory pointer-holders for Tree-ness
}

func (e *Entry) Child() *Entry        { return e.child }
func (e *Entry) Sibling() *Entry      { return e.sibling }
func (e *Entry) Parent() *Entry       { return e.parent }
func (e *Entry) Less(cmp *Entry) bool { return e.Score() < cmp.Score() }

//Return an ordered *Entry tree
//Order among siblings is determined by Score
//Score is determined recursively, with all Child (and Child's Siblings, their children, etc)
// contributing to the score
func Arrange(e *Entry) *Entry {
	if e == nil {
		return nil
	}

	//Continue through all child nodes to ensure everything gets sorted
	if e.Child() != nil {
		e.child = Arrange(e.Child())
	}

	//Continue through all sibling nodes to ensure everything gets sorted
	if e.Sibling() != nil {
		e.sibling = Arrange(e.Sibling())
	}

	//If we have a sibling, and if we are not merely a sibling ourselves, mergesort this like a linked list
	if e.Sibling() != nil && (e.Parent() == nil || e.Parent().Sibling() != e) {
		//Current node is root or its parent is a true parent
		e = mergeSort(e)
	}

	return e
}

//Do a mergeSort to put the siblings in order
//Based on Java code from http://www.dontforgettothink.com/2011/11/23/merge-sort-of-linked-list/
func mergeSort(e *Entry) *Entry {
	if e == nil || e.Sibling() == nil {
		//Not even a list, or is a list of exactly one
		return e
	}

	//Get the middle node in the list, then designate the node right after that
	// as the first of a new list.
	var middle *Entry = e.getMiddle()
	var sHalf *Entry = middle.Sibling()

	//Unlink the two lists.
	middle.sibling, sHalf.parent = nil, nil

	return merge(mergeSort(e), mergeSort(sHalf))
}

//Find the middle entry among a list of siblings
//Required for mergeSort
func (e *Entry) getMiddle() *Entry {
	if e == nil {
		return e
	}

	var slow, fast *Entry
	slow, fast = e, e

	for fast.Sibling() != nil && fast.Sibling().Sibling() != nil {
		slow, fast = slow.Sibling(), fast.Sibling().Sibling()
	}

	return slow
}

//Do the merge step of mergeSort, using the Less() method to sort siblings
func merge(a, b *Entry) *Entry {
	dummyHead := &Entry{}
	curr := dummyHead

	for a != nil && b != nil {
		if b.Less(a) {
			curr.sibling, a = a, a.Sibling()
			//May need to split into two lines
		} else {
			curr.sibling, b = b, b.Sibling()
		}

		curr = curr.Sibling()
	}

	if a == nil {
		curr.sibling = b
	} else {
		curr.sibling = a
	}

	return dummyHead.Sibling()
}

//Points return a user-visible indicator of Upvotes - Downvotes
func (e *Entry) Points() int64 {
	if e == nil {
		return 0
	}

	return e.Upvotes - e.Downvotes
}

//Score determines sort order and can also be shown to help explain why comments are in their given order
func (e *Entry) Score() int64 {
	if e == nil {
		return 0
	}

	if e.Child() == nil {
		return e.score()
	} else {
		//If the entry has children, all of the entry's children's (child + sibs) scores count for and against it,
		// as do their children's scores, etc.
		return e.score() + e.Child().recursiveScore()
	}
}

//The actual definition of a score can rely on anything found in Entry
func (e *Entry) score() int64 {
	if e == nil {
		return 0
	}

	return e.Upvotes - e.Downvotes
}

//Traverses both sides of the tree starting from an Entry and sums the score
func (e *Entry) recursiveScore() int64 {
	if e == nil {
		return 0
	}

	return e.score() + e.Child().recursiveScore() + e.Sibling().recursiveScore()
}

//Add a child node to the current entry
//If the current entry's child slot is full, recursively try the child's sibling(s)' slots
//until an open (nil) slot is found
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

//Add a sibling to the specified node. If the node already has a sibling, add to that sibling.
//Recurse until there is an open sibling slot.
//Note that, while this does not put the entries in the precisely correct order based on
//recursive score (because there is no guarantee that an entry's children have been associated
// with the entry yet, so the recursive calculation may well miss a good chunk of points),
// it's still better than a non-score-based approach. Why? This gives us partial ordering
func (e *Entry) addSibling(newE *Entry) {
	if newE == nil {
		return
	}

	// The new element will be inserted ABOVE the old one
	// This optimizes for the case where the new element has
	// no siblings

	// New element may have a sibling (or may be nil). We will pop it off and then add it back
	// at the end to cover our bases in case it's not nil.
	newESib := newE.Sibling()

	if e.Parent() == nil {
		// Old element was a root node, and we are directly adding a sibling to it (this should probably not be allowed)
		newE.sibling, e.parent = e, newE
	} else if e == e.Parent().child {
		// Old element was a child of its parent
		e.Parent().child, newE.parent, newE.sibling, e.parent = newE, e.Parent(), e, newE
	} else {
		// Old element was presumptively a sibling of its parent
		e.Parent().sibling, newE.parent, newE.sibling, e.parent = newE, e.Parent(), e, newE
	}

	// Add back sibling of new element (if any)
	newE.addSibling(newESib)

	return
}
