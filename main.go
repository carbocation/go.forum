/*
To be initialized, the forum expects a live Postgres database connection handle to
be passed in. For example:

var db *sql.DB

func main() {
	db = (...get the object...)

	forum.Initialize(db)
}
*/
package forum

import (
	"database/sql"
)

type conf struct {
	DB *sql.DB //A live database object
}

//Create a package-global config object holding needed globals
var Config *conf = &conf{}

//Niladic function to setup the forum
func Initialize(db *sql.DB) {
	Config.DB = db
}
