## Shopper - Listing products in a category 

Don also wants to browse through some appliances. Maybe, a dishwasher to wash his cup. What do you think?

He clicks on the "Appliances" category on the site menu, and the website displays a list of appliances he can browse through.

<pre id="example">
    SELECT
	product 
	FROM product
	UNNEST product.categories as categories
	WHERE categories = "Appliances"
</pre>
