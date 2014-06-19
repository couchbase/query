## Document Meta-Data

Document databases such as Couchbase often store meta-data about a document outside of the document.

In the example on the right, the META() function is used to access the meta-data for each document.  In the tutorial database the only meta-data field returned is the document ID.

<pre id="example">
SELECT META() AS meta
	FROM tutorial
</pre>
