## String concatenation

The string concatenation operator allows you to combine multiple
string values into one.

In the example on the right we combine first and last names into full
names.

Try adding the field "title" to the output as well.

<pre id="example">
SELECT fname || " " || lname AS full_name
    FROM tutorial 
</pre>