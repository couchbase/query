/*
Copyright 2014-Present Couchbase, Inc.

Use of this software is governed by the Business Source License included in
the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
file, in accordance with the Business Source License, use of this software will
be governed by the Apache License, Version 2.0, included in the file
licenses/APL2.txt.
*/

/*
NOTE:
-Receiver “this” is not a keyword in Go, but is defined that way so as to maintain consistency.
-Methods to access fields : In order to maintain encapsulation on value in the const struct to  prevent it from being set.
- According to the Go Docs A defer statement pushes a function call onto a list. The list of saved calls is executed after the surrounding function returns. Defer is commonly used to simplify functions that perform various clean-up actions.

The expression package provides expression evaluation for query and indexing. It imports the value code.

arith_* : Arithmetic terms allow for performing basic arithmetic within an expression. These arithmetic operators only operate on numbers. The standard addition, subtraction, multiplication, division, and modulo operators are supported. Additionally, a negation operator will change the sign of the expression. If either operand is not a number, it will evaluate to NULL.

base.go : It represents a base class for all expressions.

binding.go: It is a helper class.

case_* : There are two types of case expressions, searched and simple.  For the simple case there is a conditional matching within an expression. But for searched case expressions allow for conditional logic within an expression.

coll_* : Represents collection expressions EXISTS, IN, WITHIN, Range Transforms and Range Predicates. Range transforms are ARRAY, FIRST and OBJECT. They allow you to map and filter the elements or attributes of a collection or object(s). Range predicates are ANY (/SOME) and EVERY. They allow you to test a boolean condition over the elements or attributes of a collection or object(s). OBJECT has not been implemented as of yet.

comp_*: Comparison terms allow for comparing two expressions. Standard "equal", "not equal", "greater than", "greater than or equal", "less than", and "less than or equal" are supported. For equal (= and ==) and not equal (!= and <>) two forms are supported to aid in compatibility with other query languages.

N1QL specifies a total ordering for all values, across all types. This ordering is used by the comparison operators, and is described under ORDER BY. String comparison is done using a raw byte collation of UTF8 encoded strings (sometimes referred to as binary, C, or memcmp). This collation is case
sensitive. Case insensitive comparisons can be performed using UPPER() or LOWER() functions. Array and object comparisons are done as described under ORDER BY.

The LIKE operator allows for wildcard matching of string values. The right-hand side of the operator is a pattern, optionally containg '%' and '_' wildcard characters. Percent (%) matches any string of zero or more characters; underscore (_) matches any single character. The wildcards can be escaped by preceding them with a backslash (\). Backslash itself can also be escaped by preceding it with another backslash.

concat.go : If both operands are strings, the || operator concatenates these strings. Otherwise the expression evaluates to NULL.

cons_* : Represents construction expressions object and array. Objects and arrays can be constructed with arbitrary structure, nesting, and embedded expressions.

constant.go: Represents constant expressions using values and expression base. It inherits from expression base.

context.go: It imports the time package that provides the functionality to measure and display the time. (Refer to the time package and Time function from the GoLang docs)

expression.go:  Adding the expression type() allows you to know the schema or shape of the query without actually evaluating the query. As per the N1QL specs we can return the terminal identifier of an expression that is a path. It can be thought of an expression alias. For example if for the following select statement, b is the Alias. Select a.b.  When we equate two expressions, it is important to note that false negative are fine but false positives are not, and this needs to be enforced. The method SubsetOf  is not implemented yet. As of now it calls EquivalentTo. The Children method returns Expressions. For example if we had the expression tree for (a+b)*(c-d), the children for * are a+b and c-d, the children for + are a and b etc..

Formalizer : Takes an expression and converts it to its full equivalent form.

func_* : Function names are case-insensitive. See Appendices of the N1QL Specs for the supported functions. [https://github.com/couchbase/query/blob/master/docs/n1ql-select.md]

identifier.go :An identifier is a symbolic reference to a particular value in the current context.
An identifier can either be escaped or unescaped. Unescaped identifiers cannot support the full range of idenfiers allowed in a JSON document, but do support the most common ones with a simpler syntax. Escaped identifiers are surrounded with backticks and support all identifiers allowed in JSON. Using the backtick character within an escaped identifier can be accomplised by using two consecutive backtick characters.

Keywords cannot be escaped; therefore, escaped identifiers can overlap with keywords.

Index.go : Represents the Index context that uses the time package to get the time at that instant with nanosecond precision. Refer to golang docs for more details about the time package.

logic_* : Logical terms allow for combining other expressions using boolean logic. Standard AND, OR and NOT operators are supported.

mapper.go: It inherits from visitor and defines mapping from one expression to another.

nav_* : Nested expressions are used to access fields inside of objects and elements and slices inside of arrays. Navigation through fields of an object is supported using the dot (.) operator to access fields nested inside of other objects. The form .[expr] is used to access an object field named by evaluating the expr contained in the brackets. By default the names are case sensitive. In our example, the expressions address.city, address.`CITY`i, address.["ci" || "ty"], and address.["CI" || "TY"]i all evaluate to "Mountain View". Navigation through the elements of an array is supported using the bracket notation ([position]) to access elements inside an array. Negative positions are counted backwards from the end of the array. The form source-expr [ start : end ] is called array slicing. It returns a new array containing a subset of the source, containing the elements from position start to end-1. The element at start is included, while the element at end is not. If end is omitted, all elements from start to the end of the source array are included. Negative positions are counted backwards from the end of the array.

Parameter.go: It defines two types of parameters, named and positional. The named parameter is specified using formal param names in a query. The main advantage of a named parameter is that we dont have to remember the position of the parameter.  A positional parameter uses a position of the parameter in the query.

path.go: It defines the expression (alias) path.

stringer.go : Stringer implements the Visitor methods.

subquery.go : Used to define and subqueries.

visitor.go : The Gang of Four defines the Visitor as: "Represent an operation to be performed on elements of an object structure. Visitor lets you define a new operation without changing the classes of the elements on which it operates."  The type of Visitor is an interface with a list of methods that are implemented in Stringer.go. Named and Positional parameters are set by its index in the clause and by its name respectively. It is used to separate algorithm from an object structure on which it operates. This results in the ability to add new operations to the existing object structure.

stringer.go: Stringer implements the Visitor methods.
*/
package expression
