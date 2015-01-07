## UPDATE
This feature is currently experimental and not for use in production.

UPDATE replaces a document that already exists with updated values

The example on the right would have changed the type of the document "baldwin" from 
"contact" to "actor"

<pre id="example">
    EXPLAIN 
        UPDATE tutorial 
            USE KEYS "baldwin" 
                SET type = "actor" RETURNING tutorial.type
</pre>
