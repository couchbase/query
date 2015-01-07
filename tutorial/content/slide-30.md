## DELETE
This feature is currently experimental and not for use in production.

Keys can be deleted using the DELETE clause. The returning clause in the example query on the right will return the list of keys that were deleted from the bucket

The example below will delete all documents where tutorial.title = "Mrs"

<span style="color: red">
EXPLAIN DELETE FROM tutorial t WHERE t.title = "Mrs"
</span>

<pre id="example">
    EXPLAIN DELETE FROM tutorial t 
        USE KEYS "baldwin" RETURNING t
</pre>
