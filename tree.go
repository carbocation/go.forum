package forum

// Tree is an element in the tree.
type Tree interface {
	// Parent, left, and right pointers or nil.
	Child() *Tree
	Sibling() *Tree
	Parent() *Tree
	AddChild(*Tree) //Add a child
	Points() int64 //Return the node's score (on whatever metric) 
	
}
