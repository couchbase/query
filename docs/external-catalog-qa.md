# External Catalog QA Reference

N1QL DDL and SELECT features for External Catalogs, CredentialStores, and External Collections.

---

## 1. CREATE CREDENTIALSTORE

### Syntax
```sql
CREATE CREDENTIALSTORE [IF NOT EXISTS] <name> WITH <options>
```

### Example
```sql
CREATE CREDENTIALSTORE `aws-creds` WITH {
    "type": "aws_key",
    "accessKeyId": "AKIAIOSFODNN7EXAMPLE",
    "secretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
    "region": "us-east-1"
}
```

### Semantics
- `IF NOT EXISTS` — skip error if the credential store already exists (omit to get error 12058 on duplicate)
- `WITH` — required; JSON object with credential store configuration (type-dependent)
- Enterprise only — not available on CE builds

### Required Privilege
`PRIV_SECURITY_WRITE` (cluster-level security admin)

### Underlying REST Call
`POST /settings/credentials/<name>` on the Couchbase node

---

## 2. ALTER CREDENTIALSTORE

### Syntax
```sql
ALTER CREDENTIALSTORE <name> WITH <options>
```

### Example
```sql
-- Must supply ALL fields — this replaces the entire credential store document
ALTER CREDENTIALSTORE `aws-creds` WITH {
    "type": "aws_key",
    "accessKeyId": "AKIAIOSFODNN7EXAMPLE",
    "secretAccessKey": "newSecretKey",
    "region": "us-east-1"
}
```

### Semantics
- **Full replacement** — the WITH clause replaces the entire credential store document.
  Omitting a field removes it. There is no partial/merge update.
- Returns error 12059 if the store does not exist

### Required Privilege
`PRIV_SECURITY_WRITE`

### Underlying REST Call
`POST /settings/credentials/<name>` — same HTTP method as CREATE; the full document
in the WITH clause overwrites the stored credential.

---

## 3. DROP CREDENTIALSTORE

### Syntax
```sql
DROP CREDENTIALSTORE [IF EXISTS] <name>
```

### Example
```sql
DROP CREDENTIALSTORE IF EXISTS `aws-creds`
```

### Semantics
- `IF EXISTS` — no error if the store does not exist (without it, returns error 12059)

### Required Privilege
`PRIV_SECURITY_WRITE`

### Underlying REST Call
`DELETE /settings/credentials/<name>`

---

## 4. CREATE CATALOG

### Syntax
```sql
CREATE CATALOG [IF NOT EXISTS] <name>
    TYPE <catalogType>
    SOURCE <source>
    AT <credentialStoreName>
    [WITH <options>]
```

**BNF (from parser/n1ql/n1ql.y):**
```
create_catalog ::=
    CREATE CATALOG opt_if_not_exists perm_ident_or_str
        TYPE perm_ident_or_str
        SOURCE perm_ident_or_str
        AT perm_ident_or_str
        opt_with_clause

perm_ident_or_str ::=
    permitted_identifier   -- unquoted (ICEBERG, AWS_GLUE, ...) or backtick-quoted (`my-name`)
    | string_literal       -- single-quoted ('ICEBERG', 'my-name')
```

All four positional values — catalog name, TYPE value, SOURCE value, AT value — accept
either an unquoted/backtick-quoted identifier or a single-quoted string literal.

### Examples
```sql
-- Using unquoted identifiers for TYPE, SOURCE, AT
CREATE CATALOG `my-iceberg` TYPE ICEBERG SOURCE AWS_GLUE AT `aws-creds` WITH {}

-- Using single-quoted strings for TYPE, SOURCE, AT
CREATE CATALOG 'my-iceberg' TYPE 'ICEBERG' SOURCE 'AWS_GLUE' AT 'aws-creds' WITH {}

-- Mixed: backtick name, unquoted TYPE and SOURCE, single-quoted AT
CREATE CATALOG `my-iceberg` TYPE ICEBERG SOURCE AWS_GLUE AT 'aws-creds' WITH {}

-- AWS Glue REST
CREATE CATALOG `my-glue-rest` TYPE ICEBERG SOURCE AWS_GLUE_REST AT `aws-creds`
WITH { "uri": "https://glue.us-east-1.amazonaws.com", "sigv4SigningRegion": "us-east-1" }

-- Nessie: AT is always syntactically required; pass "" when no credential is needed
CREATE CATALOG `nessie-cat` TYPE ICEBERG SOURCE NESSIE AT ""
WITH { "uri": "http://nessie-host:19120/api/v1", "warehouse": "s3://bucket/warehouse" }

-- Generic REST
CREATE CATALOG `rest-cat` TYPE ICEBERG SOURCE REST AT `creds`
WITH { "uri": "https://catalog.example.com" }
```

