## <b>Section 3. DML Statements</b>

## DML statements

N1QL provides UPDATE, DELETE, INSERT, UPSERT, and MERGE
statements. These statements allow you to create, delete, and modify
data.

Currently, these statements only provide document-level atomicity, so
they are not supported for use in production.

Go ahead and try out the example query on the right. Note that since
the query is prefixed with the EXPLAIN keyword the actual insert
operation will not be performed. If you are running this tutorial on
your own Couchbase installation, you can remove the EXPLAIN keyword to
perform the actual insert.

<pre id="example">
    EXPLAIN 
        INSERT INTO tutorial (KEY, VALUE) 
            VALUES ("baldwin", {"name":"Alex Baldwin", "type":"contact"})
</pre>
