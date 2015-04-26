## NEST

NEST performs a join across two buckets. But instead of producing an
object for each combination of left and right hand inputs, NEST
produces a single object for each left hand input, while the
corresponding right hand inputs are collected into an array and nested
as a single array-valued field in the result object.

The query on the right nests a user's orders in the result. Try
replacing NEST with JOIN in the query to see the difference between
these two operators.

Similar to JOIN, NEST also supports LEFT [ OUTER ] NEST.

<pre id="example"> 
SELECT usr.personal_details, orders
    FROM users_with_orders usr 
        USE KEYS "Elinor_33313792" 
            NEST orders_with_users orders 
               ON KEYS ARRAY s.order_id FOR s IN usr.shipped_order_history END
</pre>
