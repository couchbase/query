curl -X POST \
"$BASEPATH/evaluator/v1/libraries/science?bucket=travel-sample&scope=inventory" \
-u $USER:$PASSWORD \
-H 'content-type: application/json' \
-d 'function f2c(f) { return (5/9)*(f-32); }'