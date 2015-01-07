## Assembling and loading user profiles

When a player loads his gameworld the client application needs to load the data from all the buckets. 
This can be accomplished by running a single N1QL query. The query on the right assembles 
the blobs from all three buckets for a user with key "zid-jungle-0001".

If you notice the game-data attribute has a hyphen in its name, and when used in the query it needs to be
escaped with a backtick character, `game-data`. 

<pre id="example">
SELECT * 
FROM jungleville AS `game-data` 
JOIN  jungleville_stats AS stats
      ON KEYS "zid-jungle-stats-0001" 
NEST  jungleville_inbox AS inbox 
      ON KEYS "zid-jungle-inbox-0001" 
WHERE `game-data`.uuid="zid-jungle-0001"
</pre>
