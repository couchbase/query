{{- /* iterations=5 */ -}}   
{{- /* First template comment matching the above defines how many queries should be generated using it */}}
{{- /* As is common in this testing, a negative number means "pick a random number up to abs(val)" */}}
{{- /* If not specified then it is used once.  Each iteration has a different RandomKeyspace list and the variable */}}
{{- /* .Iteration holds an integer indicating the iteration (starting with 0) */}} 
{{- /* JoinStrings is strings.Join provided for utility but is likely of limited use as unique elements aren't guaranteed */}}
{{- $t1 := index .Keyspaces .Iteration }}      {{- /* data element: keyspace names (strings) */}}
{{- $t2 := index .RandomKeyspaces 0}}          {{- /* data element: keyspace names (strings) */}}
{{- $jn1to2 := GetJoinOn $t1 "t1" $t2 "t2"}}   {{- /* GetJoinOn returns ON clause content (string) */}}
{{- $tf1 := RandomFields $t1 5 }}              {{- /* RandomFields returns a list of randomly selected field names (strings) */}}
{{- if $tf1 }}
  {{- $rf := index $tf1 0 }}
  SELECT {{range $i,$e:=$tf1}}{{$e}} AS p{{$i}},{{end}}
         (SELECT COUNT(1)
          FROM {{ $t2 }} AS t2
          WHERE {{ $jn1to2 }}
         ) AS x 
  FROM {{ $t1 }} AS t1
  WHERE {{ RandomFilter $t1 }}                     {{- /* RandomFilter generates a random filter clause for the keyspace */}}
  AND ({{ RandomFilter $t1 }}
      OR {{ RandomFilter $t1 }}
      )
  {{- $rfv:=""}}
  {{- if $rf }}
    {{- $rfv = GetValue $t1 $rf }}               {{- /* GetValue returns a generated value (in text form, string values quoted) */}}
  {{- end}}
  {{- if $rfv }}
    AND {{ $rf }} != {{ $rfv }}
  {{- end}}
  AND NVL(t1.known_field,"") != "something"
  {{- $kfv := GetValue $t1 "known_field2"}}
  {{- if $kfv }}
    AND NVL(t1.known_field2,0) < {{ $kfv }}
  {{- end}}
{{- end}} {{- /* if $tf1 */}}
