## Merchant - Finding the most valued shoppers

The marketing team at dimestore wants to e-mail special discount
coupons to the top 10 loyal shoppers.

List the top 10 shoppers based on the total amount spent 
 
![ScreenShot](./images/coupons.png)

<pre id="example">
	SELECT 	customer.firstName, 
		customer.lastName, 
		customer.emailAddress,
		SUM(items.count) purchaseCount, 
		ROUND(SUM(product.unitPrice * items.count))  totalSpent 
	FROM purchases UNNEST purchases.lineItems AS items 
	JOIN product ON KEYS items.product
	JOIN customer ON KEYS purchases.customerId 
	GROUP BY customer 
	ORDER BY totalSpent DESC LIMIT 10
</pre>
