package main

import (
	"fmt"
	"time"
)

// Tree is an element in the tree.
type Tree struct {
	// Parent, left, and right pointers in the tree of elements.
	// The root of the tree has parent = nil
	parent, child, sibling *Tree
	Value Entry
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
}

func (e Entry) Points() int64 {
	return e.Upvotes - e.Downvotes
}

func New(value Entry) *Tree {
	t := &Tree{Value: value}
	
	fmt.Printf("Created %p %v\n", t, t.Value.Title)
	
	return t
}

func (e *Tree) Child() *Tree { return e.child }

func (e *Tree) Sibling() *Tree { return e.sibling }

func (e *Tree) Parent() *Tree { return e.parent }

func (e *Tree) AddChild(newE *Tree) {
	fmt.Printf("Adding %p %v as a child of %p %v\n\n", newE, newE.Value.Title, e, e.Value.Title)
	
	//Slot is available
	if e.child == nil {
		e.child, newE.parent = newE, e
		
		return
	} else {
		//Slot is unavailable. Set a sibling on the child
		//newE.parent = e.parent
		e.child.addSibling(newE)
		
		return
	}
}

func (e *Tree) addSibling(newE *Tree) {
	if newE == nil {
		e.sibling = nil
		return
	}

	fmt.Printf("%p %v's slot is full. Adding %p %v as a sibling of %p %v\n\n", e.parent, e.parent.Value.Title, newE, newE.Value.Title, e, e.Value.Title)

	if newE.Value.Points() <= e.Value.Points() {
		fmt.Printf("%p %v should be below %p %v\n\n", newE, newE.Value.Title, e, e.Value.Title)
		// The new element belongs lower in the sibling tree
		if e.sibling == nil {
			// If the new element belongs after the old one in the sibling list, 
			// and if the old element has an empty sibling slot, 
			// set the new element to be the sibling of the old one
			newE.parent, e.sibling = e, newE
			
			return
		} else {
			// The old element already has a sibling
			// Try to add the new element as a sibling of the sibling
			e.sibling.addSibling(newE)
			
			return 
		}
	} else {
		// The new element belongs above the old sibling
		fmt.Printf("%p %v should be above %p %v\n\n", newE, newE.Value.Title, e, e.Value.Title)

		if newE.sibling == nil {
			fmt.Printf("newE (%p) %v has no sibling.\n", newE, newE.Value.Title)
			
			//New element has no sibling, so old element becomes its sibling trivially
			if e == e.parent.child {
				fmt.Printf("e (%p) was a child of its parent (%p). Now newE (%p) %v will interpose.\n", e, e.parent, newE, newE.Value.Title)
				
				newE.parent, e.parent.child, newE.sibling, e.parent = e.parent, newE, e, newE
				
				return
			} else {
				fmt.Printf("e (%p) %v was a sib of its parent (%p) %v. Now newE (%p) %v will interpose.\n", e, e.Value.Title, e.parent, e.parent.Value.Title, newE, newE.Value.Title)
				
				newE.parent, e.parent.sibling, newE.sibling, e.parent = e.parent, newE, e, newE
				
				return
			}
			
		} else {
			fmt.Println("New e has a sibling.")
			//New element has a sibling. Old element becomes its sibling, but new element's old sibling needs to be added back
			newESib := newE.sibling
			
			fmt.Printf("e: %p %v\n", e, e)
			fmt.Printf("newE: %p %v\n", newE, newE)
			fmt.Printf("e.Parent: %p %v\n", e.parent, e.parent)
			fmt.Printf("newE.sibling %p %v\n", newE.sibling, newE.sibling)
			fmt.Printf("newE.parent %p %v\n", newE.parent, newE.parent)
			
			
			//New element has no sibling, so old element becomes its sibling trivially
			if e == e.parent.child {
				fmt.Printf("e (%p) was a child of its parent (%p). Now newE (%p) %v will interpose.\n", e, e.parent, newE, newE.Value.Title)
				
				newE.parent, e.parent.child, newE.sibling, e.parent = e.parent, newE, e, newE
			} else {
				fmt.Printf("e (%p) %v was a sib of its parent (%p) %v. Now newE (%p) %v will interpose.\n", e, e.Value.Title, e.parent, e.parent.Value.Title, newE, newE.Value.Title)
				
				newE.parent, e.parent.sibling, newE.sibling, e.parent = e.parent, newE, e, newE
			}
			
			newE.addSibling(newESib)
			
			return
		}
	}
}

func (e *Tree) Walk() {
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
	t := New(Entry{Title: "Root", Upvotes: 10})
	t.AddChild(New(Entry{Title: "Level 2 #1", Upvotes: 1}))
	//t.AddChild(New(Entry{Title: "Level 2 #2", Upvotes: 0}))
	//t.AddChild(New(Entry{Title: "Level 2 #3", Upvotes: 1}))
	
	
	t2 := New(Entry{Title: "Level 2 #2", Upvotes: 2})
	t2.AddChild(New(Entry{Title: "Level 3 #1", Upvotes: 2}))
	//t2.sibling, t2.child  = t2.child, nil
	//t2.sibling.parent = t2
	
	t.AddChild(t2)
	
	t.AddChild(New(Entry{Title: "Level 2 #3", Upvotes: 3}))
	
	
	t.Walk()
}
