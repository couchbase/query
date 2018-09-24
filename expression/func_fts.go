//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"fmt"
	"net/url"
	"strings"
	//"sync"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
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
	FunctionBase
	myCurl   *Curl
	FtsCache []*FTSNode
	//ftsLock  sync.Mutex
	counter int64
}

func NewFTSQuery(operands ...Expression) Function {
	newC := &Curl{
		*NewFunctionBase("curl", operands...),
		nil,
	}
	rv := &FTSQuery{
		*NewFunctionBase("search_query", operands...),
		newC,
		make([]*FTSNode, 0),
		//sync.Mutex{},
		0,
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *FTSQuery) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *FTSQuery) Type() value.Type { return value.ARRAY }

func (this *FTSQuery) Evaluate(item value.Value, context Context) (value.Value, error) {
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

func (this *FTSQuery) Apply(context Context, args ...value.Value) (value.Value, error) {
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
	user = getCredentials(context)

	// If no host is given then get one
	// If the cache contains a list of FTS nodes already
	iter := 0
	for {
		if len(this.FtsCache) == 0 || errC != nil {
			// This is called when cache is empty
			err = this.PopulateFTSCache(context, user)
			if err != nil {
				return value.NULL_VALUE, err
			}
		}

		if hostname != "" {
			v1, user, err = ProcessHostname(this.FtsCache, hostname, idxName)
			if user == "" {
				user = getCredentials(context)
			}
		} else {
			// Round robin through the cache
			v1 = ftsUrl(this.FtsCache[this.counter].nodeIp, idxName, this.FtsCache[this.counter].portNo)
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
func (this *FTSQuery) Constructor() FunctionConstructor {
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
	if err != nil || newUrl != nil || newUrl.Hostname() == "" {
		return value.NULL_VALUE, "", err
	}

	// We restrict the user to provide hosts only on the current cluster.
	// Hence if this value is not in the cache we throw an error
	// Do a FTSCache lookup.
	port, isContain := searchCache(FtsCache, newUrl.Hostname())
	if !isContain {
		return value.NULL_VALUE, "", errors.NewNodeServiceErr(newUrl.Hostname())
	}

	if newUrl.Port() != "" && portStr(port) != newUrl.Port() {
		// Invalid hostname return error
		return value.NULL_VALUE, "", errors.NewFTSMissingPortErr(hostname)
	}

	// Complete input url by user.
	if newUrl.Port() != "" && newUrl.Path != "" && newUrl.User != nil {
		return value.NewValue(hostname), newUrl.User.String(), nil
	}

	// Incomplete URL given by user
	// 3. See if input hostname has a port
	if newUrl.Port() == "" && newUrl.Path != "" {
		// Input URL is correct but doesnt contain a port value. So the URL endpoint is incorrect
		// For eg "http://127.0.0.1/index/path"
		// This should throw a missing port error
		return value.NULL_VALUE, "", errors.NewFTSMissingPortErr("")
	} else if newUrl.Port() == "" && newUrl.Path == "" {
		hname = ftsUrl(newUrl.Hostname(), idxName, port)
	}

	// 4. See if it has user credentials
	if newUrl.User != nil {
		user = newUrl.User.String()
	}
	return hname, user, nil
}

func (this *FTSQuery) PopulateFTSCache(context Context, user string) error {
	// Make a call to nodeServices endpoint to get cluster info

	// Get url to call rest endpoint for nodeServices
	localhost := context.(CurlContext).DatastoreURL()

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
			if err != nil || newUrl.Hostname() == "" {
				return err
			}

			//this.ftsLock.Lock()
			this.FtsCache = append(this.FtsCache, &FTSNode{newUrl.Hostname(), fts.(value.NumberValue).Int64()})
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
func getCredentials(context Context) (user string) {
	// Get the credentials
	// If there are input credentials in the hostname then use those
	// Otherwise always use current credentials (the first one)
	up := context.(CurlContext).Credentials()
	if up == nil {
		up = context.(CurlContext).UrlCredentials()
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
