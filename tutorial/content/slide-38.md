## Shopper - Browsing products and sorting results 

Don wants a list of new and popular items - i.e. items that are recently added to the product catalog and have many units sold.

![ScreenShot](./images/sortby.png)

<pre id="example">
	SELECT product.name, product.dateAdded, sum(items.count) as unitsSold 
		FROM purchases unnest purchases.lineItems as items 
		JOIN product key items.product 
		GROUP BY product 
		ORDER BY product.dateAdded, unitsSold desc limit 10
</pre>

How about sorting by price? - Copy, paste, and run the following query to sort by price (Low to high)

<span style="color: red">
select product from product order by unitPrice desc limit 100
</span>
