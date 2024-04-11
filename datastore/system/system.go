//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"time"

	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/value"
)

const STORE_ID = datastore.SYSTEM_NAMESPACE_NAME
const NAMESPACE_ID = datastore.SYSTEM_NAMESPACE
const NAMESPACE_NAME = datastore.SYSTEM_NAMESPACE_NAME
const KEYSPACE_NAME_DATASTORES = "datastores"
const KEYSPACE_NAME_NAMESPACES = "namespaces"
const KEYSPACE_NAME_BUCKETS = "buckets"
const KEYSPACE_NAME_SCOPES = "scopes"
const KEYSPACE_NAME_ALL_SCOPES = "all_scopes"
const KEYSPACE_NAME_KEYSPACES = "keyspaces"
const KEYSPACE_NAME_ALL_KEYSPACES = "all_keyspaces"
const KEYSPACE_NAME_KEYSPACES_INFO = "keyspaces_info"
const KEYSPACE_NAME_ALL_KEYSPACES_INFO = "all_keyspaces_info"
const KEYSPACE_NAME_INDEXES = "indexes"
const KEYSPACE_NAME_ALL_INDEXES = "all_indexes"
const KEYSPACE_NAME_DUAL = "dual"
const KEYSPACE_NAME_PREPAREDS = "prepareds"
const KEYSPACE_NAME_FUNCTIONS_CACHE = "functions_cache"
const KEYSPACE_NAME_FUNCTIONS = "functions"
const KEYSPACE_NAME_DICTIONARY_CACHE = "dictionary_cache"
const KEYSPACE_NAME_DICTIONARY = "dictionary"
const KEYSPACE_NAME_REQUESTS = "completed_requests"
const KEYSPACE_NAME_ACTIVE = "active_requests"
const KEYSPACE_NAME_USER_INFO = "user_info"
const KEYSPACE_NAME_MY_USER_INFO = "my_user_info"
const KEYSPACE_NAME_GROUP_INFO = "group_info"
const KEYSPACE_NAME_BUCKET_INFO = "bucket_info"
const KEYSPACE_NAME_DATABASE_INFO = "database_info"
const KEYSPACE_NAME_NODES = "nodes"
const KEYSPACE_NAME_APPLICABLE_ROLES = "applicable_roles"
const KEYSPACE_NAME_TASKS_CACHE = "tasks_cache"
const KEYSPACE_NAME_TRANSACTIONS = "transactions"
const KEYSPACE_NAME_VITALS = "vitals"
const KEYSPACE_NAME_SEQUENCES = "sequences"
const KEYSPACE_NAME_ALL_SEQUENCES = "all_sequences"

// TODO, sync with fetch timeout
const scanTimeout = 30 * time.Second

type store struct {
	actualStore              datastore.Datastore
	acctStore                accounting.AccountingStore
	systemDatastoreNamespace *namespace
}

