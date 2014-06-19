## Review

The real power of the language comes through when we combine all the elements together.

The query on the right illustrates many of the previous concepts working together.

1.  First we start with all 6 documents FROM the tutorial bucket.
2.  The WHERE clause eliminates children 10 years old or younger
3.  Next the GROUP BY forms 3 groups, one for each relation ("friend", "parent", "cousin")
4.  Then the HAVING clause removes group "parent" (only has 1 member)
5.  Next the groups are ordered by the average age of the group members descending
6.  The we skip over one value in the output and limit the result to a single value.
7.  Finally the expressions in the SELECT clause are projected, showing the grouping criteria (relation), the count of items in the group, and average age of the group members


<pre id="example"> 
SELECT t.relation, count(*) as count, avg(c.age) as avg_age
    FROM tutorial t
    UNNEST t.children c
    WHERE c.age > 10
    GROUP BY t.relation
    HAVING count(*) > 1
    ORDER BY avg_age DESC
    LIMIT 1 OFFSET 1
</pre>
