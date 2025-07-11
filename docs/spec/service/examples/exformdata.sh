curl -v $BASEPATH/query/service \
     -d 'statement=SELECT name FROM `travel-sample`.inventory.hotel LIMIT 1' \
     -u $USER:$PASSWORD