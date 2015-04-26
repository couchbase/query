## Chaining JOINs

JOIN, NEST, and UNNEST can be chained and combined, in any order, as
many times as desired.

In the example on the right, we perform an UNNEST to generate a
complete order that contains the order Ids along with the user
details. This is then JOINed with the orders from the
users_with_orders bucket.

<pre id="example">
    SELECT  u.personal_details.display_name name, s AS order_no, o.product_details  
        FROM users_with_orders u USE KEYS "Aide_48687583" 
            UNNEST u.shipped_order_history s 
                JOIN users_with_orders o ON KEYS s.order_id
</pre>
