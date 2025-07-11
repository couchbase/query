curl -v $BASEPATH/query/service \
     -u $USER:$PASSWORD \
     -d 'statement=SELECT meta().id
                   FROM `travel-sample`.inventory.hotel
                   WHERE meta().id LIKE $pattern
       & $pattern="hotel_1002%25"'