func (s *store) PrivilegesFromPath(fullname string, keyspace string, privilege auth.Privilege, privs *auth.Privileges) {
	switch privilege {
	case auth.PRIV_QUERY_DELETE:
		switch keyspace {

		// currently these keyspaces require system read for delete if on prem
		// and open (but limited to the user) for elixir
		case KEYSPACE_NAME_ACTIVE, KEYSPACE_NAME_REQUESTS, KEYSPACE_NAME_PREPAREDS:
			if !tenant.IsServerless() {
				privs.Add("", auth.PRIV_SYSTEM_READ, auth.PRIV_PROPS_NONE)
			}

		// currently these keyspaces require system read for delete
		case KEYSPACE_NAME_FUNCTIONS_CACHE, KEYSPACE_NAME_DICTIONARY_CACHE:
			privs.Add("", auth.PRIV_SYSTEM_READ, auth.PRIV_PROPS_NONE)

			// for all other keyspaces, we rely on the implementation do deny access
		}

	// for SELECT previous code specified a target, even though it's not needed
	// we still specify a target for backward compatibility and to avoid test failures
	case auth.PRIV_QUERY_SELECT:
		switch keyspace {
		case KEYSPACE_NAME_USER_INFO, KEYSPACE_NAME_APPLICABLE_ROLES, KEYSPACE_NAME_GROUP_INFO:
			privs.Add(fullname, auth.PRIV_SECURITY_READ, auth.PRIV_PROPS_NONE)

		// may be open, depending whether admin REST endpoint is open
		case KEYSPACE_NAME_NODES:
			privs.Add("", auth.PRIV_SYSTEM_OPEN, auth.PRIV_PROPS_NONE)

		// open to all, no privileges required
		case KEYSPACE_NAME_DATASTORES:
		case KEYSPACE_NAME_NAMESPACES:
		case KEYSPACE_NAME_DUAL:

		// these keyspaces filter results according to user privileges
		// no further privilegs required
		case KEYSPACE_NAME_KEYSPACES:
		case KEYSPACE_NAME_ALL_KEYSPACES:
		case KEYSPACE_NAME_KEYSPACES_INFO:
		case KEYSPACE_NAME_ALL_KEYSPACES_INFO:
		case KEYSPACE_NAME_SCOPES:
		case KEYSPACE_NAME_ALL_SCOPES:
		case KEYSPACE_NAME_BUCKETS:
		case KEYSPACE_NAME_INDEXES:
		case KEYSPACE_NAME_ALL_INDEXES:
		case KEYSPACE_NAME_MY_USER_INFO:
		case KEYSPACE_NAME_FUNCTIONS:
		case KEYSPACE_NAME_SEQUENCES:
		case KEYSPACE_NAME_ALL_SEQUENCES:

		// currently these keyspaces require system read for select if on prem
		// and open (but limited to the user) for elixir
		case KEYSPACE_NAME_ACTIVE, KEYSPACE_NAME_REQUESTS, KEYSPACE_NAME_PREPAREDS:
			if !tenant.IsServerless() {
				privs.Add("", auth.PRIV_SYSTEM_READ, auth.PRIV_PROPS_NONE)
			}

		// system read for everything else
		default:
			privs.Add(fullname, auth.PRIV_SYSTEM_READ, auth.PRIV_PROPS_NONE)
		}

		// for every other privilege, the keyspaces internally deny access
		// should this change, this method needs to be used in the algebra package
		// for the privileges of any new DML / DDL involved
	}
}

func (s *store) Id() string {
	return STORE_ID
}

func (s *store) URL() string {
	return ""
}

func (s *store) Info() datastore.Info {
	return s.actualStore.Info()
}

func (s *store) NamespaceIds() ([]string, errors.Error) {
	namespaceIds, err := s.actualStore.NamespaceIds()
	if err != nil {
		return nil, err
	}
	namespaceIds = append(namespaceIds, s.systemDatastoreNamespace.Id())
	return namespaceIds, err
}

func (s *store) NamespaceNames() ([]string, errors.Error) {
	namespaceNames, err := s.actualStore.NamespaceNames()
	if err != nil {
		return nil, err
	}
	namespaceNames = append(namespaceNames, s.systemDatastoreNamespace.Name())
	return namespaceNames, err
}

func (s *store) NamespaceById(id string) (datastore.Namespace, errors.Error) {
	if id == NAMESPACE_ID {
		return s.systemDatastoreNamespace, nil
	}
	return s.actualStore.NamespaceById(id)
}

func (s *store) NamespaceByName(name string) (datastore.Namespace, errors.Error) {
	if name == NAMESPACE_NAME {
		return s.systemDatastoreNamespace, nil
	}
	return s.actualStore.NamespaceByName(name)
}

func (s *store) Authorize(*auth.Privileges, *auth.Credentials) errors.Error {
	logging.Logf(logging.INFO, "System authorize")
	return nil
}

func (s *store) AuthorizeInternal(*auth.Privileges, *auth.Credentials) errors.Error {
	logging.Logf(logging.INFO, "System authorize internal action")
	return nil
}

func (s *store) AdminUser(node string) (string, string, error) {
	return "", "", nil
}

func (s *store) GetUserUUID(*auth.Credentials) string {
	return ""
}

func (s *store) GetUserBuckets(*auth.Credentials) []string {
	return []string{}
}

func (s *store) GetImpersonateBuckets(string, string) []string {
	return []string{}
}

func (s *store) PreAuthorize(*auth.Privileges) {
}

func (s *store) CredsString(*auth.Credentials) (string, string) {
	return "", ""
}

