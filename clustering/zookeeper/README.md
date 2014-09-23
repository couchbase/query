
## Running clustering_zk unit tests

In order to run the unit tests, it is necessary to get the external go-zookeeper package and install and start zookeeper.

### Package go-zookeeper 

The clustering_zk implementation depends on an external [Go Zookeeper Client](https://github.com/samuel/go-zookeeper)

To update your repository with this package:

    $ cd $GOPATH/src/github.com/couchbaselabs/query
    $ go get -d -v ./...

### Zookeeper Installation and Start-up

Install and start zookeeper:

+ [Mac OS specific instructions](http://blog.kompany.org/2013/02/23/setting-up-apache-zookeeper-on-os-x-in-five-minutes-or-less/)
+ [Apache Zookeeper Getting Started Guide](http://zookeeper.apache.org/doc/r3.1.2/zookeeperStarted.html)

### Unit-tests

With zookeeper installed and running at localhost:2181, clustering_zk unit tests can be run:

    $ cd $GOPATH/src/github.com/couchbaselabs/query/clustering/zookeeper
    $ go test

