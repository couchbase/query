## Pagination with LIMIT and OFFSET

Sometimes queries return a large number of results, and you want to
process them in smaller batches.  After processing a smaller batch,
you also want to skip over a batch to process the next one.

In the example on the right we specify that there should be at most 2
results.

Try adding OFFSET 4 to skip 4 results.

<pre id="example">
SELECT fname, age
    FROM tutorial 
        ORDER BY age 
            LIMIT 2
</pre>