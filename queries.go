/*
This file manages all SQL queries that are made in the forum package.
*/
package forum

var queries = struct {
	DescendantEntriesChildParent         string //Entry itself and all descendents, only pulling self- and child-parent relationships
	AncestorEntriesChildParent           string //Entry itself and all ancestors, only pulling self- and child-parent relationships
	DepthOneDescendantEntriesChildParent string //Entry itself and all immediate descendents, only pulling self- and child-parent relationships
	OneEntry                             string //Retrieve one entry alone
	EntryCreate                          string //Create a new entry
	EntryClosureTableCreate              string //Create all closure table entries for the new entry
	VoteUpsert                           string //Upsert a vote
	FindVote                             string //Retrieve a vote by userId and entryId
}{
	DescendantEntriesChildParent: `select ancestor, e.id, e.title, e.body, e.url, e.created, e.author_id, e.forum, a.handle, extract(epoch from (now()-e.created)) seconds, COALESCE(v.upvotes, 0) upvotes, COALESCE(v.downvotes, 0) downvotes, COALESCE(vu.upvote::int,0) uupvote, COALESCE(vu.downvote::int,0) udownvote 
from entry e
join entry_closures ec ON (
	e.id=ec.descendant
	AND ec.descendant IN (
		-- Descendant is a descendant of a depth-1 descendant of the primary ancestor
		select descendant
		from entry_closures
		where ancestor IN
		(
			-- Descendant is a depth-1 descendant of the primary ancestor
			select descendant
			from entry_closures
			where ancestor=$1
			AND depth=1
			ORDER BY descendant DESC
			LIMIT 2000 offset 200*0
		)
		OR (ancestor=descendant AND ancestor=$1)
	) 
	and ancestor in (
		select descendant
		from entry_closures
		where ancestor=$1
	)
	and (
		(ec.ancestor=$1 AND ec.descendant=$1)
		OR ec.depth=1
	)
)
join account a ON a.id=e.author_id
left join (
	select entry_id, SUM(upvote::int) upvotes, SUM(downvote::int) downvotes 
	from vote
	group by entry_id
) v ON v.entry_id=e.id
left join vote vu on (
	vu.entry_id=e.id
	AND vu.user_id=$2
)`,
	AncestorEntriesChildParent: `select descendant, e.id, e.title, e.body, e.url, e.created, e.author_id, e.forum, a.handle, extract(epoch from (now()-e.created)) seconds, COALESCE(v.upvotes, 0) upvotes, COALESCE(v.downvotes, 0) downvotes, COALESCE(vu.upvote::int,0) uupvote, COALESCE(vu.downvote::int,0) udownvote 
from entry e
join entry_closures ec ON (
	e.id=ec.ancestor
	AND ec.ancestor IN (
		-- Descendant is a descendant of a depth-1 descendant of the primary ancestor
		select ancestor
		from entry_closures
		where descendant IN
		(
			-- Descendant is a depth-1 descendant of the primary ancestor
			select ancestor
			from entry_closures
			where descendant=$1
			AND depth=1
			ORDER BY ancestor DESC
			LIMIT 2000 offset 200*0
		)
		OR (ancestor=descendant AND ancestor=$1)
	) 
	and descendant in (
		select ancestor
		from entry_closures
		where descendant=$1
	)
	and (
		(ec.descendant=$1 AND ec.ancestor=$1)
		OR ec.depth=1
	)
)
join account a ON a.id=e.author_id
left join (
	select entry_id, SUM(upvote::int) upvotes, SUM(downvote::int) downvotes 
	from vote
	group by entry_id
) v ON v.entry_id=e.id
left join vote vu on (
	vu.entry_id=e.id
	AND vu.user_id=$2
)`,
	DepthOneDescendantEntriesChildParent: `select ancestor, e.id, e.title, e.body, e.url, e.created, e.author_id, e.forum, a.handle, extract(epoch from (now()-e.created)) seconds, COALESCE(v.upvotes, 0) upvotes, COALESCE(v.downvotes, 0) downvotes, COALESCE(vu.upvote::int,0) uupvote, COALESCE(vu.downvote::int,0) udownvote 
from entry_closures closure
join entry e ON e.id = closure.descendant
join account a ON a.id=e.author_id
left join (
	select entry_id, SUM(upvote::int) upvotes, SUM(downvote::int) downvotes 
	from vote
	group by entry_id
) v ON v.entry_id=e.id
left join vote vu on (
	vu.entry_id=e.id
	AND vu.user_id=$2
)
where 1=1
AND closure.ancestor = $1
AND (closure.depth=1 OR closure.depth=0)`,
	OneEntry: `SELECT e.id, e.title, e.body, e.url, e.created, e.author_id, e.forum, a.handle, extract(epoch from (now()-e.created)) seconds, COALESCE(v.upvotes, 0), COALESCE(v.downvotes, 0)
FROM entry e
JOIN account a ON a.id=e.author_id
LEFT JOIN (
	SELECT entry_id, SUM(upvote::int) upvotes, SUM(downvote::int) downvotes 
	FROM vote
	GROUP BY entry_id
) v ON v.entry_id=e.id
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
	VoteUpsert: `WITH new_values (user_id, entry_id, upvote, downvote) as (
  values 
     ($2::int, $1::int, $3::bool, $4::bool)
),
upsert as
( 
    update vote m 
        set upvote = nv.upvote, downvote = nv.downvote
    FROM new_values nv
    WHERE m.user_id = nv.user_id AND m.entry_id = nv.entry_id
    RETURNING m.*
)
INSERT INTO vote (user_id, entry_id, upvote, downvote)
SELECT user_id, entry_id, upvote, downvote
FROM new_values
WHERE NOT EXISTS (SELECT 1 
	FROM upsert up 
	WHERE up.user_id = new_values.user_id AND up.entry_id = new_values.entry_id)`,
	FindVote: `SELECT entry_id, user_id, upvote, downvote, created FROM vote WHERE entry_id=$1 and user_id=$2`,
}
