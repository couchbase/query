module github.com/couchbase/query

go 1.18

replace github.com/couchbase/cbauth => ../cbauth

replace github.com/couchbase/cbft => ../../../../../cbft

replace github.com/couchbase/cbftx => ../../../../../cbftx

replace github.com/couchbase/cbgt => ../../../../../cbgt

replace github.com/couchbase/eventing-ee => ../eventing-ee

replace github.com/couchbase/go-couchbase => ../go-couchbase

replace github.com/couchbase/go_json => ../go_json

replace github.com/couchbase/gomemcached => ../gomemcached

replace github.com/couchbase/indexing => ../indexing

replace github.com/couchbase/gometa => ../gometa

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
	github.com/couchbase/gocbcore-transactions v0.0.0-20210420181536-74d74532b5ca
	github.com/couchbase/gocbcore/v9 v9.1.7-0.20210825200734-fa22caf5138a
	github.com/couchbase/godbc v0.0.0-20210615233721-eaf4ccfce7f8
	github.com/couchbase/gomemcached v0.0.0-20200618124739-5bac349aff71
	github.com/couchbase/goutils v0.0.0-20201030094643-5e82bb967e67
	github.com/couchbase/indexing v0.0.0-00010101000000-000000000000
	github.com/couchbase/n1fty v0.0.0-00010101000000-000000000000
	github.com/couchbase/query-ee v0.0.0-00010101000000-000000000000
	github.com/couchbase/retriever v0.0.0-20150311081435-e3419088e4d3
	github.com/couchbasedeps/go-curl v0.0.0-20190830233031-f0b2afc926ec
	github.com/gorilla/mux v1.7.4
	github.com/peterh/liner v1.2.0
	github.com/russross/blackfriday v1.5.2
	github.com/samuel/go-zookeeper v0.0.0-20200724154423-2164a8ac840e
	golang.org/x/crypto v0.0.0-20221010152910-d6f0a8c073c2
	golang.org/x/net v0.0.0-20220728211354-c7608f3a8462
	gopkg.in/couchbase/gocb.v1 v1.6.7
)

require (
	github.com/RoaringBitmap/roaring v0.4.23 // indirect
	github.com/blevesearch/bleve-mapping-ui v0.4.0 // indirect
	github.com/blevesearch/bleve/v2 v2.0.4-0.20210810162943-2b21ae8f266f // indirect
	github.com/blevesearch/bleve_index_api v1.0.1 // indirect
	github.com/blevesearch/go-porterstemmer v1.0.3 // indirect
	github.com/blevesearch/mmap-go v1.0.2 // indirect
	github.com/blevesearch/scorch_segment_api/v2 v2.0.1 // indirect
	github.com/blevesearch/segment v0.9.0 // indirect
	github.com/blevesearch/snowballstem v0.9.0 // indirect
	github.com/blevesearch/upsidedown_store_api v1.0.1 // indirect
	github.com/blevesearch/vellum v1.0.3 // indirect
	github.com/blevesearch/zapx/v11 v11.2.1-0.20210809173656-f061f2a21cb9 // indirect
	github.com/blevesearch/zapx/v12 v12.2.1-0.20210809173531-2ea06c038419 // indirect
	github.com/blevesearch/zapx/v13 v13.2.1-0.20210809173433-6a16986ce5d9 // indirect
	github.com/blevesearch/zapx/v14 v14.2.1-0.20210809173320-a8a0c8c03c5b // indirect
	github.com/blevesearch/zapx/v15 v15.2.1-0.20210809172947-0534019802b1 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/couchbase/blance v0.0.0-20210701151549-a83d808be6d1 // indirect
	github.com/couchbase/cbft v0.0.0-00010101000000-000000000000 // indirect
	github.com/couchbase/cbgt v0.0.0-00010101000000-000000000000 // indirect
	github.com/couchbase/ghistogram v0.1.0 // indirect
	github.com/couchbase/gometa v0.0.0-20200717102231-b0e38b71d711 // indirect
	github.com/couchbase/moss v0.1.0 // indirect
	github.com/dustin/go-jsonpointer v0.0.0-20140810065344-75939f54b39e // indirect
	github.com/dustin/gojson v0.0.0-20150115165335-af16e0e771e2 // indirect
	github.com/elazarl/go-bindata-assetfs v1.0.0 // indirect
	github.com/glycerine/go-unsnap-stream v0.0.0-20181221182339-f9677308dec2 // indirect
	github.com/golang/protobuf v1.4.0 // indirect
	github.com/golang/snappy v0.0.3 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/json-iterator/go v0.0.0-20171115153421-f7279a603ede // indirect
	github.com/mattn/go-runewidth v0.0.3 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/natefinch/npipe v0.0.0-20160621034901-c1b8fa8bdcce // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/philhofer/fwd v1.0.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0 // indirect
	github.com/steveyen/gtreap v0.1.0 // indirect
	github.com/syndtr/goleveldb v1.0.0 // indirect
	github.com/tinylib/msgp v1.1.0 // indirect
	github.com/willf/bitset v1.1.10 // indirect
	go.etcd.io/bbolt v1.3.5 // indirect
	golang.org/x/sys v0.0.0-20220728004956-3c1f35247d10 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20180817151627-c66870c02cf8 // indirect
	google.golang.org/grpc v1.24.0 // indirect
	google.golang.org/protobuf v1.21.0 // indirect
	gopkg.in/couchbase/gocbcore.v7 v7.1.18 // indirect
	gopkg.in/couchbaselabs/gocbconnstr.v1 v1.0.4 // indirect
	gopkg.in/couchbaselabs/gojcbmock.v1 v1.0.4 // indirect
	gopkg.in/couchbaselabs/jsonx.v1 v1.0.0 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
)