### Semantics
- Enterprise only — returns `NewEnterpriseFeature` error on CE
- `name` — required; empty string rejected (E_FIELD_EMPTY)
- `TYPE` — required; empty string rejected (E_FIELD_EMPTY); currently supported: `ICEBERG`
- `SOURCE` — required; empty string rejected (E_FIELD_EMPTY); supported values:
  `AWS_GLUE`, `AWS_GLUE_REST`, `S3_TABLES`, `BIGLAKE_METASTORE`, `NESSIE`, `NESSIE_REST`
- `AT` — **always syntactically required** (grammar has no opt_ variant); semantically:
  - Non-NESSIE sources: credential name must be non-empty (E_FIELD_EMPTY if empty)
  - NESSIE/NESSIE_REST sources: credential name may be empty string `""` — semantic check
    is skipped when `SOURCE` starts with `NESSIE` (case-insensitive prefix match)
- `WITH` — optional; provides source-specific parameters (see tables below)

### Catalog WITH clause parameters — by TYPE

The following parameters are always required. When using the N1QL DDL, they are
populated automatically from the TYPE, SOURCE, and AT clauses. When supplying
parameters directly (e.g., via REST API), they must be provided explicitly.

#### TYPE = ICEBERG (all sources)

| Parameter | DDL clause | Type | Mandatory |
|-----------|------------|------|-----------|
| `catalogType` | TYPE | string | Yes |
| `catalogSource` | SOURCE | string | Yes |
| `credentialId` | AT | string | Yes (empty `""` allowed for NESSIE/NESSIE_REST) |

Optional WITH parameters valid for all ICEBERG sources:

| Parameter | Type | Mandatory | Description |
|-----------|------|-----------|-------------|
| `rev` | int | No | Revision number (managed by server) |
| `uid` | string | No | Unique catalog ID (managed by server) |
| `compat_version` | int | No | Compatibility version (managed by server) |

#### TYPE = ICEBERG, SOURCE = AWS_GLUE

No source-specific parameters. WITH clause must still be supplied as an empty object.

```sql
CREATE CATALOG `my-iceberg` TYPE ICEBERG SOURCE AWS_GLUE AT `aws-creds` WITH {}
```

#### TYPE = ICEBERG, SOURCE = AWS_GLUE_REST

| Parameter | Type | Mandatory | Description |
|-----------|------|-----------|-------------|
| `uri` | string | Yes | Glue REST endpoint URI |
| `sigv4SigningRegion` | string | Yes | AWS region for SigV4 request signing |

#### TYPE = ICEBERG, SOURCE = S3_TABLES

| Parameter | Type | Mandatory | Description |
|-----------|------|-----------|-------------|
| `uri` | string | Yes | S3 Tables endpoint URI |
| `sigv4SigningRegion` | string | Yes | AWS region for SigV4 request signing |
| `sigv4SigningName` | string | Yes | AWS service name for SigV4 signing |
| `warehouse` | string | Yes | S3 warehouse location |

#### TYPE = ICEBERG, SOURCE = BIGLAKE_METASTORE

| Parameter | Type | Mandatory | Description |
|-----------|------|-----------|-------------|
| `uri` | string | Yes | BigLake Metastore endpoint URI |
| `warehouse` | string | Yes | GCS warehouse location |
| `quotaProjectId` | string | Yes | GCP project ID for quota billing |

#### TYPE = ICEBERG, SOURCE = NESSIE

| Parameter | Type | Mandatory | Description |
|-----------|------|-----------|-------------|
| `uri` | string | Yes | Nessie server URI |
| `warehouse` | string | Yes | Warehouse base path |

#### TYPE = ICEBERG, SOURCE = NESSIE_REST

| Parameter | Type | Mandatory | Description |
|-----------|------|-----------|-------------|
| `uri` | string | Yes | Nessie REST endpoint URI |

### Required Privilege
`PRIV_CLUSTER_ADMIN` (full cluster admin)

### Underlying REST Call
`POST /pools/default/externalCatalogs` with JSON body including `name`, `catalogType`,
`catalogSource`, `credentialId`, and source-specific fields.

---

## 5. ALTER CATALOG

### Syntax
```sql
ALTER CATALOG <name> WITH <options>
```

### Example
```sql
ALTER CATALOG `my-iceberg` WITH { "sigv4SigningRegion": "eu-west-1" }
```

