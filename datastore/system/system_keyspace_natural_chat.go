//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"strings"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/natural"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type naturalchatsKeyspace struct {
	keyspaceBase
	indexer datastore.Indexer
}

func (b *naturalchatsKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	var count int64
	natural.ForEachConversation(func(chatId string, entry *natural.ChatEntry) bool {
		count++
		return true
	}, nil)

	store := context.Datastore()
	if store == nil {
		context.Warning(errors.NewNoDatastoreError())
	} else {
		err := natural.ScanQueryMetadataForNLChat(nil,
			func(key string, keyspace datastore.Keyspace) errors.Error {
				if strings.HasPrefix(key, natural.CHAT_DOC_PREFIX) {
					count++
				}
				return nil
			},
			nil)
		if err != nil {
			if err.Code() == errors.E_ENTERPRISE_FEATURE {
				return count, nil
			}
			context.Warning(err)
		}
	}
	return count, nil
}

func (b *naturalchatsKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue, context datastore.QueryContext,
	subPath []string, projection []string, useSubDoc bool) errors.Errors { // Bulk key-value fetch from this keyspace

	var creds distributed.Creds

	userName := datastore.CredsString(context.Credentials())
	if userName == "" {
		creds = distributed.NO_CREDS
	} else {
		creds = distributed.Creds(userName)
	}

	whoamI := distributed.RemoteAccess().WhoAmI()
	diskkeys := []string{}
	for _, key := range keys {

		prefixlen := len(natural.CHAT_DOC_PREFIX)
		if len(key) > prefixlen && key[:prefixlen] == natural.CHAT_DOC_PREFIX {
			diskkeys = append(diskkeys, key)
			continue
		}

		node, localkey := distributed.RemoteAccess().SplitKey(key)
		if node == whoamI {
			rv := natural.GetConversation(localkey)
			if rv == nil {
				continue // no such chat, possibly removed
			}
			ce := rv.(*natural.ChatEntry)
			if userName != "" && ce.User != userName {
				continue
			}
			itemMap := natural.FormatChatEntry(ce)
			itemMap["node"] = whoamI
			item := value.NewAnnotatedValue(itemMap)
			item.SetId(key)

			keysMap[key] = item
		} else {
			distributed.RemoteAccess().GetRemoteDoc(node, localkey, "natural_chats", "GET",
				func(doc map[string]interface{}) {
					doc["node"] = node
					item := value.NewAnnotatedValue(doc)
					item.SetId(key)

					keysMap[key] = item
				},
				func(warn errors.Error) {
					if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
						context.Warning(warn)
					}
				}, creds, "", nil)
		}
	}

	var errs errors.Errors
	if len(diskkeys) > 0 {
		store := context.Datastore()
		if store == nil {
			return []errors.Error{errors.NewNoDatastoreError()}
		} else {
			queryMetadata, err := store.GetQueryMetadata()
			if err != nil {
				if !(err.Code() == errors.E_CB_KEYSPACE_NOT_FOUND &&
					strings.Contains(err.Error(), "No bucket named QUERY_METADATA")) {
					return []errors.Error{err}
				}
				return nil
			}
			fetchMap := make(map[string]value.AnnotatedValue, len(diskkeys))
			errs = queryMetadata.Fetch(diskkeys, fetchMap, datastore.NULL_QUERY_CONTEXT, nil, nil, false)
			for _, key := range diskkeys {
				if val, ok := fetchMap[key]; ok {
					b, err := natural.GetChatDataFromObjectValue(val)
					if err != nil {
						context.Warning(errors.NewQueryMetadataError("unexpected document for key "+key, err))
						continue
					}
					item := value.NewAnnotatedValue(b)
					chatId := strings.TrimPrefix(key, natural.CHAT_DOC_PREFIX)
					item.SetId(key)
					item.SetField("paused", true)
					item.SetField("chatId", chatId)
					keysMap[key] = item
				}
			}
		}
	}
	return errs
}

