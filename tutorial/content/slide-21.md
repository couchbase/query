## Filtering grouped data with HAVING

Sometimes you want to filter which groups are returned.

Similar to filtering documents with the WHERE clause, we can filter
groups with the HAVING clause.

Here we filter to only include groups with more than 1 member.

<pre id="example">
SELECT relation, COUNT(*) AS count
    FROM tutorial
        GROUP BY relation
        	HAVING COUNT(*) > 1
</pre>