### Semantics
- Enterprise only
- Name must be non-empty; returns error 12052 if not found
- Updates catalog config via the NS-Server REST API

### Required Privilege
`PRIV_CLUSTER_ADMIN`

### Underlying REST Call
`PATCH /pools/default/externalCatalogs/<name>`

---

## 6. DROP CATALOG

### Syntax
```sql
DROP CATALOG [IF EXISTS] <name>
```

### Example
```sql
DROP CATALOG IF EXISTS `my-iceberg`
```

### Semantics
- Enterprise only
- `IF EXISTS` prevents error 12052 on missing catalog
- Dropping a catalog does **not** automatically drop associated external collections

### Required Privilege
`PRIV_CLUSTER_ADMIN`

### Underlying REST Call
`DELETE /pools/default/externalCatalogs/<name>`

---

## 7. CREATE EXTERNAL COLLECTION

### Syntax
```sql
CREATE EXTERNAL COLLECTION [IF NOT EXISTS]
    <keyspace_ref>
    ON <catalogName>
    AT <credentialStoreName>
    [WITH <options>]
```

`<keyspace_ref>` uses the same path resolution as normal `CREATE COLLECTION` — missing
parts are filled in from the query context (`query_context` parameter or session default).

**Accepted forms (same as normal collections):**

| Form | Example | Requires query_context |
|------|---------|----------------------|
| collection name only | `iceberg_flights` | namespace + bucket + scope |
| bucket.collection | `travel-sample.iceberg_flights` | namespace + scope |
| bucket.scope.collection | `travel-sample.inventory.iceberg_flights` | namespace only |
| namespace:bucket.scope.collection | `default:travel-sample.inventory.iceberg_flights` | none |

### Examples
```sql
-- Fully qualified (no query_context needed)
CREATE EXTERNAL COLLECTION default:travel-sample.inventory.iceberg_flights
    ON `my-iceberg` AT `aws-creds`

-- With query_context = "default:travel-sample.inventory"
CREATE EXTERNAL COLLECTION iceberg_flights
    ON `my-iceberg` AT `aws-creds`

-- 3-part path, query_context provides namespace
CREATE EXTERNAL COLLECTION travel-sample.inventory.iceberg_flights
    ON `my-iceberg` AT `aws-creds`
    WITH { "tablePath": "s3://my-bucket/data/flights/" }
```

### Semantics
- Keyspace path resolution is identical to normal collections; query_context is honored
- After resolution, all four parts (namespace, bucket, scope, collection) must be
  non-empty — the semantic checker rejects any that remain empty after context resolution
- `ON <catalogName>` — required; references an existing catalog
- `AT <credentialStoreName>` — required; references an existing credential store
- External collections are backed by Iceberg (or similar) tables — no KV data
- Required privilege is on the **scope**: `PRIV_QUERY_SCOPE_ADMIN` on `<bucket>.<scope>`

### External Collection WITH clause parameters

The following parameters are always required. When using the N1QL DDL, they are
populated automatically from the collection name, ON, and AT clauses. When supplying
parameters directly (e.g., via REST API), they must be provided explicitly.

| Parameter | DDL clause | Type | Mandatory |
|-----------|------------|------|-----------|
| `name` | collection name | string | Yes |
| `catalog` | ON | string | Yes |
| `catalogType` | resolved from catalog | string | Yes |
| `credentialId` | AT | string | Yes |

Mandatory WITH parameters (TYPE = ICEBERG):

| Parameter | Type | Mandatory | Description |
|-----------|------|-----------|-------------|
| `namespace` | string | Yes | Iceberg namespace (e.g., Glue database name) |
| `tableName` | string | Yes | Iceberg table name within the namespace |

Optional WITH parameters (TYPE = ICEBERG):

| Parameter | Type | Mandatory | Description |
|-----------|------|-----------|-------------|
| `format` | string | No | File format: `PARQUET`, `ORC`, `AVRO`, etc. |
| `snapshotId` | string | No | Default snapshot ID to use when querying |
| `snapshotTimestamp` | string | No | Default snapshot timestamp to use when querying |
| `parallelScans` | int | No | Number of parallel scan threads (default: 1) |
| `rev` | string | No | Revision (managed by server) |
| `uid` | string | No | Unique collection ID (managed by server) |
| `compat_version` | int | No | Compatibility version (managed by server) |

---

## 8. ALTER COLLECTION (external collections only)

### Syntax
```sql
ALTER COLLECTION <keyspace_ref> WITH <options>
```

Same `<keyspace_ref>` path resolution as CREATE EXTERNAL COLLECTION — query_context is
honored and partial paths are accepted.

