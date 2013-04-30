/*
This file manages all SQL queries that are made in the forum package.
*/
package forum

var queries = struct {
	DescendantEntries         string //Entry itself and all descendents
	DescendantClosureTable    string
	DepthOneDescendantEntries string //Entry itself and all immediate descendents
	DepthOneClosureTable      string
	OneEntry                  string //Retrieve one entry alone
	EntryCreate               string //Create a new entry
	EntryClosureTableCreate   string //Create all closure table entries for the new entry
}{
	DescendantEntries: `SELECT e.id, e.title, e.body, e.url, e.created, e.author_id, e.forum, a.handle
FROM entry_closures closure
JOIN entry e ON e.id = closure.descendant
JOIN account a ON a.id=e.author_id
WHERE closure.ancestor = $1`,
	DescendantClosureTable: `select * 
from entry_closures
where descendant in (
select descendant
from entry_closures
where ancestor=$1
)
and ancestor in (
select descendant
from entry_closures
where ancestor=$1
)
and depth = 1`,
	DepthOneDescendantEntries: `SELECT e.id, e.title, e.body, e.url, e.created, e.author_id, e.forum, a.handle
FROM entry_closures closure
JOIN entry e ON e.id = closure.descendant
JOIN account a ON a.id=e.author_id
WHERE 1=1
AND closure.ancestor = $1
AND (closure.depth=1 OR closure.depth=0)`,
	DepthOneClosureTable: `select * 
from entry_closures
where ancestor=$1
and depth=1`,
	OneEntry: `SELECT e.id, e.title, e.body, e.url, e.created, e.author_id, e.forum, a.handle
FROM entry e
JOIN account a ON a.id=e.author_id
WHERE 1=1
AND e.id=$1
`,
	EntryCreate: `INSERT INTO entry (title, body, url, author_id) VALUES ($1, $2, $3, $4) RETURNING id`,
	EntryClosureTableCreate: `INSERT INTO entry_closures
	select cast($1 as bigint) newancestor, cast($1 as bigint) newdescendant, 0 newdepth
	union 
	select e.ancestor newancestor, cast($1 as bigint) newdescendant, e.depth+1 newdepth
	from entry_closures e
	where e.descendant = $2
	order by newdepth asc
`,
}
