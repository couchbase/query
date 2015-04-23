## Merchant - Big ticket orders

Sonia now wants to find out which of the orders in the month of April
exceeded the unit price of $500. The query on the right uses UNNEST
and JOIN clauses to get the desired results.

<pre id="example">
    SELECT purchases.purchaseId, l.product, prod.name
        FROM purchases UNNEST purchases.lineItems l
            JOIN product prod ON KEYS l.product
        WHERE DATE_PART_STR(purchases.purchasedAt,"month") = 4
            AND DATE_PART_STR(purchases.purchasedAt,"year") = 2014 
            AND prod.unitPrice > 500
</pre>

