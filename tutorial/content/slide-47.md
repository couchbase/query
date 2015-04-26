## Merchant - Generating the month-over-month sales report

Sonia, the general manager of dimestore, has asked her sales staff to
put together a month-over-month sales report.

![ScreenShot](./images/salesmam.png)

Rudy runs the N1QL query to generate the data needed for his
report. Try it out.

<pre id="example">
SELECT SUBSTR(purchases.purchasedAt, 0, 7) as month, 
	ROUND(SUM(product.unitPrice * items.count)/1000000, 3) revenueMillion
FROM purchases UNNEST purchases.lineItems AS items JOIN product ON KEYS items.product 
GROUP BY SUBSTR(purchases.purchasedAt, 0, 7) 
ORDER BY month 
</pre>
