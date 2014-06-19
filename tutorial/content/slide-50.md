## Listing friends

To compute local leaderboards, you might want to join a player's blob with their friends blob. This N1QL query shows you how to do that.

<pre id="example">
SELECT jungleville.level, friends 
FROM jungleville KEY "zid-jungle-0002" 
JOIN jungleville.friends
	KEYS jungleville.friends
</pre>

Thank you for exploring N1QL. Remember, this was just a quick tutorial. <a href="http://www.couchbase.com/communities/n1ql#n1qldownload">Download N1QL</a> today and try out more complex queries.
