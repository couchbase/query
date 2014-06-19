## Pattern Matching with LIKE

Inexact string matching can be accomplished using the LIKE operator in the WHERE clause.

The argument on the right hand side of the keyword LIKE is the pattern that the expression must match.
In these patterns `%` is a wildcard which will match 0 or more characters.  Also `_` can be used to match
exactly 1 character.

In the example on the right we look for people who have a yahoo.com email address.

Try changing LIKE to NOT LIKE.

<pre id="example">
SELECT fname, email
    FROM tutorial 
        WHERE email LIKE '%@yahoo.com'
</pre>
