## Shopper - Browsing products and sorting results 

Don wants a list of new and popular items - i.e. items that are
recently added to the product catalog and have many units sold.

![ScreenShot](./images/sortby.png)

<pre id="example">
	SELECT product.name, product.unitPrice, product.dateAdded, SUM(items.count) AS unitsSold 
		FROM purchases UNNEST purchases.lineItems AS items 
		JOIN product ON KEYS items.product 
		GROUP BY product 
		ORDER BY product.dateAdded, unitsSold DESC LIMIT 10
</pre>

How about sorting by price? Edit the query at the right to sort by
unit price, lowest to highest.
