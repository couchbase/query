## UPDATE

UPDATE modifies an existing document.

Currently, this statement only provides document-level atomicity, so
it is not supported for use in production.

The example on the right would change the type of the document
"baldwin" from "contact" to "actor".

<pre id="example">
    EXPLAIN 
        UPDATE tutorial 
            USE KEYS "baldwin" 
                SET type = "actor" RETURNING tutorial.type
</pre>
