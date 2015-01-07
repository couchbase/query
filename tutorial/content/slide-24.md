## <b>Section 2. Joins</b>

## JOIN clause

N1QL supports the JOIN clause that allows you to create new input objects by combining two or more source objects. 

For example, let us assume that there are two buckets users_with_orders and orders_with_users. The bucket users_with_orders contains user profiles along with the orders Ids of the orders that they placed. 

The bucket orders_with_users contains the description of a particular order placed by a user. Now in order to view a user-profile along with the orders that she has placed in the past we can use the JOIN clause to accomplish that. 

The example on the right combines a users profile document referenced by the key "Elinor_33313792", with the description of the orders that the user has placed. Note that joins must use the ON KEYS clause. In the example on the right we get a list of orders by unrolling the shipped_order_history array and use that as input to the JOIN clause to combine users personal details with the order descriptions.

<pre id="example">
    SELECT usr.personal_details, orders 
        FROM users_with_orders usr 
            USE KEYS "Elinor_33313792" 
                JOIN orders_with_users orders 
                    ON KEYS ARRAY s.order_id FOR s IN usr.shipped_order_history END
</pre> 
