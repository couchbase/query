## Shopper - Listing the top 10 best selling products

Don wants to know what are the top 10 best selling products on the dimestore website. 

Thanks to N1QL, we can now easily query the data in Couchbase to produce that list. 

![ScreenShot](./images/top10.png)

<pre id="example">
	SELECT product.name, sum(items.count) as unitsSold 
	FROM purchases unnest purchases.lineItems as items 
	JOIN product key items.product 
	GROUP BY product 
	ORDER BY unitsSold desc limit 10	
</pre>
