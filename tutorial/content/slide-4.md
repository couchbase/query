## Document meta-data

Document databases such as Couchbase often store meta-data about a
document outside of the document.

In the example on the right, the META() function is used to access the
meta-data for each document.  In the tutorial database the only
meta-data field returned is the document ID.

<pre id="example">
SELECT META(tutorial) AS meta
	FROM tutorial
</pre>
