Introduction
-------------
This is a basic introduction of a mapping of SQL concepts to UnQL 2013. It is neither a tutorial nor a complete introduction to UnQL, and so is suitable for a casual reading. For anything more serious, please refer to [UnQL 2013 Specification](unql-2013.md) and the to be written tutorials.

Fundamental differences
------------------------
The most important difference versus traditional SQL and UnQL are not lingual but the data model. In traditional SQL, data is constrained to tables with a uniform structure, and many such tables exist.


    EMPLOYEE
    -----------------
    Name | SSN | Wage
    -----------------
    Jamie | 234 | 123 
    Steve | 123 | 456 
    -----------------
    
    SCHEMA:
    Name -> String of width 100
    SSN -> Number of width 9
    Wage -> Number of width 10

    EMPLOYERS:
    -----------------------------------
    Name_Key | Company   | Start | End 
    -----------------------------------
    Jamie     | Yahoo     | 2005  | 2006
    Jamie     | Oracle    | 2006  | 2012
    Jamie     | Couchbase | 2012  | NULL
    -----------------------------------


In UnQL, the data exists as free form documents, gathered as large collections called buckets. There is no uniformity and there is no logical proximity of objects of the same data shape in a bucket.  

    (HRData bucket)
    {
        'Name': 'Jamie'
        'SSN': 234
        'Wage': 123
        'History': 
        [
          ['Yahoo', 2005, 2006],
          ['Oracle', 2006, 2012],
          ['Couchbase', 2012, null]
        ]
    },
    {
        'Name': Steve
        'SSN': 123,
        'Wage': 456,
    }



Projection differences
----------------------
When one runs a query in SQL, a set of rows, consisting of one or more columns each is returned, and a header can be retrieved to obtain metadata about each column. It is generally not possible to get rowset where each row has a different set of columns.


    SELECT Name, Company 
    FROM Employee, Employers
    WHERE Name_Key = Name
    
    ----------------
    Name | Company
    ----------------
    Jamie | Oracle
    Jamie | Yahoo
    Jamie | Couchbase
    ----------------

In UnQL, a query returns a set of documents. The returned document set may be uniform, but it need not be. Typically, specifying a SELECT with fixed set of attribute ('column') names results in a uniform set of documents. SELECT with a wildcard('*'') results in non-uniform result set. The only guarantee is that every returned document meets the query criteria.

    SELECT Name, History
    FROM HRData
    
    {
        'Name': Jamie,
        'History':
        [
          ['Yahoo', 2005, 2006],
          ['Oracle', 2006, 2012],
          ['Couchbase', 2012]
        ]
    }
    {
        'Name': 'Steve'
    }


Like SQL, UnQL allows renaming fields using the AS keyword. However, UnQL also allows reshaping the data, which has no analog in SQL. This is done by embedding the attributes of the statement in the desired result object shape.


    SELECT Name, History, {'FullTime': true} AS 'Status'
    FROM HRData
    
    {
        'Name': 'Jamie',
        'History':
        [
          ['Yahoo', 2005, 2006],
          ['Oracle', 2006, 2012],
          ['Couchbase', 2012]
        ],
        'Status': { 'FullTime': true }
    }
    {
        'Name': 'Steve',
        'Status': { 'FullTime': true }
    }


Selection differences
---------------------
The major difference between UnQL and SQL is that there are no tables in UnQL database. Hence, the FROM clause is used to select between data sources, i.e., buckets. If HRData is a bucket, the following will select the Name attribute from all documents in HRData bucket that have a Name attribute defined.

    SELECT Name FROM HRData

A new twist is that each document can itself be regarded as a data source and the query run over its nested elements. Such nested elements are addressed using the '.' operator to descend a level, and the '[ ]' operator to index into an array element.

    SELECT FullTime FROM HRData.Status

    {
        'FullTime': true
    }


The selected fields can also be renamed using the AS oeprator, just like in SQL.


    SELECT firstjob FROM HRData.History[0] AS firstjob

    {
        'firstjob': ['Yahoo', 2005, 2006]
    }


    SELECT firstjob[2] FROM HRData.History[0] AS firstjob

    {
        'firstjob[2]': 2006
    }


Join differences
----------------
We colloquially categorize joins that one encounters in relational databases as "bad joins" and "good joins". Bad joins are what one would perform to compose an object that got shredded while being stored into tables. These account for bulk of the joins one would encounter with a business application running against a relational database. These joins are unnecessary in documenta databases that UnQL operates on because objects are stored as documents, where there is no shredding.

"Good joins" are where the relationship between the joined item has a real world representation as well. An example would be the relation between an expense report and a reimbursement check. In most business applications today, these joins are represented using application logic and not as database joins, as the latter is too restrictive to model the real world relationships.

### Joins in UnQL
The UnQL language in the first revision does not specify join behavior, except for self joins. It reserves all join related keywords so that the topic can be dealt with in a future version. Unlike SQL, UnQL is substantially usable without joins as there is no need to join tables to recompose documents.

### Self Joins
A self join is a trivial join where one part of the document is joined with another part of itself. It is largely a convenience construct. Self joins in UnQL are expressed on the "FROM" clause of the statement, similar to implicit inner joins in a SQL statement. Unlike SQL, the joined parts can refer only to sub parts of a single document and not across documents in this revision of UnQL.

The self join is effected using a new keyword, OVER.

    SELECT Name, Job FROM HRData OVER HRData.History AS Job

    {
        'Name': Jamie
        'Job': ['Yahoo', 2005, 2006]
    }
    {
        'Name': Jamie
        'Job': ['Oracle', 2006, 2012]
    }
    {
        'Name': Jamie
        'Job': ['Couchbase', 2012]
    }

The over keyword does a cartesian product of the parent element with the child elements. This is not a join in the traditional SQL sense because at no point where two documents involved. The join result was generated from a single document.


Filtering differences
---------------------
UnQL uses WHERE clause similar to SQL. The '.' and the '[]' operator can be used for accessing nested and array elements, similar to usage in select clauses. 

A minor deviation is that the expressions follow JavaScript semantcis. For example, undefined values are recognized as distinct from null and a complementary set of operators like IS MISSING are added in addition to standard operators like IS NULL. Further, JavaScript conversions, for example from non-zero integer values to boolean value true, are supported as well. 

In general, expressions behave as a subset of JavaScript expressions, and most standard SQL functions like LOWER() are defined.

In addition to standard filtering predicates, two new operators are introduced, ANY and ALL. These operators help in dealing with arrays in documents. The ANY will apply a filter on each element, and return true if any element meets the condition. The ALL does the same, except it reutrns true only if all elements matched the condition. These operators are used in conjection with OVER which specifies which sub element array to iterate over.

    SELECT Name WHERE ANY Job[0] = 'Couchbase' OVER History

    {
        'Name': Jamie
    }

In the above example, the condition clause Job[0] = 'Couchbase' is evaluated over the array element History. If any element of the history array has first element equal to Couchbase, the name attribute of the document is returned.


Aggregation and Grouping
------------------------
Standar aggregation operators, such as MIN, MAX, COUNT are defined and work similar to SQL. Grouping operators, GROUP BY and group filter HAVING are defined and behave similar to SQL equivalents.