- Attempting `ALTER COLLECTION` on an **internal** collection returns:
  `ALTER COLLECTION is not supported on non-external collections`

---

## 9. DROP COLLECTION (external collections)

### Syntax
```sql
DROP COLLECTION [IF EXISTS] <keyspace_ref>
```

Same `<keyspace_ref>` path resolution as CREATE EXTERNAL COLLECTION — query_context is
honored. Syntax is identical to dropping an internal collection.

---

## 10. GRANT / REVOKE — Catalog Privileges

### Syntax
```sql
-- Grant to users
GRANT {SELECT|INSERT|UPDATE|DELETE} CATALOG ON <catalogName>[, ...] TO <user>[, ...]
GRANT {SELECT|INSERT|UPDATE|DELETE} CATALOG ON <catalogName>[, ...] TO USERS <user>[, ...]

-- Grant to groups
GRANT {SELECT|INSERT|UPDATE|DELETE} CATALOG ON <catalogName>[, ...] TO {GROUP|GROUPS} <group>[, ...]

-- Revoke from users
REVOKE {SELECT|INSERT|UPDATE|DELETE} CATALOG ON <catalogName>[, ...] FROM <user>[, ...]
REVOKE {SELECT|INSERT|UPDATE|DELETE} CATALOG ON <catalogName>[, ...] FROM USERS <user>[, ...]

-- Revoke from groups
REVOKE {SELECT|INSERT|UPDATE|DELETE} CATALOG ON <catalogName>[, ...] FROM {GROUP|GROUPS} <group>[, ...]
```

### Examples
```sql
GRANT SELECT CATALOG ON `my-iceberg` TO user1
GRANT INSERT CATALOG, UPDATE CATALOG ON `my-iceberg` TO USERS user1, user2
GRANT DELETE CATALOG ON `cat1`, `cat2` TO GROUP analysts
REVOKE SELECT CATALOG ON `my-iceberg` FROM user1
```

### Catalog Privilege Mapping

| SQL Privilege | Internal Role | Code |
|---------------|---------------|------|
| `SELECT CATALOG` | `PRIV_CATALOG_SELECT` | 51 |
| `INSERT CATALOG` | `PRIV_CATALOG_INSERT` | 53 |
| `UPDATE CATALOG` | `PRIV_CATALOG_UPDATE` | 52 |
| `DELETE CATALOG` | `PRIV_CATALOG_DELETE` | 54 |

---

## 11. GRANT / REVOKE — CredentialStore Privileges

### Syntax
```sql
GRANT CONSUME CREDENTIALSTORE ON <credStoreName>[, ...] TO <user>[, ...]
GRANT CONSUME CREDENTIALSTORE ON <credStoreName>[, ...] TO USERS <user>[, ...]
GRANT CONSUME CREDENTIALSTORE ON <credStoreName>[, ...] TO {GROUP|GROUPS} <group>[, ...]

REVOKE CONSUME CREDENTIALSTORE ON <credStoreName>[, ...] FROM <user>[, ...]
REVOKE CONSUME CREDENTIALSTORE ON <credStoreName>[, ...] FROM USERS <user>[, ...]
REVOKE CONSUME CREDENTIALSTORE ON <credStoreName>[, ...] FROM {GROUP|GROUPS} <group>[, ...]
```

### Examples
```sql
GRANT CONSUME CREDENTIALSTORE ON `aws-creds` TO user1
GRANT CONSUME CREDENTIALSTORE ON `aws-creds`, `gcp-creds` TO GROUP dataops
REVOKE CONSUME CREDENTIALSTORE ON `aws-creds` FROM user1
```

### CredentialStore Privilege Mapping

| SQL Privilege | Internal Role | Code |
|---------------|---------------|------|
| `CONSUME CREDENTIALSTORE` | `PRIV_CLUSTER_CREDENTIALSTORE_CONSUME` | 48 |

---

## 12. System Keyspaces

### system:catalogs

Lists all catalogs visible to the current user.

```sql
SELECT * FROM system:catalogs
SELECT * FROM system:catalogs WHERE name = "my-iceberg"
```

**Columns:**

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Catalog name (primary key) |
| `catalogType` | string | e.g., `"ICEBERG"` |
| `catalogSource` | string | e.g., `"AWS_GLUE"`, `"AWS_GLUE_REST"`, `"NESSIE"`, `"REST"` |
| `credentialId` | string | Referenced credential store name |
| `uid` | string | Unique catalog ID |
| `rev` | number | Revision number |
| `uri` | string | URI (REST/Nessie sources only) |
| `warehouse` | string | Warehouse path (Nessie only) |
| `sigv4SigningRegion` | string | AWS region for SigV4 signing (AWS_GLUE_REST only) |
| `quotaProjectId` | string | GCP quota project (GCP sources only) |

