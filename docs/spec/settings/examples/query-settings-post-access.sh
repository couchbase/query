curl -v -X POST -u $USER:$PASSWORD \
  $BASEPATH/settings/querySettings/curlWhitelist \
  -H 'Content-Type: application/json' \
  -d '{"all_access": false,
       "allowed_urls": ["https://company1.com"],
       "disallowed_urls": ["https://company2.com"]}'