func (b *naturalchatsKeyspace) Delete(deletes value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {

	var creds distributed.Creds

	userName := credsFromContext(context)
	if userName == "" {
		creds = distributed.NO_CREDS
	} else {
		creds = distributed.Creds(userName)
	}

	whoAmI := distributed.RemoteAccess().WhoAmI()
	var err errors.Error
	diskPairs := []value.Pair{}
	deletedCount := 0
	deleted := []value.Pair{}
	for _, pair := range deletes {
		name := pair.Name
		prefixlen := len(natural.CHAT_DOC_PREFIX)
		if len(name) > prefixlen && name[:prefixlen] == natural.CHAT_DOC_PREFIX {
			diskPairs = append(diskPairs, pair)
			continue
		}
		node, localKey := distributed.RemoteAccess().SplitKey(name)
		if node == whoAmI {
			c := natural.GetConversation(localKey)
			if c != nil {
				if ce, ok := c.(*natural.ChatEntry); ok {
					if userName != "" && ce.User != userName {
						continue
					}
					ce.Lock()
					if ce.Removed || ce.Paused {
						// already removed, possibly by another routine.
						ce.Unlock()
						continue
					}
					natural.DeleteConversation(localKey)
					ce.Removed = true
					ce.Unlock()
					deletedCount++
				}
			} else {
				err = errors.NewNaturalLanguageRequestError(errors.E_NL_NO_SUCH_CHAT, localKey)
			}
		} else {
			// remote entry
			distributed.RemoteAccess().GetRemoteDoc(node, localKey, "natural_chats", "DELETE", nil,
				func(warn errors.Error) {
					if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
						context.Warning(warn)
					}
				},
				creds, "", nil)
			deletedCount++
		}

		if err != nil {
			errs := errors.Errors{err}
			if preserveMutations {
				return deletedCount, deleted, errs
			} else {
				return deletedCount, nil, errs
			}
		} else {
			if preserveMutations {
				deleted = append(deleted, pair)
			}
		}
	}

	var errs errors.Errors
	var diskdeletes []value.Pair
	var dCount int
	if len(diskPairs) > 0 {
		store := context.Datastore()
		if store == nil {
			errs = []errors.Error{errors.NewNoDatastoreError()}
			if preserveMutations {
				return deletedCount, deleted, errs
			} else {
				return deletedCount, nil, errs
			}
		} else {
			queryMetadata, err := store.GetQueryMetadata()
			if err != nil {
				errs = []errors.Error{err}
				if preserveMutations {
					return deletedCount, deleted, errs
				} else {
					return deletedCount, nil, errs
				}
			}
			dCount, diskdeletes, errs = queryMetadata.Delete(diskPairs, datastore.NULL_QUERY_CONTEXT, preserveMutations)
			deletedCount += dCount
			if preserveMutations {
				deleted = append(deleted, diskdeletes...)
			}
		}
	}

	if preserveMutations {
		return deletedCount, deleted, errs
	} else {
		return deletedCount, nil, errs
	}
}

func (b *naturalchatsKeyspace) Id() string {
	return b.name
}

func (b *naturalchatsKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *naturalchatsKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *naturalchatsKeyspace) Name() string {
	return b.name
}

func (b *naturalchatsKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *naturalchatsKeyspace) Release(close bool) {
}

func (b *naturalchatsKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func NewNaturalChatsKeyspace(p *namespace) (*naturalchatsKeyspace, errors.Error) {
	b := new(naturalchatsKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_NATURAL_CHAT)

	primary := &naturalChatIndex{
		name:     PRIMARY_INDEX_NAME,
		primary:  true,
		keyspace: b,
	}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)

	expr, err := parser.Parse(`node`)

	if err == nil {
		key := expression.Expressions{expr}
		nodes := &naturalChatIndex{
			name:     "#nodes",
			keyspace: b,
			primary:  false,
			idxKey:   key,
		}
		setIndexBase(&nodes.indexBase, b.indexer)
		b.indexer.(*systemIndexer).AddIndex(nodes.name, nodes)
	} else {
		return nil, errors.NewSystemDatastoreError(err, "")
	}

	return b, nil
}

type naturalChatIndex struct {
	indexBase
	name     string
	primary  bool
	keyspace *naturalchatsKeyspace
	idxKey   expression.Expressions
}

func (pi *naturalChatIndex) Condition() expression.Expression {
	return nil
}

func (pi *naturalChatIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *naturalChatIndex) Id() string {
	return pi.name
}

func (pi *naturalChatIndex) IsPrimary() bool {
	return pi.primary
}

