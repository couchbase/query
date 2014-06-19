## Documents, not rows

Data in Couchbase Server is stored in the form of documents, not rows or columns.

As documents can have nested elements and embedded arrays, a few additional operators are needed. The '.' operator is used to refer to children, and the '[]' is used to refer to an element in an array. You can use a combination of these operators to access data at any depth in a document.

In the example on the right, the document in the tutorial bucket has an embedded 'children' array. We fetch the first child's name. This query deals with two distinct 'name' attributes - the name attribute from the top level document object, and the name attribute from the child object. In general, an identifier like 'name' 
always refers to an attribute in the parent document or an alias. Attributes from child documents must be explicitly aliased using the **AS** clause. Here, the child's name is aliased to 'cname'.

Try removing the 'AS cname' clause and see what happens.

<pre id="example">
SELECT children[0].fname AS cname
	FROM tutorial
       WHERE fname='Dave'
</pre>
