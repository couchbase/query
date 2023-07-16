#!/bin/bash

#  Copyright 2023-Present Couchbase, Inc.
#
#  Use of this software is governed by the Business Source License included
#  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
#  in that file, in accordance with the Business Source License, use of this
#  software will be governed by the Apache License, Version 2.0, included in
#  the file licenses/APL2.txt.

# This scripts generates a go-lang map object containing the grammar used by the Query engine

if [ $# != 1 ]
then
  echo "Missing argument to $0"
  exit 1
fi

BASEPATH=$1
FILE=${BASEPATH}/shell/cbq/command/syntax_data.go

if [ ! -f ${BASEPATH}/parser/n1ql/n1ql.y ]
then
  echo "Invalid base path: ${BASEPATH}"
  exit 1
fi

if [ ! -f `which bison` ]
then
  echo "bison not found"
  exit 1
fi

AC='
BEGIN \
{
  active=0
  count=0
}
index($2,":")!=0 \
{
  gsub(":","",$2)
  if ($2=="input"||$2=="expr_input"||$2=="hints_input"||index($2,"$")!=0||$2==rn) next
  active=1
  if ($2=="statement_body") $2="statements"
  if (count>0) printf("\n\t},\n");
  printf("\t\"%s\": [][]string{\n\t\t[]string{",$2)
  rn=$2
  for (i=3;i<NF;i++) if (index($i,"$")==0) printf("\"%s\", ",$i)
  if (index($NF,"$")==0) { printf("\"%s\"},",$NF) } else { printf("},") }
  count++
  next
}
$2=="|" \
{
  if (active!=1) next
  printf("\n\t\t[]string{")
  for (i=3;i<NF;i++) if (index($i,"$")==0) printf("\"%s\", ",$i)
  if (index($NF,"$")==0) { printf("\"%s\"},",$NF) } else { printf("},") }
  next
}
END \
{
  print "\n\t},\n}"
}
'

bison -v -o /dev/null --report-file=/tmp/$$.bison ${BASEPATH}/parser/n1ql/n1ql.y

cat - << EOF > "${FILE}"
//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package command

var statement_syntax = map[string][][]string{
EOF
sed -n '/^Grammar/,/^Terminals/ p' /tmp/$$.bison|grep -v "^[A-Z]"|sed 's/stmt/statement/g'|awk "${AC}" >> "${FILE}"
rm -f /tmp/$$.bison
