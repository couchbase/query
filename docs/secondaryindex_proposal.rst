files, prefixed with `secondary_*` implementing datastore.go and index.go
interfaces for secondary-index might reside under datastore/couchbase/
directory.

**primary index for couchbase datastore**

primary index can be build using views or can be built using
secondary-index and this can be passed as an option to couchbase:newKeyspace()
function. Based on this option datastore.CreatePrimaryIndex() can use
newPrimaryIndex() from views or newPrimaryIndex() from secondary to create
couchbase datastore's primary index.

**metadata repository**

secondary index will handle its own meta-data repository, which shall be a
distributed data store, and will be updated when ever CREATE INDEX, DELETE
INDEX statements are issued. It is also the reponsibility of
secondary-index to keep the meta-data information synchronised across
several N1QL servers.

- secondaryIndexes{} structure will hold one to a list of network-address for
  metadata respository.

- additionally each secondaryIndex{} structure will hold on to a single
  network-addres of the node that is hosting that secondaryIndex.

- both secondaryIndexes{} and secondaryIndex{} structures will be updated
  asynchronously, hence protected via sync.Mutex.

**secondary index**

since secondaryIndexes{} structure will updated asynchronously, I would
suggest couchbase Keyspace to use IndexIds(), IndexNames(), IndexById(),
IndexByName(), IndexByPrimary(), Indexes() method receivers on
secondaryIndexes{}

**secondary index coordinator**

- responsible to assign a unique value, called `defID` for every index
  created.

- responsible to notify metadata changes and topology changes to query
  server.

**consistent queries**

TBD

Following is a write up on consistent queries from secondary-index's
perspective.
https://github.com/couchbase/indexing/blob/master/secondary/docs/design/markdown/query.md
