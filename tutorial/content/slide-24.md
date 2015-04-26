## <b>Section 2. Joins</b>

## JOIN clause

N1QL provides joins, which allow you to assemble new objects by
combining two or more source objects.

For example, suppose there are two buckets, users_with_orders and
orders_with_users. The bucket users_with_orders contains user profiles
along with the order Ids of the orders that they placed.

The bucket orders_with_users contains the description of a particular
order placed by a user. You can use the JOIN clause to view a
user-profile along with the orders that she has placed over time.

The example on the right combines a users profile document having the
key "Elinor_33313792", with the orders that the user has placed. In
N1QL, joins must use the ON KEYS clause. In this example, we get a
list of orders by unrolling the shipped_order_history array and using
that as input to the JOIN clause.

<pre id="example">
    SELECT usr.personal_details, orders 
        FROM users_with_orders usr 
            USE KEYS "Elinor_33313792" 
                JOIN orders_with_users orders 
                    ON KEYS ARRAY s.order_id FOR s IN usr.shipped_order_history END
</pre> 
