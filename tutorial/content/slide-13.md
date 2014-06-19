## Combining Multiple Conditions with AND

The AND operator allows us to match documents satisfying two or more conditions.

In the example on the right we only return people having at least one child and having a gmail address.

Try changing AND to OR.

<pre id="example">
SELECT fname, email, children
    FROM tutorial 
        WHERE LENGTH(children) > 0 AND email LIKE '%@gmail.com'
</pre>