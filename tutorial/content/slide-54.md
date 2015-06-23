## Listing friends

To compute local leaderboards, you might want to join a player's blob
with their friends blob. This query shows you how to do that.

<pre id="example">
SELECT jungleville.level, friends 
FROM jungleville USE KEYS "zid-jungle-0002" 
JOIN jungleville.friends
     ON KEYS jungleville.friends
</pre>

Thank you for exploring N1QL. Remember, this was just a quick
tutorial.
<a
href="http://www.couchbase.com/nosql-databases/downloads#PreRelease">Download
Couchbase 4.0</a> today and try out more complex queries.
