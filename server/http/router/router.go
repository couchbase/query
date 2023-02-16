//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package router

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/couchbase/query/logging"
)

type Router interface {
	Map(path string, handler func(http.ResponseWriter, *http.Request), method ...string)
	MapPrefix(path string, handler func(http.ResponseWriter, *http.Request), method ...string)
	ServeHTTP(w http.ResponseWriter, req *http.Request)
	SetNotFoundHandler(h func(http.ResponseWriter, *http.Request))
}

const _INIT_MAPPINGS = 32

type variable struct {
	name   string
	suffix string
}

type route struct {
	prefix     string
	vars       []variable
	prefixOnly bool
	methods    map[string]bool
	handler    func(http.ResponseWriter, *http.Request)
}

func (this *route) setMethods(methods []string) {
	if len(methods) == 0 {
		this.methods = nil
	} else {
		this.methods = make(map[string]bool, len(methods))
		for _, m := range methods {
			this.methods[m] = true
		}
	}
}

func (this *route) sameVars(vars []variable) bool {
	if len(this.vars) != len(vars) {
		return false
	}
	for i := range this.vars {
		if this.vars[i].name != vars[i].name || this.vars[i].suffix != vars[i].suffix {
			return false
		}
	}
	return true
}

func (this *route) match(req *http.Request, vars *map[string]string) bool {
	if len(this.methods) > 0 {
		if _, ok := this.methods[req.Method]; !ok {
			return false
		}
	}
	if !strings.HasPrefix(req.URL.Path, this.prefix) {
		return false
	}
	if this.prefixOnly {
		return true
	}
	if len(this.vars) == 0 {
		return len(this.prefix) == len(req.URL.Path)
	}
	n := len(this.prefix)
	var i int
	for i = 0; i < len(this.vars); i++ {
		var val string
		if len(this.vars[i].suffix) == 0 {
			val = req.URL.Path[n:]
			n = len(req.URL.Path)
		} else {
			e := strings.Index(req.URL.Path[n:], this.vars[i].suffix)
			if e == -1 {
				return false
			}
			val = req.URL.Path[n : n+e]
			n = n + e + len(this.vars[i].suffix)
		}
		if -1 != strings.IndexByte(val, '/') {
			return false
		}
		if *vars == nil {
			*vars = make(map[string]string, 2)
		}
		(*vars)[this.vars[i].name] = val
	}
	return i == len(this.vars) && n == len(req.URL.Path)
}

type routerImpl struct {
	sync.RWMutex
	routes   []*route
	notFound func(http.ResponseWriter, *http.Request)
}

func NewRouter() *routerImpl {
	rv := &routerImpl{
		routes:   make([]*route, 0, _INIT_MAPPINGS),
		notFound: http.NotFoundHandler().ServeHTTP,
	}
	return rv
}

func (this *routerImpl) MapPrefix(path string, handler func(http.ResponseWriter, *http.Request), methods ...string) {
	this.Lock()
	for _, r := range this.routes {
		if r.prefixOnly == true && r.prefix == path {
			r.setMethods(methods)
			r.handler = handler
			this.Unlock()
			return
		}
	}
	r := &route{
		prefix:     path,
		prefixOnly: true,
		handler:    handler,
	}
	r.setMethods(methods)
	this.routes = append(this.routes, r)
	this.Unlock()
}

func (this *routerImpl) Map(path string, handler func(http.ResponseWriter, *http.Request), methods ...string) {
	var prefix, name, suffix string
	var vars []variable

	n := strings.IndexByte(path, '{')
	if n == -1 {
		prefix = path
	} else {
		prefix = path[:n]
		for n < len(path) {
			n++
			e := strings.IndexByte(path[n:], '}')
			if e == -1 {
				logging.Debugf("Invalid path: missing '}': %v", path)
				return
			}
			name = path[n : n+e]
			n += e + 1
			if len(path) <= n {
				suffix = ""
			} else {
				i := strings.IndexByte(path[n:], '{')
				if i == -1 {
					suffix = path[n:]
					n += len(suffix)
				} else {
					suffix = path[n : n+i]
					n += i
				}
			}
			vars = append(vars, variable{name: name, suffix: suffix})
		}
	}

	this.Lock()
	for _, r := range this.routes {
		if r.prefixOnly == false && r.prefix == prefix && r.sameVars(vars) {
			r.setMethods(methods)
			r.handler = handler
			this.Unlock()
			return
		}
	}
	r := &route{
		prefix:     prefix,
		prefixOnly: false,
		vars:       vars,
		handler:    handler,
	}
	r.setMethods(methods)
	this.routes = append(this.routes, r)
	this.Unlock()
}

func (this *routerImpl) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var h func(http.ResponseWriter, *http.Request)
	var vars map[string]string

	this.RLock()
	for _, r := range this.routes {
		if r.match(req, &vars) {
			h = r.handler
			ctx := req.Context()
			for k, v := range vars {
				ctx = context.WithValue(ctx, k, v)
			}
			vars = nil
			req = req.WithContext(ctx)
			break
		}
		vars = nil
	}
	this.RUnlock()
	if h == nil {
		h = this.notFound
	}

	h(w, req)
}

func (this *routerImpl) SetNotFoundHandler(h func(http.ResponseWriter, *http.Request)) {
	this.notFound = h
}

func RequestValue(req *http.Request, v string) (bool, string) {
	if val := req.Context().Value(v); val != nil {
		return true, val.(string)
	}
	return false, ""
}
