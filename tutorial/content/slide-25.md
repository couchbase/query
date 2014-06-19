## LEFT OUTER JOINS

By default, an INNER join is performed. This means that for each joined object produced, both the left and right hand source objects must be non-missing and non-null.

If LEFT or LEFT OUTER is specified, then a left outer join is performed. At least one joined object is produced for each left hand source object. If the right hand source object is NULL or MISSING, then the joined object's right-hand side value is also NULL or MISSING (omitted), respectively.

Try the example on the right and also try removing the LEFT clause to see the difference in the output. In this query user "Tamekia_13483660" has no orders so running the query without the LEFT clause will produce an empty result

<pre id="example">
    SELECT user.personal_details, orders
        FROM users_with_orders user 
            KEY "Tamekia_13483660" 
                LEFT JOIN orders_with_users orders 
                    KEYS ARRAY s.order_id FOR s IN user.shipped_order_history END
</pre> 

