## Merchant - Identifying non-performing products

In order to maintain an assortment of products that reflect customer
demand and inventory productivity, dimestore uses product reviews to
get a list of low rated products to be removed.

Mary, a category manager at dimestore has asked Judy to come up with a
list of products that have average review score less than 1.

Run the query to find out which products have an average rating below
1.

<pre id="example">
	SELECT product.name, product.productId, ROUND(AVG(reviews.rating), 3) avgRating, COUNT(reviews) numReviews 
	FROM product JOIN reviews ON KEYS product.reviewList 
	GROUP BY product HAVING AVG(reviews.rating) < 1
</pre>
