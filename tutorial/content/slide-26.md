## NEST clause 

Nesting is conceptually the inverse of unnesting. Nesting performs a join across two buckets. But instead of producing a cross-product of the left and right hand inputs, a single result is produced for each left hand input, while the corresponding right hand inputs are collected into an array and nested as a single array-valued field in the result object.

The query on the right nests a users order descriptions in the result. Try replacing the NEST keyword with JOIN in the query to see the difference between these two types of JOIN operators. 

Similar to the JOIN clause, the NEST clause also supports LEFT joins

<pre id="example"> 
SELECT user.personal_details, orders
    FROM users_with_orders user 
        KEY "Elinor_33313792" 
            NEST orders_with_users orders 
                KEYS ARRAY s.order_id FOR s IN user.shipped_order_history END
</pre>
