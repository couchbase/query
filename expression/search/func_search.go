//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package search

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
N1QL FTS integration using a table function SEARCH_QUERY
It returns an array of meta().id that qualifies the search criteria.
The input is an FTS search JSON object and the FTS index to be used.
There is an optional argument to input the FTS node to use (IP or hostname)
*/

const (
	_SERVICES_PATH = "/pools/default/nodeServices"
	_FTS_PATH      = "/api/index/"
	_QUERY_PATH    = "/query"
)

type FTSNode struct {
	nodeIp string
	portNo int64
}

type FTSQuery struct {
	expression.FunctionBase
	myCurl   *expression.Curl
	FtsCache []*FTSNode
	//ftsLock  sync.Mutex
	counter int64
}

func NewFTSQuery(operands ...expression.Expression) expression.Function {
	rv := &FTSQuery{
		*expression.NewFunctionBase("search_query", operands...),
		expression.NewCurl(operands...).(*expression.Curl),
		make([]*FTSNode, 0),
		//sync.Mutex{},
		0,
	}

	rv.SetExpr(rv)
	return rv
}

/*
Visitor pattern.
*/
func (this *FTSQuery) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *FTSQuery) Type() value.Type { return value.ARRAY }

func (this *FTSQuery) Evaluate(item value.Value, context expression.Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *FTSQuery) Privileges() *auth.Privileges {
	unionPrivileges := auth.NewPrivileges()
	unionPrivileges.Add("", auth.PRIV_QUERY_EXTERNAL_ACCESS)

	children := this.Children()
	for _, child := range children {
		unionPrivileges.AddAll(child.Privileges())
	}

	return unionPrivileges
}

func (this *FTSQuery) Apply(context expression.Context, args ...value.Value) (value.Value, error) {
	var err, errC error
	hostname := ""
	user := ""
	v1 := value.EMPTY_STRING_VALUE
	v := value.EMPTY_ARRAY_VALUE

	for k, arg := range args {
		if arg.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		}
		if arg.Type() == value.NULL {
			return value.NULL_VALUE, nil
		}
		if k == 1 {
			if arg.Type() != value.OBJECT {
				return value.NULL_VALUE, nil
			}
		} else {
			if arg.Type() != value.STRING {
				return value.NULL_VALUE, nil
			}
			if k == 2 {
				hostname = arg.Actual().(string)
			}
		}
	}

	search := args[1].String()
	idxName := args[0].Actual().(string)

	// If no host is given then get one
	// If the cache contains a list of FTS nodes already
	iter := 0
	for {
		if len(this.FtsCache) == 0 || errC != nil {
			// Get current node auth credentials
			user = this.getCredentials(context, "")
			// This is called when cache is empty
			err = this.PopulateFTSCache(context, user)
			if err != nil {
				return value.NULL_VALUE, err
			}
		}

		if hostname != "" {
			v1, user, err = ProcessHostname(this.FtsCache, hostname, idxName)
			if user == "" {
				hname, _ := url.Parse(v1.String())
				user = this.getCredentials(context, hname.Hostname()+":"+hname.Port())
			}
		} else {
			// Round robin through the cache
			host := this.FtsCache[this.counter].nodeIp
			port := this.FtsCache[this.counter].portNo
			v1 = ftsUrl(host, idxName, port)
			user = this.getCredentials(context, host+":"+portStr(port))
			this.counter = (this.counter + 1) % int64(len(this.FtsCache))
		}

		//ftsUrlPath := "api/index/" + idxName + "/query"
		//"http://127.0.0.1:8094/api/index/" + idxName + "/query"

		newM := getMap(user, search, "POST", "Content-Type: application/json")
		v2 := value.NewValue(newM)

		// Create the CURL request
		v, errC = this.myCurl.Apply(context, v1, v2)
		if errC != nil && len(args) == 3 {
			// Hostname was given. Directly throw err
			break
		}
		if iter > 1 {
			break
		}
		iter = iter + 1
	}

	if errC != nil {
		return value.NULL_VALUE, errC
	}
	return v, nil
}

