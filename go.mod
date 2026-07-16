module github.com/couchbase/query

go 1.25.5

replace golang.org/x/text => golang.org/x/text v0.4.0

replace github.com/couchbase/bhive => ../bhive

replace github.com/couchbase/cbauth => ../cbauth

replace github.com/couchbase/cbft => ../../../../../cbft

replace github.com/couchbase/cbftx => ../../../../../cbftx

replace github.com/couchbase/hebrew => ../../../../../hebrew

replace github.com/couchbase/cbgt => ../../../../../cbgt

replace github.com/couchbase/eventing => ../eventing

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

replace github.com/couchbase/regulator => ../regulator

replace github.com/couchbase/gocbcrypto => ../gocbcrypto

require (
	cloud.google.com/go/storage v1.62.1
	github.com/apache/arrow-go/v18 v18.6.0
	github.com/apache/iceberg-go v0.6.0
	github.com/aws/aws-sdk-go-v2 v1.41.7
	github.com/aws/aws-sdk-go-v2/config v1.32.17
	github.com/aws/aws-sdk-go-v2/credentials v1.19.16
	github.com/aws/aws-sdk-go-v2/service/s3 v1.101.0
	github.com/couchbase/cbauth v0.1.20
	github.com/couchbase/clog v0.1.0
	github.com/couchbase/eventing-ee v0.0.0-00010101000000-000000000000
	github.com/couchbase/go-couchbase v0.1.1
	github.com/couchbase/go_json v0.0.0-20250803152502-4641f4ef8578
	github.com/couchbase/gocb/v2 v2.12.4
	github.com/couchbase/gocbcore/v10 v10.9.3
	github.com/couchbase/gocbcrypto v0.0.0-00010101000000-000000000000
	github.com/couchbase/godbc v0.0.0-20210615212222-79da1b49cb4d
	github.com/couchbase/gomemcached v0.3.4
	github.com/couchbase/goutils v0.3.0
	github.com/couchbase/indexing v0.0.0-20220923223016-071e9308c875
	github.com/couchbase/n1fty v0.0.0-00010101000000-000000000000
	github.com/couchbase/query-ee v0.0.0-00010101000000-000000000000
	github.com/couchbase/regulator v0.0.0-00010101000000-000000000000
	github.com/golang/snappy v1.0.0
	github.com/hamba/avro/v2 v2.31.0
	github.com/kylelemons/godebug v1.1.0
	github.com/lib/pq v1.10.9
	github.com/mattn/go-runewidth v0.0.20
	github.com/mattn/go-sqlite3 v1.14.34
	github.com/peterh/liner v1.2.0
	github.com/russross/blackfriday v1.5.2
	github.com/samuel/go-zookeeper v0.0.0-20201211165307-7117e9ea2414
	github.com/scritchley/orc v0.0.0-20210513144143-06dddf1ad665
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78
	golang.org/x/crypto v0.52.0
	golang.org/x/net v0.55.0
	golang.org/x/oauth2 v0.36.0
	golang.org/x/sys v0.46.0
	golang.org/x/term v0.43.0
	google.golang.org/api v0.278.0
)

