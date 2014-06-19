## Merchant - Finding the most valued shoppers

The marketing team at dimestore wants to e-mail special discount coupons to the top 10 loyal shoppers.

List the top 10 shoppers based on the total amount spent 
 
![ScreenShot](./images/coupons.png)

<pre id="example">
	SELECT 	customer.firstName, 
		customer.lastName, 
		customer.emailAddress,
		sum(items.count) purchaseCount, 
		round(sum(product.unitPrice * items.count))  totalSpent 
	FROM purchases unnest purchases.lineItems as items 
	JOIN product key items.product join customer key purchases.customerId 
	GROUP BY customer 
	ORDER BY totalSpent desc limit 10	
</pre>
