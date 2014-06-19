## Use Functions on the Data

Built-in functions allow greater flexibility when working with the data.

The ROUND() and TRUNC() functions allow us to round and truncate numeric values.

In the example on the right, we've updated the previous example to round the dog_years calculation to an integer.

Try changing ROUND() to TRUNC()

<pre id="example">
SELECT fname, age, ROUND(age/7) AS age_dog_years 
    FROM tutorial 
        WHERE fname = 'Dave'
</pre>