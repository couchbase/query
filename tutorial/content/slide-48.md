## Merchant - Big ticket orders

Sonia now wants to find out which of the orders in the month of April exceeded the unit price of $500. The query on right uses subquery clause to get the desired results

Note: The query examines a lot of document so it might take some time to finish

<pre id="example">
    SELECT purchases.purchaseId, l.product 
        FROM purchases UNNEST purchases.lineItems l 
            WHERE DATE_PART_STR(purchases.purchasedAt,"month") = 4
            AND DATE_PART_STR(purchases.purchasedAt,"year") = 2014 
            AND EXISTS (SELECT product.productId FROM product USE KEYS l.product 
                WHERE product.unitPrice > 500)
</pre>