**Access control:** Requires `PRIV_CATALOGS_READ`. Entries are filtered per-catalog by
credential read access.

---

### system:catalogs_info

Provides extended catalog information including schema metadata and snapshot history.

```sql
SELECT * FROM system:catalogs_info
SELECT * FROM system:catalogs_info WHERE name = "my-iceberg"
```

Same columns as `system:catalogs`, plus additional schema/snapshot data when loaded.

**Access control:** Requires both `PRIV_CATALOGS_READ` **and**
`PRIV_CLUSTER_CREDENTIALSTORE_CONSUME` on the referenced credential store. Users without
credential consume access will not see catalog info for catalogs using that credential.

---

### system:keyspaces — extra fields for external collections

External collections show additional fields beyond what internal collections show:

```json
{
  "id": "iceberg_flights",
  "name": "iceberg_flights",
  "bucket_id": "travel-sample",
  "scope_id": "inventory",
  "catalog": "my-iceberg",
  "catalogType": "ICEBERG",
  "credentialId": "aws-creds",
  "with": {
    "namespace": "my_glue_db",
    "tableName": "flights",
    "snapshotId": "5688794825202065870",
    "snapshotTimestamp": "2025-04-14T00:00:00Z",
    "format": "PARQUET"
  }
}
```

| Extra field | Description |
|-------------|-------------|
| `catalog` | Catalog name the collection is linked to |
| `catalogType` | e.g., `"ICEBERG"` |
| `credentialId` | Referenced credential store |
| `with.namespace` | Iceberg namespace / Glue database |
| `with.tableName` | Iceberg table name |
| `with.snapshotId` | Default snapshot ID (if pinned at creation) |
| `with.snapshotTimestamp` | Default snapshot timestamp (if pinned at creation) |
| `with.format` | File format: `"PARQUET"`, `"ORC"`, `"AVRO"`, etc. |

---

## 13. Admin REST Endpoints (Query Service)

### GET /admin/catalogs

Retrieves catalog details via the query service admin endpoint.

```
GET /admin/catalogs
GET /admin/catalogs?name=my-iceberg
GET /admin/catalogs?name=my-iceberg&infoFlags=7
```

**Query parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `name` | string | (Optional) Filter to a specific catalog name |
| `infoFlags` | uint64 | (Optional) Bitmask controlling detail level |

**infoFlags bitmask:**

| Value | Meaning |
|-------|---------|
| `1` | Include schema information |
| `2` | Include snapshots history |
| `4` | Include data files listing |
| `7` | All of the above |

**Authorization:** `PRIV_CLUSTER_ADMIN`

**Audit event:** `API_ADMIN_CATALOGS`

**Response:** JSON object of catalog details.

---

## 14. Prometheus Metrics

### n1ql_external_scans

| Attribute | Value |
|-----------|-------|
| Metric name | `n1ql_external_scans` |
| Type | Counter (cumulative) |
| Added | 8.1.0 |
| Help text | "Total number of external collection scans." |
| Endpoints | `GET /_prometheusMetrics`, `GET /_prometheusMetricsHigh` |

This counter increments once per external collection scan operation within a query execution.

**Sample output:**
```
# TYPE n1ql_external_scans counter
n1ql_external_scans 42
```

---

## 15. Error Codes Reference

### Catalog Errors

| Code | Symbol | Description |
|------|--------|-------------|
| 12051 | `E_CB_CATALOG_EXISTS` | Catalog already exists |
| 12052 | `E_CB_CATALOG_NOT_FOUND` | Catalog not found |
| 12053 | `E_CB_CATALOG_CREATE` | Error while creating catalog |
| 12054 | `E_CB_CATALOG_ALTER` | Error while altering catalog |
| 12055 | `E_CB_CATALOG_DROP` | Error while dropping catalog |
| 12056 | `E_CB_CATALOG_GET` | Error while getting catalogs |
| 12057 | `E_CB_EXTERNAL_COLLECTION` | External collection error |

### CredentialStore Errors

| Code | Symbol | Description |
|------|--------|-------------|
| 12058 | `E_CB_CREDENTIALSTORE_EXISTS` | Credential store already exists |
| 12059 | `E_CB_CREDENTIALSTORE_NOT_FOUND` | Credential store not found |
| 12060 | `E_CB_CREDENTIALSTORE_CREATE` | Error while creating credential store |
| 12061 | `E_CB_CREDENTIALSTORE_ALTER` | Error while altering credential store |
| 12062 | `E_CB_CREDENTIALSTORE_DROP` | Error while dropping credential store |

