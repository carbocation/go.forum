package forum

/*
Define the methods that a user object must have in order to
be compatible with this forum system.
*/

type User interface {
	GetId() int64
}
