## Matching Elements in Nested Arrays with ANY

Sometimes the conditions you want to filter need to be applied to arrays nested inside the document.  In the tutorial sample dataset people contain an array of children, and each child has a name and an age.

In the example on the right we want to find any person that has a child over the age of 10.

This can be achieved by using the ANY/EVERY - SATISFIES construct. 

The expression after the ANY clause allows us to assign a name to an element in the array that we are searching through. The SATISFIES keyword is used to specify the filter condition. 

Try changing ANY to EVERY.

<pre id="example">
SELECT fname, children
    FROM tutorial 
        WHERE ANY child IN tutorial.children SATISFIES child.age > 10  END
</pre>