/*
Minimum input arguments required is 2.
*/
func (this *FTSQuery) MinArgs() int { return 2 }

func (this *FTSQuery) Indexable() bool {
	return false
}

/*
Maximum input arguments allowed.
*/
func (this *FTSQuery) MaxArgs() int { return 3 }

/*
Factory method pattern.
*/
func (this *FTSQuery) Constructor() expression.FunctionConstructor {
	return NewFTSQuery
}

/*
Input - hostname
Output - final host string
Constructs a hostname that is valid.
*/
func ProcessHostname(FtsCache []*FTSNode, hostname, idxName string) (hname value.Value, user string, err error) {
	// User can input hostname , hostname + port , User Info + hostname + port , full url, part url
	// In case we have an input hostname and we need to discover the port
	// Check if the port has already been input

	// 1. Make sure the input hostname has scheme
	// TODO - What if url has a different scheme

	if !strings.HasPrefix(hostname, "http://") && !strings.HasPrefix(hostname, "https://") &&
		!strings.HasPrefix(hostname, "couchbase://") && !strings.HasPrefix(hostname, "couchbases://") {
		hostname = "http://" + hostname
	}

	// 2. Make sure it is a valid hostname
	newUrl, err := url.Parse(hostname)

	if err != nil || newUrl != nil || hostname_go(newUrl) == "" {
		return value.NULL_VALUE, "", err
	}

	hostN := hostname_go(newUrl)
	portN := port_go(newUrl)
	// We restrict the user to provide hosts only on the current cluster.
	// Hence if this value is not in the cache we throw an error
	// Do a FTSCache lookup.
	port, isContain := searchCache(FtsCache, hostN)
	if !isContain {
		return value.NULL_VALUE, "", errors.NewNodeServiceErr(hostN)
	}

	if portN != "" && portStr(port) != portN {
		// Invalid hostname return error
		return value.NULL_VALUE, "", errors.NewFTSMissingPortErr(hostname)
	}

	// Complete input url by user.
	if portN != "" && newUrl.Path != "" && newUrl.User != nil {
		return value.NewValue(hostname), newUrl.User.String(), nil
	}

	// Incomplete URL given by user
	// 3. See if input hostname has a port
	if portN == "" && newUrl.Path != "" {
		// Input URL is correct but doesnt contain a port value. So the URL endpoint is incorrect
		// For eg "http://127.0.0.1/index/path"
		// This should throw a missing port error
		return value.NULL_VALUE, "", errors.NewFTSMissingPortErr("")
	} else if portN == "" && newUrl.Path == "" {
		hname = ftsUrl(hostN, idxName, port)
	}

	// 4. See if it has user credentials
	if newUrl.User != nil {
		user = newUrl.User.String()
	}
	return hname, user, nil
}

