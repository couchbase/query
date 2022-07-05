// Copyright 2020-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
// file, in accordance with the Business Source License, use of this software
// will be governed by the Apache License, Version 2.0, included in the file
// licenses/APL2.txt.
//
// Currently, the community edition does not have access to update
// statistics, so this stub returns an error.

//go:build enterprise

package couchbase

import (
	"time"

	"github.com/couchbase/query-ee/dictionary"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

const (
	_SYSTEM_SCOPE      = dictionary.SYSTEM_SCOPE
	_SYSTEM_COLLECTION = dictionary.SYSTEM_COLLECTION
	_SYSTEM_PRIM_INDEX = dictionary.SYSTEM_PRIM_INDEX
)

func (s *store) CreateSysPrimaryIndex(idxName, requestId string, indexer3 datastore.Indexer3) errors.Error {
	// get number of index nodes in the cluster, and create the primary index
	// with replicas in the following fashion:
	//    numIndexNode >= 4    ==> num_replica = 2
	//    numIndexNode >  1    ==> num_replica = 1
	//    numIndexNode == 1    ==> no replica
	numIndexNode, errs := s.getNumIndexNode()
	if len(errs) > 0 {
		return errs[0]
	}

	var with value.Value
	var replica map[string]interface{}
	num_replica := 0
	if numIndexNode >= 4 {
		num_replica = 2
	} else if numIndexNode > 1 {
		num_replica = 1
	}
	if num_replica > 0 {
		replica = make(map[string]interface{}, 1)
		replica["num_replica"] = num_replica
		with = value.NewValue(replica)
	}

	_, er := indexer3.CreatePrimaryIndex3(requestId, idxName, nil, with)
	if er != nil && er.Code() != errors.E_INDEX_ALREADY_EXISTS {
		// if the create failed due to not enough indexer nodes, retry with
		// less number of replica
		for num_replica > 0 {
			// defined as ErrNotEnoughIndexers in indexing/secondary/common/const.go
			if !er.ContainsText("not enough indexer nodes to create index with replica") {
				return er
			}

			num_replica--
			if num_replica == 0 {
				with = nil
			} else {
				replica["num_replica"] = num_replica
				with = value.NewValue(replica)
			}

			// retry with less number of replica
			_, er = indexer3.CreatePrimaryIndex3(requestId, idxName, nil, with)
			if er == nil || er.Code() == errors.E_INDEX_ALREADY_EXISTS {
				break
			}
		}
		if er != nil && er.Code() != errors.E_INDEX_ALREADY_EXISTS {
			return er
		}
	}

	var sysIndex datastore.Index
	maxRetry := 8
	interval := 250 * time.Millisecond
	for i := 0; i < maxRetry; i++ {
		time.Sleep(interval)
		interval *= 2

		sysIndex, er = indexer3.IndexByName(idxName)
		if sysIndex != nil {
			break
		} else if er != nil && er.Code() != errors.E_CB_INDEX_NOT_FOUND {
			return er
		}
	}

	return nil
}

func (s *store) GetSystemCollection(bucketName string) (datastore.Keyspace, errors.Error) {
	return datastore.GetKeyspace("default", bucketName, _SYSTEM_SCOPE, _SYSTEM_COLLECTION)
}

func (s *store) getNumIndexNode() (int, errors.Errors) {
	info := s.Info()
	nodes, errs := info.Topology()
	if len(errs) > 0 {
		return 0, errs
	}

	numIndexNode := 0
	for _, node := range nodes {
		nodeServices, errs := info.Services(node)
		if len(errs) > 0 {
			return 0, errs
		}
		// the nodeServices should have an element named "services" which is
		// an array of service names on that node, e.g. ["n1ql", "kv", "index"]
		if services, ok := nodeServices["services"]; ok {
			if serviceArr, ok := services.([]interface{}); ok {
				for _, serv := range serviceArr {
					if name, ok := serv.(string); ok {
						if name == "index" {
							numIndexNode++
						}
					}
				}
			}
		}
	}

	return numIndexNode, nil
}

// check for existance of system collection, and create primary index if necessary
func (s *store) CheckSystemCollection(bucketName, requestId string) errors.Error {
	sysColl, err := s.GetSystemCollection(bucketName)
	if err != nil {
		return err
	}

	indexer, er := sysColl.Indexer(datastore.GSI)
	if er != nil {
		return er
	}

	indexer3, ok := indexer.(datastore.Indexer3)
	if !ok {
		return errors.NewInvalidGSIIndexerError("Cannot get primary index on system collection")
	}

	_, er = indexer3.IndexByName(_SYSTEM_PRIM_INDEX)
	if er != nil {
		if er.Code() != errors.E_CB_INDEX_NOT_FOUND {
			// only ignore index not found error
			return er
		}

		// create primary index on system collection if not already exists
		er = s.CreateSysPrimaryIndex(_SYSTEM_PRIM_INDEX, requestId, indexer3)
		if er != nil {
			return er
		}
	}

	return nil
}
