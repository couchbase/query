Introduction
-------------
This is a basic introduction of a mapping of SQL concepts to N1QL. It is neither a tutorial nor a complete introduction to N1QL, and so is suitable for a casual reading. For anything more serious, please refer to [N1QL SELECT Specification](n1ql-select.md) and to the [N1QL Community page](http://query.couchbase.com).

Fundamental differences
------------------------
The most important difference versus traditional SQL and N1QL are not lingual but the data model. In traditional SQL, data is constrained to tables with a uniform structure, and many such tables exist.


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


In N1QL, the data exists as free form documents, gathered as large collections called buckets. There is no uniformity and there is no logical proximity of objects of the same data shape in a bucket.  

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

In N1QL, a query returns a set of documents. The returned document set may be uniform, but it need not be. Typically, specifying a SELECT with fixed set of attribute ('column') names results in a uniform set of documents. SELECT with a wildcard('*'') results in non-uniform result set. The only guarantee is that every returned document meets the query criteria.

    SELECT Name, History
    FROM HRData
    
    {
        'Name': Jamie,
        'History':
        [
          ['Yahoo', 2005, 2006],
          ['Oracle', 2006, 2012],
          ['Couchbase', 2012, null]
        ]
    }
    {
        'Name': 'Steve'
    }


Like SQL, N1QL allows renaming fields using the AS keyword. However, N1QL also allows reshaping the data, which has no analog in SQL. This is done by embedding the attributes of the statement in the desired result object shape.


    SELECT Name, History, {'FullTime': true} AS 'Status'
    FROM HRData
    
    {
        'Name': 'Jamie',
        'History':
        [
          ['Yahoo', 2005, 2006],
          ['Oracle', 2006, 2012],
          ['Couchbase', 2012, null]
        ],
        'Status': { 'FullTime': true }
    }
    {
        'Name': 'Steve',
        'Status': { 'FullTime': true }
    }


Selection differences
---------------------
The major difference between N1QL and SQL is that there are no tables in N1QL database. Hence, the FROM clause is used to select between data sources, i.e., buckets. If HRData is a bucket, the following will select the Name attribute from all documents in HRData bucket that have a Name attribute defined.

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
We colloquially categorize joins that one encounters in relational databases as "bad joins" and "good joins". Bad joins are what one would perform to compose an object that got shredded while being stored into tables. These account for bulk of the joins one would encounter with a business application running against a relational database. These joins are unnecessary in documenta databases that N1QL operates on because objects are stored as documents, where there is no shredding.

"Good joins" are where the relationship between the joined item has a real world representation as well. An example would be the relation between an expense report and a reimbursement check. In most business applications today, these joins are represented using application logic and not as database joins, as the latter is too restrictive to model the real world relationships.

### Join operations in N1QL

N1QL provides three kinds of join operations--join, nest, and unnest.

#### Joins

Joins in N1QL are similar to SQL, except that the join condition must be based on primary key lookups. The keyword KEYS is provided for specifying the join condition. If we had a second bucket whose primary keys were company names, the following query would return each employee's name and the address of his or her first job.

SELECT h.Name, firstjob.Address FROM HRData AS h JOIN Company AS firstjob KEYS [ h.History[0][0] ]

#### Nests

Nests in N1QL make use of the document model to provide another type of join operation. Whereas a standard join produces a result for every matching combination of left and right hand inputs, a nest produces a result for every left hand input. For each left hand input, the matching right hand inputs are collected into an array, whis is then embedded in the result. Like JOIN, NEST requires a KEYS clause.

The following query returns each employee's name and an array containing the addresses of all his or her jobs.

SELECT h.Name, jobAddress FROM HRData AS h NEST Company.Address AS jobAddress KEYS ARRAY hi[0] FOR hi IN h.History END

#### Unnests
An unnest is a trivial join where one part of the document is joined with another part of itself. It is largely a convenience construct. Unnests in N1QL are expressed on the "FROM" clause of the statement, similar to implicit inner joins in a SQL statement. Unlike standard joins, the joined parts of an unnest can refer only to sub parts of a single document and not across documents.

The unnest is effected using a new keyword, UNNEST.

    SELECT Name, Job FROM HRData UNNEST HRData.History AS Job

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

The UNNEST keyword does a cartesian product of the parent element with the child elements. This is not a join in the traditional SQL sense because at no point where two documents involved. The join result was generated from a single document.


Filtering differences
---------------------
N1QL uses WHERE clause similar to SQL. The '.' and the '[]' operator can be used for accessing nested and array elements, similar to usage in select clauses. 

A minor deviation is expression semantics. For example, undefined values are recognized as distinct from null and a complementary set of operators like IS MISSING are added in addition to standard operators like IS NULL. Furthermore, new conversions, for example from non-zero integer values to boolean value true, are supported as well. 

In general, most standard SQL functions like LOWER() are defined.

In addition to standard filtering predicates, two new operators are introduced, ANY and EVERY. These operators help in dealing with arrays in documents. The ANY will apply a filter on each element, and return true if any element meets the condition. EVERY does the same, except it returns true only if all elements matched the condition. These operators are used in conjuncction with SATISFIES and END.

    SELECT Name FROM HRData WHERE ANY h IN History SATISFIES h.Job[0] = 'Couchbase' END

    {
        'Name': Jamie
    }

In the above example, the condition clause Job[0] = 'Couchbase' is evaluated over the array element History. If any element of the history array has first element equal to Couchbase, the name attribute of the document is returned.


Aggregation and Grouping
------------------------
Standar aggregation operators, such as MIN, MAX, COUNT are defined and work similar to SQL. Grouping operators, GROUP BY and group filter HAVING are defined and behave similar to SQL equivalents.