func (this *FTSQuery) PopulateFTSCache(context expression.Context, user string) error {
	// Make a call to nodeServices endpoint to get cluster info

	// Get url to call rest endpoint for nodeServices
	localhost := context.(expression.CurlContext).DatastoreURL()

	// Request to get list of FTS nodes in the cluster
	nv := localhost + _SERVICES_PATH
	res, err := this.myCurl.Apply(context, value.NewValue(nv), value.NewValue(getMap(user, "", "", "")))
	if err != nil {
		return err
	}

	// nodes is an array of objects that contains node information
	// within each obj, hostname string represents the datastore url
	// and services is an array listing the services
	listofNodes, ok := res.Field("nodesExt")
	if !ok {
		return errors.NewNodeInfoAccessErr("nodesExt")
	}

	//Reset the counter
	//this.ftsLock.Lock()
	this.counter = 0
	this.FtsCache = nil
	//this.ftsLock.Unlock()

	// JSON object is different depending on number of nodes in the system
	arr := listofNodes.Actual().([]interface{})
	if len(arr) == 1 {

		// This is a localhost configuration and there is no hostname field in nodesExt
		// We need to check if thisnode = true
		el := value.NewValue(arr[0])
		thisNode, ok := el.Field("thisNode")
		if !ok {
			return errors.NewNodeInfoAccessErr("nodesExt/thisNode")
		}

		if thisNode.Actual().(bool) {
			// Input hostname is localhost
			// FTS node is same as hostname
			services, ok := el.Field("services")
			if !ok {
				return errors.NewNodeInfoAccessErr("nodesExt/services")
			}
			fts, ok := services.Field("fts")
			if !ok {
				return errors.NewNodeServiceErr("fts")
			}

			// 2. Make sure it is a valid hostname
			newUrl, err := url.Parse(localhost)
			if err != nil || hostname_go(newUrl) == "" {
				return err
			}

			//this.ftsLock.Lock()
			this.FtsCache = append(this.FtsCache, &FTSNode{hostname_go(newUrl), fts.(value.NumberValue).Int64()})
			//this.ftsLock.Unlock()

			return nil
		}
		return errors.NewNodeServiceErr(localhost)
	}

	// If it comes here then we have more than 1 node.
	// In this case hostnames are given as part of the REST API result
	for _, node := range arr {
		// node is an object
		el := value.NewValue(node)
		services, ok := el.Field("services")
		if !ok {
			return errors.NewNodeInfoAccessErr("nodesExt/services")
		}
		// If no fts node then dont add to the cache.
		fts, ok := services.Field("fts")
		if !ok {
			continue
		}
		host, ok := el.Field("hostname")
		if !ok {
			return errors.NewNodeInfoAccessErr("nodesExt/hostname")
		}

		// This is an FTS node. Get Hostname and Port num
		//this.ftsLock.Lock()
		this.FtsCache = append(this.FtsCache, &FTSNode{host.Actual().(string), fts.ActualForIndex().(int64)})
		//this.ftsLock.Unlock()
	}

	return nil
}

func ftsUrl(nodeIp, index string, portNo int64) value.Value {
	return value.NewValue("http://" + nodeIp + ":" + portStr(portNo) + _FTS_PATH + index + _QUERY_PATH)
}
func (this *FTSQuery) getCredentials(context expression.Context, hname string) (user string) {
	// Get the credentials
	// If there are input credentials in the hostname then use those
	// Otherwise always use current credentials (the first one)
	// This depends on which node we are sending the request to
	up := context.(expression.CurlContext).Credentials()
	if up == nil {
		up = context.(expression.CurlContext).UrlCredentials(hname)
	}
	for i, k := range up {
		if i != "" && k != "" {
			user = i + ":" + k
			break
		}
	}

	return
}

func getMap(user, data, request, header string) map[string]interface{} {
	// Map to deal with options passed on to curl command.
	newM := map[string]interface{}{}
	newM["user"] = user
	if data != "" {
		newM["data"] = data
	}
	if request != "" {
		newM["request"] = request

	} else {
		newM["get"] = true
	}
	if header != "" {
		newM["header"] = header
	}
	return newM
}

func portStr(portNo int64) string {
	return fmt.Sprintf("%v", portNo)
}

func searchCache(arr []*FTSNode, val string) (int64, bool) {
	for _, f := range arr {
		if f.nodeIp == val {
			return f.portNo, true
		}
	}
	return 0, false
}

/*
Adding code from golangs functions for HOSTNAME AND PORT until indexing upgrades to go 1.8 +
*/
// Hostname returns u.Host, without any port number.
//
// If Host is an IPv6 literal with a port number, Hostname returns the
// IPv6 literal without the square brackets. IPv6 literals may include
// a zone identifier.
func hostname_go(u *url.URL) string {
	return stripPort(u.Host)
}

