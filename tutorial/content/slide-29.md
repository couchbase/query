## <b>Section 3. DML Statements</b>

## DML Statements

N1QL provides DELETE, INSERT, UPDATE, and UPSERT statements. These statements allow you to create, delete, and modify the data stored in JSON documents by specifying and executing simple commands.

This feature is currently experimental in DP4. Go ahead and try out the example query on the right. Note that since the query is prefixed with the EXPLAIN keyword the actual insert operation will not be performed. 

Without the EXPLAIN operator this example would have inserted a key "baldwin" into the tutorial bucket

The syntax for the UPSERT statement is similar to the INSERT, the difference being that with the INSERT the key being inserted must not exist. 

<pre id="example">
    EXPLAIN 
        INSERT INTO tutorial (KEY, VALUE) 
            VALUES ("baldwin", {"name":"Alex Baldwin", "type":"contact"})
</pre>
