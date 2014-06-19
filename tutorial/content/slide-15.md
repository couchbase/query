## Array Operations and Slicing 

Array slicing refers to using a subset of an array. N1QL allows slices to appear in the appear anywhere in a SELECT query. 

The query on the right will return all the children between offset 0 and 2

Try the typing out the following query on the right hand pane. This query will return all the children between offset 1 and the end of the array.

<span style="color: red">
SELECT VALUE( c )
    FROM tutorial.children[1:] as c
</span>

The difference between the first and second example is that in the second the slice operation appears in the FROM clase


N1QL also supports ARRAY functions such as ARRAY_PREPEND, ARRAY_APPEND and ARRAY_CONCAT. Try typing out the following query

<span style="color: red">
SELECT ARRAY_CONCAT(children[:1], children[1:]) FROM tutorial
</span>

<pre id="example">
SELECT children[0:2] 
    FROM tutorial 
        WHERE children[0:2] IS NOT MISSING
</pre>


