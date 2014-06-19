## User friendly language

If you are familiar with SQL, then picking up N1QL will be easy. 

A simple query in N1QL has three parts to it:

* <b>SELECT</b> - Parts of document to return
* <b>FROM</b> - The data bucket, or data store to work with
* <b>WHERE</b> - Conditions the document must satisfy

Only a SELECT clause is required in a query. The wildcard * selects all parts of the document. Queries can return a collection of different document structures or fragments. However, they will all match the conditions in the WHERE clause.

Remember there **IS NO SCHEMA** in Couchbase. You don't lose any flexibility you love about Couchbase.

If you change the * to a document field such as 'children', you will see the query return a collection of appropriate fragments of each document.

Try the next sample query where we find all documents where the name is 'ian.'

<pre id="example">
SELECT *
  FROM tutorial
    WHERE fname = 'Ian'
</pre>
