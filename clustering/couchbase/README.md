
## Running clustering_cb unit tests

In order to run the unit tests, it is necessary to install and start Couchbase.

### Couchbase Installation and Start-up

Install and start Couchbase:

+ [Install and build instructions](https://github.com/couchbase/tlm/)

Note: build the master branch, because this has support for managing the query engine (cbq-engine).

After Couchbase has been built, take the following steps:
1. Define the environment variable ENABLE_QUERY to be something (e.g. ENABLE_QUERY=Y)
2. Copy the cbq-engine binary to the install/bin directory of your Couchbase source
3. Start Couchbase: cd to the root directory of your Couchbase source and run ./install/bin/couchbase-server start

### Unit-tests

With Couchbase installed and running at 127.0.0.1:2181, clustering_cb unit tests can be run:

    $ cd $GOPATH/src/github.com/couchbaselabs/query/clustering/couchbase
    $ go test

