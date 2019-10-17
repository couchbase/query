export test=beer_sample
    export result=`diff -q $test.out.json $test.out.json`
    echo "result is: *" $result "*"
    if [ -n  "$result" ]
    then
        echo "ERROR - inferred schema doesn't match: " $result
    fi
