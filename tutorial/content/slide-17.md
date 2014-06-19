## Ordering results with ORDER BY clause

Queries can optionally include an ORDER BY clause describing how the results should be sorted.

In the example on the right we ask that the people be listed by age in ascending order.

Try adding DESC after age.

<pre id="example">
SELECT fname, age 
    FROM tutorial 
        ORDER BY age
</pre>
