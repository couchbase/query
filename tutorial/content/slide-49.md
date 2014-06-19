## Generating global leaderboards

Do we have a leader of the jungle? Leaderboards show global rankings, ordered by the number of wins and the player level. 

How does the leaderboard for jungleville look like today? Run the query to figure it out.

<pre id="example">
SELECT player.name, 
       player.level, 
       stats.loadtime, 
       SUM(CASE WHEN hist.result = "won" THEN 1 ELSE 0 END) AS wins
FROM jungleville_stats AS stats 
	UNNEST stats.pvp-hist AS hist 
JOIN jungleville AS player KEY stats.uuid
GROUP BY player, stats
ORDER BY wins DESC, player.level DESC
</pre>
