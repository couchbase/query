## Shopper - Finding the most popular products in a category 

With so many appliances to choose from, Don wants to know the top 3 appliances so that he can easily pick which one to buy. 

What are the top 3 highly rated appliances? Run the query to figure this out. 

![ScreenShot](./images/top3.png)

<pre id="example">
    SELECT
	product.name, 
	count(reviews) as reviewCount,
	round(avg(reviews.rating),1) as AvgRating,
	category from reviews
	AS reviews
	JOIN product as product key reviews.productId
	UNNEST product.categories as category
	where category = "Appliances"
	GROUP by category, product
	ORDER by AvgRating 
	DESC LIMIT 3 
</pre>
