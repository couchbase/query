## DELETE

Documents can be deleted using the DELETE clause. The RETURNING clause
in the example query on the right will return the list of keys that
were deleted from the bucket.

Currently, this statement only provides document-level atomicity, so
it is not supported for use in production.

The example below would delete all documents where tutorial.title =
"Mrs".

<span style="color: red">
EXPLAIN DELETE FROM tutorial t WHERE t.title = "Mrs"
</span>

<pre id="example">
    EXPLAIN DELETE FROM tutorial t 
        USE KEYS "baldwin" RETURNING t
</pre>
