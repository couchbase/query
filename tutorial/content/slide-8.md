## Filtering Documents with WHERE

In previous slides you used a WHERE clause like WHERE name = 'dave' to match a single document.  Other comparison operators can be used to match multiple documents.

In the query example on the right we match documents where the person's age is greater than 30.

All of the standard comparison operators are supported (>, >=, <, <=, =, and !=).  All of these comparison operators also consider the value's type, so `score > 8` will return documents containing a numeric score that is greater than 8.  Similarly, `name > 'marty'` will return documents containing a string name that is after 'marty'.

Try changing the comparison from > to <.

<pre id="example">
SELECT fname, age 
    FROM tutorial
        WHERE age > 30
</pre>