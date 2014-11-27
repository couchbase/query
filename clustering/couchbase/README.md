
## Running clustering_cb unit tests

In order to run the unit tests, it is necessary to install and start Couchbase.

### Couchbase Installation and Start-up

Install and start Couchbase:

+ [Install and build instructions](https://github.com/couchbase/tlm/)

Note: Get and build the sherlock branch, because this has support for managing the query engine (cbq-engine).

    $ repo init -u git://github.com/couchbase/manifest -m sherlock.xml
    $ repo sync
    $ make

After Couchbase has been built, take the following steps:

1. Start Couchbase: cd to the root directory of your Couchbase source and run ./install/bin/couchbase-server start

### Unit-tests

With Couchbase installed and running locally, clustering_cb unit tests can be run:

    $ cd $GOPATH/src/github.com/couchbaselabs/query/clustering/couchbase
    $ go test

