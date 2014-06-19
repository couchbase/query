## Querying primary keys 

Specific primary keys within a bucket can be queried using the KEYS clause. The argument should be an array.

The query on the right fetches a list of keys from the bucket tutorial. 

An arbitary expression can be used as an argument to the KEYS clause as long as it evaluates to an array or a single element

<pre id="example">
    SELECT fname, email
        FROM tutorial 
            KEYS ["dave", "ian"]
</pre>
