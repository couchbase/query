## Merchant - Reporting the active monthly customers

In the e-commerce world, purchases define user activity and
growth. The dimestore sales team wants to know the number of unique
customers that purchased something on the site in the last month.
Dixon has been asked to produce a report.

He uses N1QL to query Couchbase and get the numbers he needs for his
report.
 
![ScreenShot](./images/activeshopper.png)

<pre id="example">
	SELECT COUNT(DISTINCT purchases.customerId) 
	FROM purchases
	WHERE purchases.purchasedAt BETWEEN "2014-03-01" AND "2014-03-31"
</pre>

How would you change this query to get a 7-day report or a 24-hr report?
