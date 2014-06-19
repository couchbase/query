## Performing Simple Arithmetic

We can also perform basic arithmetic in our expressions.

In this example we calculate Dave's age in dog years by dividing their age by 7.

The common arithmetic operations +, -, *, / and % are supported.

Try using a different arithmetic operation.

<pre id="example">
SELECT fname, age, age/7 AS age_dog_years 
    FROM tutorial 
        WHERE fname = 'Dave'
</pre>
