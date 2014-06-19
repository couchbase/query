##  Listing messages sent by a user 

In jungleville, players can send messages to other players. 

How do you get a list of all the messages sent by a player zid-jungle-0001 to all other players? Run this N1QL query to find out all the messages sent by player zid-jungle-0001. 

<pre id="example">
SELECT player.name, inbox.messages
FROM jungleville AS player 
	KEY "zid-jungle-0001" 
LEFT JOIN jungleville_inbox AS inbox 
	KEY "zid-jungle-inbox-" || SUBSTR(player.uuid, 11)
</pre>


