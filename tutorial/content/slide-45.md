## Merchant - Reporting the active monthly customers

In the e-commerce world, purchases define user activity and growth. The dimestore sales team wants to know the number of unique customers that purchased something on the site in the last month.  Dixon has been asked to produce a report. 

He uses N1QL to query Couchbase and get the numbers he needs for his report.
 
![ScreenShot](./images/activeshopper.png)

<pre id="example">
	SELECT COUNT(DISTINCT purchases.customerId) 
	FROM purchases
	WHERE str_to_millis(purchases.purchasedAt) BETWEEN str_to_millis("2014-02-01") AND str_to_millis("2014-03-01")
</pre>

Now, think about how you would change this query to get a 7-day trend or a 24-hr trend