// Port returns the port part of u.Host, without the leading colon.
// If u.Host doesn't contain a port, Port returns an empty string.
func port_go(u *url.URL) string {
	return portOnly(u.Host)
}

func stripPort(hostport string) string {
	colon := strings.IndexByte(hostport, ':')
	if colon == -1 {
		return hostport
	}
	if i := strings.IndexByte(hostport, ']'); i != -1 {
		return strings.TrimPrefix(hostport[:i], "[")
	}
	return hostport[:colon]
}

func portOnly(hostport string) string {
	colon := strings.IndexByte(hostport, ':')
	if colon == -1 {
		return ""
	}
	if i := strings.Index(hostport, "]:"); i != -1 {
		return hostport[i+len("]:"):]
	}
	if strings.Contains(hostport, "]") {
		return ""
	}
	return hostport[colon+len(":"):]
}

///////////////////////////////////////////////////
//
//        FTS SEARCH function
//
///////////////////////////////////////////////////

type SearchVerify interface {
	Evaluate(item value.Value) (bool, errors.Error)
}

type Search struct {
	expression.FunctionBase
	keyspacePath string
	verify       SearchVerify
	err          error
}

func NewSearch(operands ...expression.Expression) expression.Function {
	rv := &Search{}
	rv.FunctionBase = *expression.NewFunctionBase("search", operands...)
	rv.SetExpr(rv)
	return rv
}

func (this *Search) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Search) Type() value.Type                           { return value.BOOLEAN }
func (this *Search) MinArgs() int                               { return 2 }
func (this *Search) MaxArgs() int                               { return 3 }
func (this *Search) Indexable() bool                            { return false }
func (this *Search) DependsOn(other expression.Expression) bool { return false }

func (this *Search) CoveredBy(keyspace string, exprs expression.Expressions,
	options expression.CoveredOptions) expression.Covered {

	if this.KeyspaceAlias() != keyspace {
		return expression.CoveredSkip
	}

	for _, expr := range exprs {
		if this.EquivalentTo(expr) {
			return expression.CoveredTrue
		}
	}

	return expression.CoveredFalse
}

func (this *Search) Evaluate(item value.Value, context expression.Context) (value.Value, error) {
	if this.verify == nil {
		return value.FALSE_VALUE, this.err
	}

	// Evaluate document for keyspace. If MISSING or NULL return (For OUTER Join)
	val, err := this.Keyspace().Evaluate(item, context)
	if err != nil || val.Type() <= value.NULL {
		return val, err
	}

	cond, err := this.verify.Evaluate(val)
	if err != nil || !cond {
		return value.FALSE_VALUE, err
	}

	return value.TRUE_VALUE, nil

}

/*
Factory method pattern.
*/
func (this *Search) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewSearch(operands...)
	}
}

func (this *Search) SetVerify(v SearchVerify, err error) {
	this.verify = v
	this.err = err
}

func (this *Search) Keyspace() *expression.Identifier {
	return expression.NewIdentifier(this.KeyspaceAlias())
}

func (this *Search) KeyspaceAlias() string {
	s, _, _ := expression.PathString(this.Operands()[0])
	return s
}

func (this *Search) KeyspacePath() string {
	return this.keyspacePath
}

func (this *Search) SetKeyspacePath(path string) {
	this.keyspacePath = path
}

func (this *Search) FieldName() string {
	_, s, _ := expression.PathString(this.Operands()[0])
	return s
}

func (this *Search) Query() expression.Expression {
	return this.Operands()[1]
}

func (this *Search) Options() expression.Expression {
	if len(this.Operands()) > 2 {
		return this.Operands()[2]
	}

	return nil
}

func (this *Search) IndexName() (name string) {
	name, _, _ = this.getIndexNameAndOutName(this.Options())
	return
}

