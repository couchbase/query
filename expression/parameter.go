//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

/*
Type NamedParameter is an interface that inherits from 
Expression, and has a method Name() that returns a 
string. It defines a Named Parameter, that is specified
using formal param names in a query. The main advantage 
of a named parameter is that we dont have to remember 
the position of the parameter.
*/
type NamedParameter interface {
	Expression
	Name() string
}

/*
Type PositionalParameter is an interface that inherits
from Expression, and has a method position that returns
an integer representing the position of the parameter 
in the query. 
*/
type PositionalParameter interface {
	Expression
	Position() int
}
