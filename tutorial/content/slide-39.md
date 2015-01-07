## Shopper - Shopping at a one-day sale

dimestore announces a one-day super sale, with deals for many products.
 
Are there any appliances on sale below $6.99? 

![ScreenShot](./images/onedaysale.png)

<pre id="example">
	SELECT product.name, product.unitPrice, product.categories 
	FROM product unnest product.categories as categories 
	WHERE categories = "Appliances" and product.unitPrice < 6.99
</pre>
