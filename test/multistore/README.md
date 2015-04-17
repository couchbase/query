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

The insert statements have the test_id appended to the document key. The documents also contain the test_id attribute that identifies which test that document belongs to. For example for the product bucket the document with key “product0_joins” has been inserted by the joins test. The value of that document will contain the field “test_id”: “joins”. All of the inserts happen on the 5 buckets, and hence the test_id was added to differentiate the data to be used by the tests for each functionality. (Both as an attribute in the document and withing the document key). 

### Adding new tests :
#### Adding a test query to an existing functionality test
* Go to the end of the existing statements within any case_*.json withing the tests ( .... } ] ) and add the new test statement as follows :
    .... } ,
    {
       "statements":" ... ",
       "results" : []
    } ]  

#### Adding a new functionality test 
* Create a new folder under query/test/multistore/test_cases/<new test>. (For eg test_cases/setop) 
* Copy the testcs_<> (and rename it. for eg testcs_setop) and testfs folders over from one of the other tests. 
* For the testcs_<new test> change the package name in both cs.go and cs_test.go. (For eg the package name will now be testcs_setop). Do not change the package name for the filestore test (testfs).
* For both testcs_<>/cs_test.go and testfs/fs_test.go, change the TestCleanupData function to delete the buckets pertaining to that test. Specify the test_id in the where clause (for setop the query will be: delete from product where test_id = "setop"). The test_id is the same as those present in the document data. (Also change the print statement in the TestInsertCaseFiles function to reflect the correct functionality).
* Create the insert.json file that contains all the create index and insert statements in the format given below :  
           [ 
            {
             "statements":"CREATE PRIMARY INDEX ON orders"
           }, 
           { 
             "statements":"INSERT INTO orders (KEY, VALUE) VALUES (...)"
           },
           {.......}
           ]
* Create the case_*.json tests that contain the actual queries to be tested. these should be in the following format :
           [
            {
               "statements": "SELECT DISTINCT custId FROM orders where test_id = \"agg_func\" ORDER BY custId",
               "results": [
                        {
            			"custId": "customer12"
        		}
    		  ]  },
              {
               "statements":" ... ",
               "results" : [ ] 
              } ...
          ]
    The tests will run the queries specified under statements and match them to the results given here. The strings in the select statements need to be escaped. (See test_id in the above example).
* The tests for this functionality have now been integrated into the unit test framework! 