### GRANT/REVOKE Role Errors

| Code | Symbol | Description |
|------|--------|-------------|
| 5251 | `E_ROLE_REQUIRES_CATALOG` | Catalog role requires a catalog name |
| 5252 | `E_ROLE_REQUIRES_CREDENTIALSTORE` | CredentialStore role requires a credential store name |

---

## 16. QA Test Scenarios

### CredentialStore
- Create with valid/invalid WITH clause; verify 12060 on NS-Server rejection
- Create duplicate — expect 12058; create duplicate with `IF NOT EXISTS` — expect success
- Alter existing — verify updated fields; alter non-existent — expect 12059
- Drop existing; drop non-existent — expect 12059; drop with `IF EXISTS` — success
- Attempt operations without `PRIV_SECURITY_WRITE` — expect auth error

### Catalog
- CE build: any catalog DDL — expect enterprise feature error
- Create with missing TYPE/SOURCE/AT — expect field-empty error
- Create NESSIE source without AT clause — expect success (credential optional)
- Create non-NESSIE source without AT clause — expect field-empty error on credential
- Create duplicate — 12051; with `IF NOT EXISTS` — success
- Drop non-existent — 12052; with `IF EXISTS` — success
- Alter non-existent — 12052
- Without `PRIV_CLUSTER_ADMIN` — auth error

### External Collection
- Create with valid catalog+credential — verify queryable via SELECT
- Create without ON/AT — expect field-empty errors
- Create with non-existent catalog name — expect error from NS-Server
- DROP external collection — verify removed from system catalog
- Query `system:catalogs` and `system:catalogs_info` with varying privilege levels
- Verify `n1ql_external_scans` counter increments on each SELECT against external collection

### GRANT/REVOKE
- Grant `SELECT CATALOG` on catalog — user can query it
- Grant `CONSUME CREDENTIALSTORE` — user can use that credential in catalog operations
- Revoke and verify access denied
- Grant catalog privilege without ON clause — should parse error
- GRANT/REVOKE to GROUP vs USERS variants

---

## 17. Unsupported SQL Statements on External Collections

All restrictions are enforced at **planner time** (before execution). The error format is:

> `<statement type> is not supported on external collections`

where statement type uses spaces (e.g., `INSERT is not supported on external collections`).

### DML Mutations — all blocked

| Statement | Example | Error |
|-----------|---------|-------|
| `INSERT` | `INSERT INTO ext_coll VALUES ...` | `INSERT is not supported on external collections` |
| `UPDATE` | `UPDATE ext_coll SET x = 1 ...` | `UPDATE is not supported on external collections` |
| `DELETE` | `DELETE FROM ext_coll WHERE ...` | `DELETE is not supported on external collections` |
| `UPSERT` | `UPSERT INTO ext_coll VALUES ...` | `UPSERT is not supported on external collections` |
| `MERGE` | `MERGE INTO ext_coll USING ...` | `MERGE is not supported on external collections` |

### Index DDL — all blocked

| Statement | Error |
|-----------|-------|
| `CREATE PRIMARY INDEX ON ext_coll` | `CREATE PRIMARY INDEX is not supported on external collections` |
| `CREATE INDEX idx ON ext_coll(field)` | `CREATE INDEX is not supported on external collections` |
| `DROP INDEX ext_coll.idx` | `DROP INDEX is not supported on external collections` |
| `ALTER INDEX ext_coll.idx ...` | `ALTER INDEX is not supported on external collections` |
| `BUILD INDEX ON ext_coll(idx)` | `BUILD INDEXES is not supported on external collections` |

### Schema / Statistics Utilities — blocked

| Statement | Error |
|-----------|-------|
| `INFER ext_coll` | `INFER KEYSPACE is not supported on external collections` |
| `UPDATE STATISTICS FOR ext_coll(...)` | `UPDATE STATISTICS is not supported on external collections` |

Both are blocked at planner time via `getNameKeyspace` with `checkExt=true`.

### ADVISE — behavior depends on whether the collection exists

**Collection exists (is an external collection):**
`ADVISE` succeeds without error. Because external collections have no GSI indexes,
the advisor returns empty index recommendations.

```sql
-- Returns empty advice, not an error
ADVISE SELECT * FROM iceberg_flights WHERE origin = "SFO"
```

**Collection does not exist:**
If the collection has not been created yet, ADVISE treats it as a regular internal
collection and returns normal GSI index recommendations based on the query predicates.

