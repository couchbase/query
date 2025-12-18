curl -X POST \
"$BASEPATH/evaluator/v1/libraries/math" \
-u $USER:$PASSWORD \
-H 'content-type: application/json' \
-d 'function add(a, b) { let data = a + b; return data; }
    function sub(a, b) { let data = a - b; return data; }
    function mul(a, b) { let data = a * b; return data; }'