func (s *store) SetLogLevel(level logging.Level) {
	// No-op. Uses query engine logger.
}

func (s *store) Inferencer(name datastore.InferenceType) (datastore.Inferencer, errors.Error) {
	return nil, errors.NewOtherNotImplementedError(nil, "INFER")
}

func (s *store) Inferencers() ([]datastore.Inferencer, errors.Error) {
	return nil, errors.NewOtherNotImplementedError(nil, "INFER")
}

func (s *store) StatUpdater() (datastore.StatUpdater, errors.Error) {
	return nil, errors.NewOtherNotImplementedError(nil, "UPDATE STATISTICS")
}

func (s *store) SetConnectionSecurityConfig(conSecConfig *datastore.ConnectionSecurityConfig) {
	// Do nothing.
}

func (s *store) AuditInfo() (*datastore.AuditInfo, errors.Error) {
	return nil, errors.NewOtherNotImplementedError(nil, "AuditInfo")
}

func (s *store) ProcessAuditUpdateStream(callb func(uid string) error) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "ProcessAuditUpdateStream")
}

func (s *store) EnableStorageAudit(val bool) {
}

func (s *store) UserInfo() (value.Value, errors.Error) {
	return s.actualStore.UserInfo()
}

func (s *store) GetUserInfoAll() ([]datastore.User, errors.Error) {
	return nil, errors.NewOtherNotImplementedError(nil, "GetUserInfoAll")
}

func (s *store) PutUserInfo(u *datastore.User) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "PutUserInfo")
}

func (s *store) DeleteUser(u *datastore.User) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "DeleteUser")
}

func (s *store) GetUserInfo(u *datastore.User) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "GetUserInfo")
}

func (s *store) GetRolesAll() ([]datastore.Role, errors.Error) {
	return nil, errors.NewOtherNotImplementedError(nil, "GetRolesAll")
}

func (s *store) GetGroupInfo(*datastore.Group) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "GetGroupInfo")
}

func (s *store) PutGroupInfo(*datastore.Group) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "PutGroupInfo")
}

func (s *store) DeleteGroup(*datastore.Group) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "DeleteGroup")
}

func (s *store) GroupInfo() (value.Value, errors.Error) {
	return s.actualStore.GroupInfo()
}

func (s *store) GetGroupInfoAll() ([]datastore.Group, errors.Error) {
	return s.actualStore.GetGroupInfoAll()
}

func (s *store) CreateBucket(string, value.Value) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "CreateBucket")
}

func (s *store) AlterBucket(string, value.Value) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "AlterBucket")
}

func (s *store) DropBucket(string) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "DropBucket")
}

func (s *store) BucketInfo() (value.Value, errors.Error) {
	return s.actualStore.BucketInfo()
}

func (s *store) CreateSystemCBOStats(requestId string) errors.Error {
	return nil
}

func (s *store) DropSystemCBOStats() errors.Error {
	return nil
}

func (s *store) GetSystemCBOStats() (datastore.Keyspace, errors.Error) {
	return nil, nil
}

func (s *store) HasSystemCBOStats() (bool, errors.Error) {
	return false, nil
}

func (s *store) GetSystemCollection(bucketName string) (datastore.Keyspace, errors.Error) {
	return nil, nil
}

func (s *store) CheckSystemCollection(bucketName, requestId string) errors.Error {
	return nil
}

func (s *store) StartTransaction(stmtAtomicity bool, context datastore.QueryContext) (map[string]bool, errors.Error) {
	return nil, errors.NewTranDatastoreNotSupportedError(NAMESPACE_NAME)
}

func (s *store) CommitTransaction(stmtAtomicity bool, context datastore.QueryContext) errors.Error {
	return errors.NewTranDatastoreNotSupportedError(NAMESPACE_NAME)
}

func (s *store) RollbackTransaction(stmtAtomicity bool, context datastore.QueryContext, sname string) errors.Error {
	return errors.NewTranDatastoreNotSupportedError(NAMESPACE_NAME)
}

func (s *store) SetSavepoint(stmtAtomicity bool, context datastore.QueryContext, sname string) errors.Error {
	return errors.NewTranDatastoreNotSupportedError(NAMESPACE_NAME)
}

func (s *store) TransactionDeltaKeyScan(keyspace string, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()
}

