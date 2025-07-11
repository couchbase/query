curl -v $BASEPATH/query/service \
     -d 'statement=SELECT airline FROM `travel-sample`.inventory.route
                   WHERE sourceairport = ? AND distance > ?
       & args=["LAX", 13000]' \
     -u $USER:$PASSWORD