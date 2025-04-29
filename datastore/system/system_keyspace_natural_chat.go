//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
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
	for _, key := range keys {
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
	return nil
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
	for i, pair := range deletes {
		name := pair.Name
		node, localKey := distributed.RemoteAccess().SplitKey(name)
		if node == whoAmI {
			c := natural.GetConversation(localKey)
			if c != nil {
				if ce, ok := c.(*natural.ChatEntry); ok {
					if userName != "" && ce.User != userName {
						continue
					}
					ce.Lock()
					if ce.Removed {
						// already removed, possibly by another routine.
						ce.Unlock()
						continue
					}
					natural.DeleteConversation(localKey)
					ce.Removed = true
					ce.Unlock()
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
		}

		if err != nil {
			errs := errors.Errors{err}
			if preserveMutations {
				deleted := make([]value.Pair, i)
				if i > 0 {
					copy(deleted, deletes[0:i])
				}
				return i, deleted, errs
			} else {
				return i, nil, errs
			}
		}
	}

	if preserveMutations {
		return len(deletes), deletes, nil
	} else {
		return len(deletes), nil, nil
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
