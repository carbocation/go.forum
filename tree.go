package main

import (
	"fmt"
	"time"
)

// Tree is an element in the tree.
type Tree interface {
	// Parent, left, and right pointers or nil.
	Child() *Tree
	Sibling() *Tree
	Parent() *Tree
	AddChild(*Tree) //Add a child
	Points() int64 //Return the node's score (on whatever metric) 
	
}

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
	
	Tree //An Entry has Tree-ness
	parent, child, sibling *Entry //Mandatory pointer-holders for Tree-ness
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

func (e *Entry) Walk() {
	if e == nil {
		return
	}
	
	if e.parent == nil {
		fmt.Printf("\n\nWalking...\n(Root) ")
	}
	fmt.Printf("%p: %+v\n", e, e)
	e.Child().Walk()
	e.Sibling().Walk()
}

func main () {
	t := &Entry{Title: "Root", Upvotes: 0}
	t.AddChild(&Entry{Title: "Depth 1 #4", Upvotes: 0})
	t.AddChild(&Entry{Title: "Depth 1 #3", Upvotes: 1})
	
	
	t2 := &Entry{Title: "Depth 1 #2", Upvotes: 2}
	t2.AddChild(&Entry{Title: "Depth 2 #2", Upvotes: 9})
	t2.AddChild(&Entry{Title: "Depth 2 #1", Upvotes: 10})
	//t2.sibling, t2.child  = t2.child, nil
	//t2.sibling.parent = t2
	
	t.AddChild(t2)
	
	t.AddChild(&Entry{Title: "Depth 1 #1", Upvotes: 3})
	
	
	t.Walk()
}
