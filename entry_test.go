package forum

import (
	"testing"
)

func makeUnsortedTree() *Entry {
	x := &Entry{Title: "Root", Upvotes: 0}
	x.AddChild(&Entry{Title: "Depth 1 #4", Upvotes: 0})
	x.AddChild(&Entry{Title: "Depth 1 #3", Upvotes: 1})

	x2 := &Entry{Title: "Depth 1 #1", Upvotes: 2}
	x2.AddChild(&Entry{Title: "Depth 2 #2", Upvotes: 9})
	x2.AddChild(&Entry{Title: "Depth 2 #1", Upvotes: 10})

	x.AddChild(x2)

	x.AddChild(&Entry{Title: "Depth 1 #2", Upvotes: 3})
	
	return x
}

func BenchmarkTree(b *testing.B) {
	x := makeUnsortedTree()
	
	for i := 0; i < b.N; i++ {
		x.AddChild(&Entry{})
	}
	
	x = Arrange(x)
	
}

func TestTree(t *testing.T) {
	x := makeUnsortedTree()
	
	x = Arrange(x)

	output := string(walk(x))

	expected := "Root:Depth 1 #1:Depth 2 #1:Depth 2 #2:Depth 1 #2:Depth 1 #3:Depth 1 #4:"

	if output != expected {
		t.Errorf("Got %s, expected %s", output, expected)
	}
}

func walk(e *Entry) []byte {
	output := make([]byte, 0)

	if e == nil {
		return output
	}

	if e.Parent() == nil {
		//fmt.Printf("\n\nWalking...\n(Root) ")
	}
	//fmt.Printf("%p: %+v\n", e, e)

	output = append(output, []byte(e.Title+":")...)
	output = append(output, walk(e.Child())...)
	output = append(output, walk(e.Sibling())...)

	return output
}
