## Shopper - Browsing and searching for a product
 
Because, there is no category called "cup", Don decides to search for product names that have the substring "cup".

<b>Did you know that ....</b><br/>
when researching branded products, 44% of online shoppers begin by using a search?

<pre id="example">
    SELECT 
	productId, name
	FROM product
	WHERE lower(name) like "%cup%"
</pre>
