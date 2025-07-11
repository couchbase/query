curl -v -X POST -u $USER:$PASSWORD \
  $BASEPATH/settings/querySettings \
  -d 'queryTmpSpaceDir=/tmp' \
  -d 'queryTmpSpaceSize=2048'