## Pagination with LIMIT/OFFSET

Sometimes queries return a large number of results and it can be helpful to process them in smaller batches.  After processing a smaller batch, you also want to skip over a batch to process the next one.

In the example on the right we ask that it return no more than 2 results.

Try adding OFFSET 2 to get the next 2 results.

<pre id="example">
SELECT fname, age
    FROM tutorial 
        ORDER BY age 
            LIMIT 2
</pre>