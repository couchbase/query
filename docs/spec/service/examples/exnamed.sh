curl -v $BASEPATH/query/service \
     -d 'statement=SELECT airline FROM `travel-sample`.inventory.route
                   WHERE sourceairport = $aval AND distance > $dval
       & $aval="LAX" & $dval=13000' \
     -u $USER:$PASSWORD