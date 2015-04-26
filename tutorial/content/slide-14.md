## Querying primary keys 

Specific primary keys within a bucket can be queried using the USE
KEYS clause.

The query on the right used a list of keys to fetch from the tutorial
bucket.

Any expression can be used as an argument to the USE KEYS clause, as
long as it evaluates to one or more values to be used as keys.

<pre id="example">
    SELECT fname, email
        FROM tutorial 
            USE KEYS ["dave", "ian"]
</pre>