require (
	atomicgo.dev/cursor v0.2.0 // indirect
	atomicgo.dev/keyboard v0.2.9 // indirect
	atomicgo.dev/schedule v0.1.0 // indirect
	cel.dev/expr v0.25.1 // indirect
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/auth v0.20.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	cloud.google.com/go/iam v1.7.0 // indirect
	cloud.google.com/go/monitoring v1.24.3 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.20.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.13.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.6.4 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.1 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.6.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.31.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.55.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.55.0 // indirect
	github.com/RoaringBitmap/roaring/v2 v2.18.0 // indirect
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/apache/thrift v0.23.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.10 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.23 // indirect
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.20.18 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.24 // indirect
	github.com/aws/aws-sdk-go-v2/service/glue v1.141.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.42.1 // indirect
	github.com/aws/smithy-go v1.25.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.24.5 // indirect
	github.com/blevesearch/bleve-mapping-ui v0.7.0 // indirect
	github.com/blevesearch/bleve/v2 v2.6.1-0.20260716162251-eea04eb5c5bf // indirect
	github.com/blevesearch/bleve_index_api v1.4.0 // indirect
	github.com/blevesearch/geo v0.2.5 // indirect
	github.com/blevesearch/go-faiss v1.1.5 // indirect
	github.com/blevesearch/go-porterstemmer v1.0.3 // indirect
	github.com/blevesearch/goleveldb v1.0.1 // indirect
	github.com/blevesearch/gtreap v0.1.1 // indirect
	github.com/blevesearch/mmap-go v1.2.0 // indirect
	github.com/blevesearch/scorch_segment_api/v2 v2.4.8 // indirect
	github.com/blevesearch/sear v0.4.2 // indirect
	github.com/blevesearch/segment v0.9.1 // indirect
	github.com/blevesearch/snowballstem v0.9.0 // indirect
	github.com/blevesearch/stempel v0.2.0 // indirect
	github.com/blevesearch/upsidedown_store_api v1.0.2 // indirect
	github.com/blevesearch/vellum v1.2.0 // indirect
	github.com/blevesearch/zapx/v11 v11.4.3 // indirect
	github.com/blevesearch/zapx/v12 v12.4.3 // indirect
	github.com/blevesearch/zapx/v13 v13.4.3 // indirect
	github.com/blevesearch/zapx/v14 v14.4.3 // indirect
	github.com/blevesearch/zapx/v15 v15.4.3 // indirect
	github.com/blevesearch/zapx/v16 v16.3.4 // indirect
	github.com/blevesearch/zapx/v17 v17.2.0 // indirect
	github.com/buger/jsonparser v1.1.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/cloudfoundry/gosigar v1.3.4 // indirect
	github.com/cncf/xds/go v0.0.0-20260202195803-dba9d589def2 // indirect
	github.com/cockroachdb/apd/v3 v3.2.1 // indirect
	github.com/containerd/console v1.0.5 // indirect
	github.com/couchbase/blance v0.1.6 // indirect
	github.com/couchbase/cbft v0.0.0-00010101000000-000000000000 // indirect
	github.com/couchbase/cbgt v0.0.0-00010101000000-000000000000 // indirect
	github.com/couchbase/ghistogram v0.1.0 // indirect
	github.com/couchbase/gocbcore/v9 v9.1.11 // indirect
	github.com/couchbase/gocbcoreps v0.1.5-0.20260107140814-1c3a03f888f8 // indirect
	github.com/couchbase/gometa v0.0.0-20220803182802-05cb6b2e299f // indirect
	github.com/couchbase/goprotostellar v1.0.6-0.20260407143512-d7af25156dcc // indirect
	github.com/couchbase/hebrew v0.0.0-00010101000000-000000000000 // indirect
	github.com/couchbase/logstats v1.1.1 // indirect
	github.com/couchbase/moss v0.3.0 // indirect
	github.com/couchbase/tools-common/cloud/v8 v8.1.4 // indirect
	github.com/couchbase/tools-common/couchbase/v4 v4.1.0 // indirect
	github.com/couchbase/tools-common/fs v1.0.3 // indirect
	github.com/couchbase/tools-common/strings v1.0.0 // indirect
	github.com/couchbase/tools-common/sync/v2 v2.0.2 // indirect
	github.com/couchbase/tools-common/testing v1.0.3 // indirect
	github.com/couchbase/tools-common/types v1.1.4 // indirect
	github.com/couchbase/tools-common/types/v2 v2.2.2 // indirect
	github.com/couchbase/tools-common/utils/v3 v3.1.1 // indirect
	github.com/couchbaselabs/gocbconnstr/v2 v2.0.0 // indirect
	github.com/creasty/defaults v1.8.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dsnet/compress v0.0.2-0.20210315054119-f66993602bf5 // indirect
	github.com/elazarl/go-bindata-assetfs v1.0.1 // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.37.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.3 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/goccy/go-yaml v1.17.1 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/flatbuffers v25.12.19+incompatible // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/google/wire v0.7.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.15 // indirect
	github.com/googleapis/gax-go/v2 v2.22.0 // indirect
	github.com/gookit/color v1.6.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/lithammer/fuzzysearch v1.1.8 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pierrec/lz4/v4 v4.1.26 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.23.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/pterm/pterm v0.12.83 // indirect
	github.com/puzpuzpuz/xsync/v3 v3.5.1 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/santhosh-tekuri/jsonschema v1.2.4 // indirect
	github.com/spiffe/go-spiffe/v2 v2.6.0 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/substrait-io/substrait v0.87.0 // indirect
	github.com/substrait-io/substrait-go/v8 v8.1.0 // indirect
	github.com/substrait-io/substrait-protobuf/go v0.85.0 // indirect
	github.com/tmthrgd/go-hex v0.0.0-20190904060850-447a3041c3bc // indirect
	github.com/twmb/avro v1.7.2 // indirect
	github.com/twmb/murmur3 v1.1.8 // indirect
	github.com/uptrace/bun v1.2.18 // indirect
	github.com/uptrace/bun/dialect/mssqldialect v1.2.18 // indirect
	github.com/uptrace/bun/dialect/mysqldialect v1.2.18 // indirect
	github.com/uptrace/bun/dialect/oracledialect v1.2.18 // indirect
	github.com/uptrace/bun/dialect/pgdialect v1.2.18 // indirect
	github.com/uptrace/bun/dialect/sqlitedialect v1.2.18 // indirect
	github.com/uptrace/bun/extra/bundebug v1.2.18 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.etcd.io/bbolt v1.4.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.42.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.68.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.67.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	gocloud.dev v0.45.0 // indirect
	golang.org/x/exp v0.0.0-20260218203240-3dfff04db8fa // indirect
	golang.org/x/mod v0.34.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	google.golang.org/genproto v0.0.0-20260319201613-d00831a3d3e7 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/grpc v1.81.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