```sql
-- Collection does not exist yet — advisor gives regular index recommendations
ADVISE SELECT * FROM new_ext_coll WHERE origin = "SFO"
```

This means ADVISE cannot distinguish a not-yet-created external collection from a
not-yet-created internal collection — it always gives regular index advice for
non-existent collections.

### SELECT Restrictions

| Scenario | Behavior |
|----------|----------|
| External collection in a correlated subquery | Planner error: `External collections are not supported in correlated subqueries` |
| `AT SNAPSHOT`/`AT TIMESTAMP` on a non-external collection | Planner error: `Snapshot options are only supported on external collections` |
| `AT SNAPSHOT` with a non-static expression (e.g., `$snap`) | Planner error: `SNAPSHOT expression must be a static value` |
| `AT TIMESTAMP` with a non-static expression | Planner error: `TIMESTAMP expression must be a static value` |
| `AT SNAPSHOT`/`AT TIMESTAMP` on a subquery | Parse-time fatal: `AT SNAPSHOT/TIMESTAMP is not allowed on subqueries.` |
| `AT SNAPSHOT`/`AT TIMESTAMP` on a parameter expression | Parse-time fatal: `AT SNAPSHOT/TIMESTAMP is not allowed on parameter expressions.` |
| `AT SNAPSHOT`/`AT TIMESTAMP` on a non-keyspace expression | Parse-time fatal: `AT SNAPSHOT/TIMESTAMP is only allowed on keyspace terms, not expressions.` |
| `USE INDEX` on an external collection | No index scans exist; planner generates ExternalScan regardless |

### ALTER COLLECTION — External Only

```sql
-- Only works on external collections; blocked on internal collections
ALTER COLLECTION default:bucket.scope.ext_collection WITH { ... }
```

Attempting `ALTER COLLECTION` on an internal collection returns:
> `ALTER COLLECTION is not supported on non-external collections`

---

## 18. Changed SQL++ SELECT Behavior for External Collections

### New: AT SNAPSHOT / AT TIMESTAMP clause (Time-Travel)

Appended to a keyspace reference in the `FROM` clause to query a specific historical version
of an Iceberg table.

#### Full Grammar

```sql
FROM <keyspace_path> [AS <alias>] AT SNAPSHOT <snapshot_id_expr>
FROM <keyspace_path> [AS <alias>] AT TIMESTAMP <timestamp_expr>
FROM <keyspace_path> [AS <alias>] AT (SNAPSHOT <snapshot_id_expr>)
FROM <keyspace_path> [AS <alias>] AT (TIMESTAMP <timestamp_expr>)
```

All four forms are equivalent in effect (parenthesized variants are aliases).

#### Syntax Examples

```sql
-- Query by Iceberg snapshot ID (integer literal — must be static)
SELECT * FROM default:travel-sample.inventory.iceberg_flights AS f
AT SNAPSHOT 5688794825202065870

-- Query by Unix millisecond timestamp (integer literal)
SELECT * FROM default:travel-sample.inventory.iceberg_flights AS f
AT TIMESTAMP 1744588800000

-- Query by RFC3339Nano timestamp string (string literal)
SELECT * FROM default:travel-sample.inventory.iceberg_flights AS f
AT TIMESTAMP "2025-04-14T00:00:00Z"

-- Multiple external collections in JOIN — each gets its own AT clause
SELECT f.*, s.name
FROM default:travel-sample.inventory.iceberg_flights AS f AT SNAPSHOT 12345678
JOIN default:travel-sample.inventory.iceberg_seats AS s AT TIMESTAMP 1744588800000
ON f.seat_id = s.id
```

#### Semantics and Constraints

| Constraint | Detail |
|------------|--------|
| Static values only | Expressions must be compile-time constants (literal numbers or strings). Named parameters (`$p`), positional parameters (`?`), subqueries, and functions are not allowed. |
| External collections only | Using these clauses on a regular KV-backed collection is a planner error. |
| Not in subqueries | `AT SNAPSHOT`/`AT TIMESTAMP` is forbidden inside subquery `FROM` clauses. |
| Snapshot ID | An `int64` integer identifying an exact Iceberg snapshot. |
| Timestamp | Either an `int64` (Unix epoch milliseconds) or an RFC3339Nano string (e.g., `"2025-04-14T00:00:00Z"`). |

---

### Plan Operator: ExternalScan

External collections **never** use `PrimaryScan`, `IndexScan`, or `Fetch`. The planner always
generates a single `ExternalScan` operator.

#### EXPLAIN output