func (this *Search) OutName() string {
	_, outName, _ := this.getIndexNameAndOutName(this.Options())
	if outName == "" {
		outName = expression.DEF_OUTNAME
	}

	return outName
}

func (this *Search) IndexMetaField() expression.Expression {
	return expression.NewField(this.Keyspace(), expression.NewFieldName(this.OutName(), false))

}

func (this *Search) getIndexNameAndOutName(arg expression.Expression) (index, outName string, err error) {
	if arg == nil {
		return
	}
	options := arg.Value()
	if options == nil {
		if oc, ok := arg.(*expression.ObjectConstruct); ok {
			for name, val := range oc.Mapping() {
				n := name.Value()
				if n == nil || n.Type() != value.STRING {
					continue
				}

				if n.Actual().(string) == "index" {
					v := val.Value()
					if v == nil || (v.Type() != value.STRING && v.Type() != value.OBJECT) {
						err = fmt.Errorf("%s() not valid third argument: %v", this.Name(),
							arg.String())
						return

					}
					index, _ = v.Actual().(string)
				}

				if n.Actual().(string) == "out" {
					v := val.Value()
					if v == nil || v.Type() != value.STRING {
						err = fmt.Errorf("%s() not valid third argument: %v", this.Name(),
							arg.String())
						return
					}
					outName, _ = v.Actual().(string)
				}
			}
		}
	} else if options.Type() == value.OBJECT {
		if val, ok := options.Field("index"); ok {
			if val == nil || (val.Type() != value.STRING && val.Type() != value.OBJECT) {
				err = fmt.Errorf("%s() not valid third argument: %v", this.Name(), arg.String())
				return
			}
			index, _ = val.Actual().(string)
		}

		if val, ok := options.Field("out"); ok {
			if val == nil || val.Type() != value.STRING {
				err = fmt.Errorf("%s() not valid third argument: %v", this.Name(), arg.String())
				return
			}
			outName, _ = val.Actual().(string)
		}
	}

	return

}

func (this *Search) ValidOperands() error {
	op := this.Operands()[0]
	a, _, e := expression.PathString(op)
	if a == "" || e != nil {
		return fmt.Errorf("%s() not valid first argument: %s", this.Name(), op.String())
	}

	op = this.Query()
	val := op.Value()
	if (val != nil && val.Type() != value.STRING && val.Type() != value.OBJECT) || op.Static() == nil {
		return fmt.Errorf("%s() not valid second argument: %s", this.Name(), op.String())
	}

	_, _, err := this.getIndexNameAndOutName(this.Options())
	return err
}

type SearchMeta struct {
	expression.FunctionBase
	keyspace *expression.Identifier
	field    *expression.Field
	second   value.Value
}

func NewSearchMeta(operands ...expression.Expression) expression.Function {
	rv := &SearchMeta{}
	rv.FunctionBase = *expression.NewFunctionBase("search_meta", operands...)
	rv.SetExpr(rv)
	return rv
}

func (this *SearchMeta) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *SearchMeta) Type() value.Type                           { return value.OBJECT }
func (this *SearchMeta) MinArgs() int                               { return 0 }
func (this *SearchMeta) MaxArgs() int                               { return 1 }
func (this *SearchMeta) Indexable() bool                            { return false }
func (this *SearchMeta) DependsOn(other expression.Expression) bool { return false }

func (this *SearchMeta) CoveredBy(keyspace string, exprs expression.Expressions,
	options expression.CoveredOptions) expression.Covered {

	if this.KeyspaceAlias() != keyspace {
		return expression.CoveredSkip
	}

	for _, expr := range exprs {
		if this.EquivalentTo(expr) {
			return expression.CoveredTrue
		}
	}

	return expression.CoveredFalse
}

func (this *SearchMeta) Keyspace() *expression.Identifier {
	op := this.Operands()[0]
	switch op := op.(type) {
	case *expression.Identifier:
		return op
	case *expression.Field:
		keyspace, _ := op.First().(*expression.Identifier)
		return keyspace
	default:
		return nil
	}
}

