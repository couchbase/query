# Copyright 2024-Present Couchbase, Inc.
#
# Use of this software is governed by the Business Source License included in
# the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
# file, in accordance with the Business Source License, use of this software
# will be governed by the Apache License, Version 2.0, included in the file
# licenses/APL2.txt.

echo "Validating error description information"
TMP=/tmp/$$.ed
awk '/ErrorCode =/{print $1}' ../errors/codes.go|egrep -v "_RETIRED|_UNUSED|E_OK"| while read sym rest
do
  C=`grep -c $sym ../errors/messages.go`
  if [ $C -eq 0 ]
  then
    echo " - missing description for: $sym"
  fi
done > $TMP
TOT=`wc -l $TMP|awk '{print $1}'`
cat $TMP
rm -f $TMP
echo "Done.  ${TOT} missing description(s)."
echo go build finderr.go
go build finderr.go
