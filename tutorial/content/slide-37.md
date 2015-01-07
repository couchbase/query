## Shopper - Finding the most popular products in a category 

With so many appliances to choose from, Don wants to know the top 3 appliances so that he can easily pick which one to buy. 

What are the top 3 highly rated appliances? Run the query to figure this out. 

![ScreenShot](./images/top3.png)

<pre id="example">
    SELECT
	product.name, 
	COUNT(reviews) AS reviewCount,
	ROUND(AVG(reviews.rating),1) AS AvgRating,
	category 
        FROM reviews AS reviews
	JOIN product AS product 
        ON KEYS reviews.productId
	UNNEST product.categories AS category
	WHERE category = "Appliances"
	GROUP BY category, product
	ORDER BY AvgRating 
	DESC LIMIT 3 
</pre>