func (this *SearchMeta) KeyspaceAlias() string {
	keyspace := this.Keyspace()
	if keyspace != nil {
		return keyspace.Alias()
	}
	return ""
}

func (this *SearchMeta) Evaluate(item value.Value, context expression.Context) (value.Value, error) {
	if this.keyspace == nil {

		// Transform argument FROM ks.idxname TO META(ks).idxname
		this.keyspace = this.Keyspace()
		if this.keyspace == nil {
			return value.NULL_VALUE, nil
		}

		op := this.Operands()[0]

		if field, ok := op.(*expression.Field); ok {
			if _, ok = field.First().(*expression.Identifier); !ok {
				return value.NULL_VALUE, nil
			}
			this.second = field.Second().Value()
			this.field = expression.NewField(nil, field.Second())
		}
	}

	val, err := this.getSmeta(this.keyspace, item, context)
	if err != nil {
		return value.NULL_VALUE, err
	}

	if this.field != nil {
		return this.field.Apply(context, val, this.second)
	} else {
		return val, err
	}
}

func (this *SearchMeta) getSmeta(keyspace *expression.Identifier, item value.Value,
	context expression.Context) (value.Value, error) {

	if keyspace == nil {
		return value.NULL_VALUE, nil
	}

	val, err := keyspace.Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	if val.Type() == value.MISSING {
		return val, nil
	}

	switch val := val.(type) {
	case value.AnnotatedValue:
		return value.NewValue(val.GetAttachment("smeta")), nil
	default:
		return value.NULL_VALUE, nil
	}
}

/*
Factory method pattern.
*/
func (this *SearchMeta) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewSearchMeta(operands...)
	}
}

type SearchScore struct {
	expression.FunctionBase
	score expression.Expression
}

func NewSearchScore(operands ...expression.Expression) expression.Function {
	rv := &SearchScore{}
	rv.FunctionBase = *expression.NewFunctionBase("search_score", operands...)

	rv.SetExpr(rv)
	return rv
}

func (this *SearchScore) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *SearchScore) Type() value.Type                           { return value.NUMBER }
func (this *SearchScore) MinArgs() int                               { return 0 }
func (this *SearchScore) MaxArgs() int                               { return 1 }
func (this *SearchScore) Indexable() bool                            { return false }
func (this *SearchScore) DependsOn(other expression.Expression) bool { return false }
func (this *SearchScore) IndexMetaField() expression.Expression      { return this.Operands()[0] }

func (this *SearchScore) CoveredBy(keyspace string, exprs expression.Expressions,
	options expression.CoveredOptions) expression.Covered {

	if this.KeyspaceAlias() != keyspace {
		return expression.CoveredSkip
	}

	for _, expr := range exprs {
		if this.EquivalentTo(expr) {
			return expression.CoveredTrue
		}
	}

	return expression.CoveredFalse
}

func (this *SearchScore) Keyspace() *expression.Identifier {
	op := this.Operands()[0]
	switch op := op.(type) {
	case *expression.Identifier:
		return op
	case *expression.Field:
		keyspace, _ := op.First().(*expression.Identifier)
		return keyspace
	default:
		return nil
	}
}

func (this *SearchScore) KeyspaceAlias() string {
	keyspace := this.Keyspace()
	if keyspace != nil {
		return keyspace.Alias()
	}
	return ""
}

func (this *SearchScore) Evaluate(item value.Value, context expression.Context) (value.Value, error) {
	if this.score == nil {
		// Transform argument FROM ks.idxname TO META(ks).idxname.score
		this.score = expression.NewField(NewSearchMeta(this.Operands()...),
			expression.NewFieldName("score", false))
	}
	return this.score.Evaluate(item, context)
}

/*
Factory method pattern.
*/
func (this *SearchScore) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewSearchScore(operands...)
	}
}