func NewDatastore(actualStore datastore.Datastore, acctStore accounting.AccountingStore) (datastore.Systemstore, errors.Error) {
	s := &store{actualStore: actualStore, acctStore: acctStore}

	e := s.loadNamespace()
	if e != nil {
		return nil, e
	}

	return s, e
}

func (s *store) loadNamespace() errors.Error {
	p, e := newNamespace(s)
	if e != nil {
		return e
	}

	s.systemDatastoreNamespace = p
	return nil
}

type systemKeyspaceBase struct {
	namespace datastore.Namespace
}

func (this *systemKeyspaceBase) Namespace() datastore.Namespace {
	return this.namespace
}

// System keyspaces do not implement the KeyspaceMetadata interface because they don't implement collections and scopes
// They will have to if this changes

type systemIndexer struct {
	keyspace datastore.Keyspace
	primary  datastore.PrimaryIndex
	indexes  map[string]datastore.Index
}

func newSystemIndexer(keyspace datastore.Keyspace, primary datastore.PrimaryIndex) datastore.Indexer {
	return &systemIndexer{
		keyspace: keyspace,
		primary:  primary,
		indexes:  map[string]datastore.Index{"#primary": primary},
	}
}

func (si *systemIndexer) AddIndex(name string, idx datastore.Index) {
	si.indexes[name] = idx
}

func (si *systemIndexer) BucketId() string {
	return ""
}

func (si *systemIndexer) ScopeId() string {
	return ""
}

func (si *systemIndexer) KeyspaceId() string {
	return si.keyspace.Id()
}

func (si *systemIndexer) Name() datastore.IndexType {
	return datastore.SYSTEM
}

func (si *systemIndexer) IndexIds() ([]string, errors.Error) {
	rv := make([]string, 0, len(si.indexes))
	for name, _ := range si.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (si *systemIndexer) IndexNames() ([]string, errors.Error) {
	rv := make([]string, 0, len(si.indexes))
	for name, _ := range si.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (si *systemIndexer) IndexById(id string) (datastore.Index, errors.Error) {
	return si.IndexByName(id)
}

func (si *systemIndexer) IndexByName(name string) (datastore.Index, errors.Error) {
	index, ok := si.indexes[name]
	if !ok {
		return nil, errors.NewSystemIdxNotFoundError(nil, name)
	}
	return index, nil
}

func (si *systemIndexer) PrimaryIndexes() ([]datastore.PrimaryIndex, errors.Error) {
	return []datastore.PrimaryIndex{si.primary}, nil
}

func (si *systemIndexer) Indexes() ([]datastore.Index, errors.Error) {
	rv := make([]datastore.Index, 0, len(si.indexes))
	for _, idx := range si.indexes {
		rv = append(rv, idx)
	}
	return rv, nil
}

func (si *systemIndexer) CreatePrimaryIndex(requestId, name string, with value.Value) (
	datastore.PrimaryIndex, errors.Error) {
	return nil, errors.NewSystemNotSupportedError(nil, "CREATE PRIMARY INDEX is not supported for system datastore.")
}

func (si *systemIndexer) CreateIndex(requestId, name string, seekKey, rangeKey expression.Expressions,
	where expression.Expression, with value.Value) (datastore.Index, errors.Error) {
	return nil, errors.NewSystemNotSupportedError(nil, "CREATE INDEX is not supported for system datastore.")
}

func (si *systemIndexer) BuildIndexes(requestId string, names ...string) errors.Error {
	return errors.NewSystemNotSupportedError(nil, "BUILD INDEXES is not supported for system datastore.")
}

func (si *systemIndexer) Refresh() errors.Error {
	return nil
}

func (si *systemIndexer) MetadataVersion() uint64 {
	return 0
}

func (si *systemIndexer) SetLogLevel(level logging.Level) {
	// No-op, uses query engine logger
}

func (si *systemIndexer) SetConnectionSecurityConfig(conSecConfig *datastore.ConnectionSecurityConfig) {
	// Do nothing.
}

func sendSystemKey(conn *datastore.IndexConnection, entry *datastore.IndexEntry) bool {
	stop := time.AfterFunc(scanTimeout, func() { conn.SendTimeout() })
	rv := conn.Sender().SendEntry(entry)
	stop.Stop()
	return rv
}
