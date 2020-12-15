module github.com/couchbase/query

go 1.13

replace github.com/couchbase/cbauth => ../cbauth

replace github.com/couchbase/cbft => ../../../../../cbft

replace github.com/couchbase/cbftx => ../../../../../cbftx

replace github.com/couchbase/cbgt => ../../../../../cbgt

replace github.com/couchbase/eventing-ee => ../eventing-ee

replace github.com/couchbase/go-couchbase => ../go-couchbase

replace github.com/couchbase/go_json => ../go_json

replace github.com/couchbaselabs/gocbcore-transactions => ../../couchbaselabs/gocbcore-transactions

replace github.com/couchbase/gomemcached => ../gomemcached

replace github.com/couchbase/indexing => ../indexing

replace github.com/couchbase/n1fty => ../n1fty

replace github.com/couchbase/plasma => ../plasma

replace github.com/couchbase/query => ./empty

replace github.com/couchbase/query-ee => ../query-ee

require (
	github.com/couchbase/cbauth v0.0.0-20200923220950-efdafddb9bd2
	github.com/couchbase/clog v0.0.0-20190523192451-b8e6d5d421bc
	github.com/couchbase/eventing-ee v0.0.0-00010101000000-000000000000
	github.com/couchbase/go-couchbase v0.0.0-20201026062457-7b3be89bbd89
	github.com/couchbase/go_json v0.0.0-00010101000000-000000000000
	github.com/couchbase/gocbcore/v9 v9.1.0
	github.com/couchbase/godbc v0.0.0-20201207142944-d43b329cdf71
	github.com/couchbase/gomemcached v0.0.0-20200618124739-5bac349aff71
	github.com/couchbase/gometa v0.0.0-20200717102231-b0e38b71d711 // indirect
	github.com/couchbase/goutils v0.0.0-20201030094643-5e82bb967e67
	github.com/couchbase/indexing v0.0.0-00010101000000-000000000000
	github.com/couchbase/n1fty v0.0.0-00010101000000-000000000000
	github.com/couchbase/query-ee v0.0.0-00010101000000-000000000000
	github.com/couchbase/retriever v0.0.0-20150311081435-e3419088e4d3
	github.com/couchbasedeps/go-curl v0.0.0-20190830233031-f0b2afc926ec
	github.com/couchbaselabs/gocbcore-transactions v0.0.0-00010101000000-000000000000
	github.com/gorilla/mux v1.7.4
	github.com/natefinch/npipe v0.0.0-20160621034901-c1b8fa8bdcce // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/peterh/liner v1.2.0
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0 // indirect
	github.com/russross/blackfriday v1.5.2
	github.com/samuel/go-zookeeper v0.0.0-20200724154423-2164a8ac840e
	github.com/sbinet/liner v0.0.0-20150202172121-d9335eee40a4
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	golang.org/x/net v0.0.0-20190404232315-eb5bcb51f2a3
	gopkg.in/couchbase/gocb.v1 v1.6.7
	gopkg.in/couchbase/gocbcore.v7 v7.1.18 // indirect
	gopkg.in/couchbaselabs/gocbconnstr.v1 v1.0.4 // indirect
	gopkg.in/couchbaselabs/gojcbmock.v1 v1.0.4 // indirect
	gopkg.in/couchbaselabs/jsonx.v1 v1.0.0 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
)
