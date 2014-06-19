## UNNEST clause

The UNNEST clause allow us to take the contents of nested arrays and join them with the parent object.

Some people in the tutorial database have an array of children.  If we had 3 people, each with 2 children, we would get 6 documents out, each containing 1 person and 1 child.

The query on the right joins Dave with each of his 2 children.

<pre id="example">
SELECT * 
    FROM tutorial AS contact
    	UNNEST contact.children
        	WHERE contact.fname = 'Dave' 
</pre>
