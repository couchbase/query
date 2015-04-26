## Aggregate functions

Sometimes you want information about groups of data, rather than
individual items.

In the example on the right we use the COUNT() function to tell us how
many documents are in the bucket.

<pre id="example">
SELECT COUNT(*) AS count
    FROM tutorial 
</pre>