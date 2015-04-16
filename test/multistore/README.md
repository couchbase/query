## Using the Test framework :

### Prerequisites :

Couchbase server Installation with Data and Index service.  (Note: Change the allocated memory for the default bucket to 100 mb during setup)

The following values need to be updated to reflect the couchbase installation parameters :

* In json.go under query/test/multistore : 
	* Auth_param = "Administrator:password" : The username and password of the couchbase server installation. 
	* Pool_CBS = "127.0.0.1:8091/" : The IP and port number of the couchbase server installation. 

* In bucket_create.sh under query/test/multistore :
	*Site=http://127.0.0.1:8091/pools/default/buckets : The IP and port number of the couchbase server installation.
	*Auth=Administrator:password : The username and password of the couchbase server installation. 

### Steps :
* ./bucket_create.sh (Run only once at the beginning)  : Deletes the buckets first and then creates the buckets on the couchbase server for the cbserver tests and under test/multistore/data for the filestore.
* Run go test ./… from the query directory. 

### Description : 
The tests are organized into different directories based on functionality. Each test is self contained and does three things : Inserts the data into the buckets (both on couchbase server and the local datastore under test/multistore/data), runs the test queries and then deletes the data pertaining to that test identified by the test_id.

The insert statements have the test_id appended to the document key. The documents also contain the test_id attribute that identifies which test that document belongs to. For example for the product bucket the document with key “product0_joins” has been inserted by the joins test. The value of that document will contain the field “test_id”: “joins”. 



