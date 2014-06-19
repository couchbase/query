## Select DISTINCT 

The DISTINCT keyword can be used to eliminate dulicates from the output. 

The query on the right used the DISTINCT keyword in the SELECT statement to produce a set of 3 unique results. 

Try removing the DISTINCT keyword from the query to see the difference. 

<pre id="example">
    SELECT DISTINCT orderlines[0].productId
        FROM orders
</pre>