func (pi *naturalChatIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *naturalChatIndex) Name() string {
	return pi.name
}

func (pi *naturalChatIndex) RangeKey() expression.Expressions {
	return pi.idxKey
}

func (pi *naturalChatIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	if span == nil || pi.primary {
		pi.ScanEntries(requestId, limit, cons, vector, conn)
	} else {

		defer conn.Sender().Close()

		var dsEntry *datastore.IndexEntry

		whoAmI := distributed.RemoteAccess().WhoAmI()
		spanEvaluator, err := compileSpan(span)
		if err != nil {
			conn.Error(err)
			return
		}

		process := func(chatId string, entry *natural.ChatEntry) bool {
			dsEntry = &datastore.IndexEntry{
				PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, chatId),
				EntryKey:   value.Values{value.NewValue(whoAmI)},
			}
			return true
		}

		send := func() bool {
			return sendSystemKey(conn, dsEntry)
		}

		idx := spanEvaluator.isEquals()
		if idx >= 0 {
			if spanEvaluator.key(idx) == whoAmI {
				natural.ForEachConversation(process, send)
			} else {
				nodes := []string{decodeNodeName(spanEvaluator.key(idx))}
				distributed.RemoteAccess().GetRemoteKeys(nodes, "natural_chats", func(id string) bool {
					n, _ := distributed.RemoteAccess().SplitKey(id)
					indexEntry := datastore.IndexEntry{
						PrimaryKey: id,
						EntryKey:   value.Values{value.NewValue(n)},
					}
					return sendSystemKey(conn, &indexEntry)
				}, func(warn errors.Error) {
					if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
						conn.Warning(warn)
					}
				}, distributed.NO_CREDS, "")
			}
		} else {
			nodes := distributed.RemoteAccess().GetNodeNames()
			eligibleNodes := []string{}
			for _, node := range nodes {
				if spanEvaluator.evaluate(node) {
					if node == whoAmI {
						natural.ForEachConversation(process, send)
					} else {
						eligibleNodes = append(eligibleNodes, node)
					}
				}
			}

			if len(eligibleNodes) > 0 {
				distributed.RemoteAccess().GetRemoteKeys(eligibleNodes, "natural_chats", func(id string) bool {
					n, _ := distributed.RemoteAccess().SplitKey(id)
					indexEntry := datastore.IndexEntry{
						PrimaryKey: id,
						EntryKey:   value.Values{value.NewValue(n)},
					}
					return sendSystemKey(conn, &indexEntry)
				},
					func(warn errors.Error) {
						if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
							conn.Warning(warn)
						}
					}, distributed.NO_CREDS, "")
			}
		}
	}
}

func (pi *naturalChatIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {

	defer conn.Sender().Close()
	var entry *datastore.IndexEntry

	whoAmI := distributed.RemoteAccess().WhoAmI()
	natural.ForEachConversation(
		func(chatId string, elem *natural.ChatEntry) bool {
			entry = &datastore.IndexEntry{PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, chatId)}
			return true
		},
		func() bool {
			return sendSystemKey(conn, entry)
		})

	distributed.RemoteAccess().GetRemoteKeys([]string{}, "natural_chats", func(id string) bool {
		indexEntry := datastore.IndexEntry{PrimaryKey: id}
		return sendSystemKey(conn, &indexEntry)
	}, func(warn errors.Error) {
		if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
			conn.Warning(warn)
		}
	}, distributed.NO_CREDS, "")

	err := natural.ScanQueryMetadataForNLChat(nil,
		func(key string, keyspace datastore.Keyspace) errors.Error {
			if strings.HasPrefix(key, natural.CHAT_DOC_PREFIX) {
				indexEntry := datastore.IndexEntry{PrimaryKey: key}
				sendSystemKey(conn, &indexEntry)
			}
			return nil
		},
		nil)
	if err != nil {
		if err.Code() == errors.E_ENTERPRISE_FEATURE {
			return
		}
		conn.Error(err)
		return
	}
}

func (pi *naturalChatIndex) SeekKey() expression.Expressions {
	return pi.idxKey
}

func (pi *naturalChatIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *naturalChatIndex) Statistics(requestId string, span *datastore.Span) (datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *naturalChatIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}
