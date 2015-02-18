
## Running clustering_cb unit tests

In order to run the unit tests, it is necessary to have a Couchbase instance running.

### Couchbase Installation and Start-up

Install and start Couchbase:

+ [Install and build instructions](https://github.com/couchbase/tlm/)

Get and build the sherlock branch,  this has support for the query engine (cbq-engine).

    $ repo init -u git://github.com/couchbase/manifest -m sherlock.xml
    $ repo sync
    $ make
    $ ./install/bin/couchbase-server start

### Unit-tests

With Couchbase installed and running locally, run the unit test:

    $ cd $GOPATH/src/github.com/couchbase/query/clustering/couchbase
    $ go test

To test against a Couchbase instance running in a different location than localhost, edit the couchbase_location parameter in clustering_cb_test.go.
