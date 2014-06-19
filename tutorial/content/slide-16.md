## Quick Review

Before we continue exploring N1QL further, let us take a look at a query that summarizes what we've learned so far.

Here we match people having a yahoo email address or having all of their children over the age of 10.  For each person satisfying these requirements, we display their full name, email address, and the full list of children.

Try appending the expression KEYS ["dave", "ian"] after the FROM expression to restict the scope of the query to primary keys "dave" and "ian"

<pre id="example">
SELECT fname || " " || lname AS full_name, email, children[0:2] AS offsprings
    FROM tutorial 
        WHERE email LIKE '%@yahoo.com' 
        OR ANY child IN tutorial.children SATISFIES child.age > 10 END
</pre>
