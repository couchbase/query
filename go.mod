module github.com/couchbase/query

go 1.13

replace golang.org/x/text => golang.org/x/text v0.3.7

replace github.com/couchbase/cbauth => ../cbauth

replace github.com/couchbase/cbft => ../../../../../cbft

replace github.com/couchbase/cbftx => ../../../../../cbftx

replace github.com/couchbase/hebrew => ../../../../../hebrew

replace github.com/couchbase/cbgt => ../../../../../cbgt

replace github.com/couchbase/eventing-ee => ../eventing-ee

replace github.com/couchbase/go-couchbase => ../go-couchbase

replace github.com/couchbase/go_json => ../go_json

replace github.com/couchbase/gomemcached => ../gomemcached

replace github.com/couchbase/goutils => ../goutils

replace github.com/couchbase/godbc => ../godbc

replace github.com/couchbase/indexing => ../indexing

replace github.com/couchbase/gometa => ../gometa

replace github.com/couchbase/n1fty => ../n1fty

replace github.com/couchbase/plasma => ../plasma

replace github.com/couchbase/query => ./empty

replace github.com/couchbase/query-ee => ../query-ee

require (
	github.com/couchbase/cbauth v0.1.1
	github.com/couchbase/clog v0.1.0
	github.com/couchbase/eventing-ee v0.0.0-00010101000000-000000000000
	github.com/couchbase/go-couchbase v0.1.1
	github.com/couchbase/go_json v0.0.0-00010101000000-000000000000
	github.com/couchbase/gocbcore-transactions v0.0.0-20220110140047-0cfbabaea2ec
	github.com/couchbase/gocbcore/v10 v10.0.9-0.20220228174725-170b8c650e3f
	github.com/couchbase/godbc v0.0.0-20210615212222-79da1b49cb4d
	github.com/couchbase/gomemcached v0.1.4
	github.com/couchbase/goutils v0.1.2
	github.com/couchbase/indexing v0.0.0-00010101000000-000000000000
	github.com/couchbase/n1fty v0.0.0-00010101000000-000000000000
	github.com/couchbase/query-ee v0.0.0-00010101000000-000000000000
	github.com/couchbase/retriever v0.0.0-20150311081435-e3419088e4d3
	github.com/couchbasedeps/go-curl v0.0.0-20190830233031-f0b2afc926ec
	github.com/gorilla/mux v1.8.0
	github.com/mattn/go-runewidth v0.0.3
	github.com/natefinch/npipe v0.0.0-20160621034901-c1b8fa8bdcce // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/peterh/liner v1.2.0
	github.com/russross/blackfriday v1.5.2
	github.com/samuel/go-zookeeper v0.0.0-20201211165307-7117e9ea2414
	golang.org/x/crypto v0.0.0-20220722155217-630584e8d5aa
	golang.org/x/net v0.8.0
	gopkg.in/couchbase/gocb.v1 v1.6.7
	gopkg.in/couchbase/gocbcore.v7 v7.1.18 // indirect
	gopkg.in/couchbaselabs/gocbconnstr.v1 v1.0.4 // indirect
	gopkg.in/couchbaselabs/gojcbmock.v1 v1.0.4 // indirect
	gopkg.in/couchbaselabs/jsonx.v1 v1.0.0 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
)
