## Array operations and slicing 

Array slicing refers to using a subset of an array. N1QL allows slices
to appear anywhere that other expressions can appear.

The query on the right will return all the children between offset 0
and 2.

N1QL also supports ARRAY functions such as ARRAY_PREPEND, ARRAY_APPEND
and ARRAY_CONCAT. Try typing out the following query:

<span>
SELECT ARRAY_CONCAT(children[0:1], children[0:1]) FROM tutorial
</span>

<pre id="example">
SELECT children[0:2] 
    FROM tutorial 
        WHERE children[0:2] IS NOT MISSING
</pre>


