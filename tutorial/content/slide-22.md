## Array Comprehensions

Array comprehensions allow you to build new arrays from existing ones.

In the example on the right we build a new array containing just the childrens names for all people that have children.

<pre id="example">
	SELECT 
		ARRAY child.fname FOR child IN tutorial.children END AS children_names
			FROM tutorial 
				WHERE children IS NOT NULL
</pre>
