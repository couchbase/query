## Shopper - Listing the top 10 best selling products

Don wants to know what are the top 10 best selling products on the dimestore website. 

Thanks to N1QL, we can now easily query the data in Couchbase to produce that list. 

![ScreenShot](./images/top10.png)

<pre id="example">
	SELECT product.name, SUM(items.count) AS unitsSold 
	FROM purchases UNNEST purchases.lineItems AS items 
	JOIN product ON KEYS items.product 
	GROUP BY product 
	ORDER BY unitsSold DESC LIMIT 10	
</pre>
