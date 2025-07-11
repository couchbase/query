curl -v $BASEPATH/query/service \
     -d 'statement=SELECT airline FROM `travel-sample`.inventory.route
                   WHERE sourceairport = $1 AND distance > $2
       & args=["LAX", 13000]' \
     -u $USER:$PASSWORD