```json
{
  "#operator": "ExternalScan",
  "namespace": "default",
  "bucket": "travel-sample",
  "scope": "inventory",
  "keyspace": "iceberg_flights",
  "as": "f",
  "filter": "(f.origin = \"SFO\")",
  "early_projection": ["origin", "dest", "flight_num"],
  "snapshot_id": "5688794825202065870",
  "optimizer_estimates": {
    "cost": -1,
    "cardinality": -1,
    "size": -1,
    "fr_cost": -1
  }
}
```

**Fields present only when applicable:**

| Field | Present when |
|-------|-------------|
| `as` | Query uses an alias |
| `subpaths` | Nested field access is needed |
| `early_projection` | SELECT references only specific top-level fields |
| `filter` | WHERE clause can be pushed down to Iceberg |
| `snapshot_id` | `AT SNAPSHOT` is specified |
| `snapshot_timestamp` | `AT TIMESTAMP` is specified |

---

### Filter Pushdown to Iceberg

The planner extracts the WHERE clause filter and passes it to the Iceberg layer before row
scanning. Not all filter expressions can be pushed down.

**Supported pushdown operators:**

| Operator | N1QL example |
|----------|-------------|
| Equality | `f.origin = "SFO"` |
| Inequality | `f.origin != "JFK"` |
| Comparison | `f.distance < 500`, `f.distance >= 100` |
| Logical AND | `f.origin = "SFO" AND f.year = 2024` |
| Logical OR | `f.origin = "SFO" OR f.dest = "JFK"` |
| Logical NOT | `NOT f.cancelled` |
| IN | `f.origin IN ["SFO", "LAX", "JFK"]` |
| IS NULL | `f.delay IS NULL` |
| IS NOT NULL | `f.delay IS NOT NULL` |
| LIKE | `f.flight_num LIKE "UA%"` |

**Not pushed down (evaluated in the query engine):**

- Nested field references (e.g., `f.address.city`) — only top-level fields are pushed
- Correlated expressions referencing other FROM aliases
- Functions, subqueries, or expressions involving multiple aliases

---

### Projection Pushdown (Early Projection)

When the SELECT list references only specific top-level fields, the planner sets
`early_projection` on the ExternalScan. Only those named columns are read from
Parquet/Iceberg files, avoiding full row deserialization.

```sql
-- Only "origin", "dest", "flight_num" are read from Iceberg files:
SELECT f.origin, f.dest, f.flight_num
FROM default:travel-sample.inventory.iceberg_flights AS f
WHERE f.origin = "SFO"
```

`SELECT *` disables early projection (all columns read).

---

### Parallelism

External collection queries always execute with `maxParallelism = 1`. The Iceberg scanner
is not parallelized at the query-service level. No `Parallel` operator wraps `ExternalScan`.

---

### Cost-Based Optimizer (CBO)

CBO is disabled for external collections. The planner does not use statistics or cost
estimates when an `ExternalScan` is involved. All optimizer estimates in EXPLAIN will
show `-1` for `cost`, `cardinality`, `size`, and `fr_cost`.

---

## 19. New Keywords (MB-68525)

Eight new keywords were added to the N1QL parser. All eight are **non-reserved** — they
are in the `permitted_identifiers` list, so they can be used as unquoted identifiers
(field names, aliases, keyspace names) without backtick-quoting in contexts where the
grammar does not expect them as keywords.

| Keyword | Used in |
|---------|---------|
| `CATALOG` | CREATE/ALTER/DROP CATALOG, GRANT/REVOKE ... CATALOG, system:catalogs |
| `CONSUME` | GRANT/REVOKE CONSUME CREDENTIALSTORE |
| `CREDENTIALSTORE` | CREATE/ALTER/DROP CREDENTIALSTORE, GRANT/REVOKE CONSUME CREDENTIALSTORE |
| `EXTERNAL` | CREATE EXTERNAL COLLECTION |
| `SNAPSHOT` | FROM ... AT SNAPSHOT \<expr\> |
| `SOURCE` | CREATE CATALOG ... SOURCE \<source\> |
| `TIMESTAMP` | FROM ... AT TIMESTAMP \<expr\> |
| `TYPE` | CREATE CATALOG ... TYPE \<type\> |

### QA implications

- All eight keywords can appear as unquoted field names, aliases, or keyspace names
  without causing parse errors.
- Examples:
  ```sql
  -- All valid — keywords used as identifiers
  SELECT CATALOG, TYPE, SOURCE FROM t
  SELECT t.CREDENTIALSTORE FROM t
  SELECT t.EXTERNAL FROM t
  SELECT t.SNAPSHOT, t.TIMESTAMP FROM t
  ```
