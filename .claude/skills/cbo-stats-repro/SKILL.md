---
name: cbo-stats-repro
description: Reproduce a customer's CBO (cost-based optimizer) query plan in house by exporting their optimizer statistics and importing them into a local cluster. Use when asked to export/import CBO stats, N1QL_SYSTEM_BUCKET, N1QL_CBO_STATS, or the _system._query collection, or to reproduce a customer EXPLAIN plan locally.
---

# CBO stats export / import (reproduce a customer plan)

Couchbase has no schema, and the planner fetches index metadata from the indexer while it optimizes a query. To get the customer's plan locally you need:

1. **The customer's CBO stats documents**
2. **The customer's index definitions** (from cbcollect or from support). Bucket, scope and collection layout can be inferred from the CREATE INDEX statements.

Buckets and collections can stay **empty**. Only the definitions and the stats need to exist.

First, ask which server version the customer runs (or check the cbcollect). The layout of the stats differs by version:

| Version | Where the stats live | Section |
|---|---|---|
| < 7.6 | `N1QL_SYSTEM_BUCKET.N1QL_SYSTEM_SCOPE.N1QL_CBO_STATS` | A |
| ≥ 7.6 (trinity and later) | `<bucket>._system._query`, one per bucket | B |

Local defaults: `couchbase://localhost:8091`, `Administrator` / `password`. Never put customer bucket, collection or field names into Jiras, commits or test data.

---

## A. Pre-7.6 (N1QL_SYSTEM_BUCKET)

### Export (customer side)

```bash
cbexport json -c couchbase://<host_addr> -u Administrator -p password \
  -b N1QL_SYSTEM_BUCKET -f list -o /tmp/n1ql_system_bucket_export.json \
  --include-key docKey --scope-field system_scope --collection-field system_collection
```

- `--include-key docKey` is **required**. The planner finds stats through the primary index by document-key format.
- Do **not** use `scope` or `collection` as the field names. Every stats document already has `scope` and `collection` fields, and those would be overwritten.

### Import (local)

1. Create the customer's buckets, scopes, collections and indexes.
2. Make sure `N1QL_SYSTEM_BUCKET.N1QL_SYSTEM_SCOPE.N1QL_CBO_STATS` and its primary index `PRIMARY_IDX_CBO_STATS` exist. For example, run `UPDATE STATISTICS` once on any keyspace.
3. Import with the original keys:

```bash
cbimport json -c couchbase://<host_addr> -u Administrator -p password \
  -b N1QL_SYSTEM_BUCKET -g %docKey% -d file://./n1ql_system_bucket_export.json \
  -f list --scope-collection-exp %system_scope%.%system_collection%
```

4. Replace the customer's index IDs with the local ones:

```sql
UPDATE N1QL_SYSTEM_BUCKET.N1QL_SYSTEM_SCOPE.N1QL_CBO_STATS st1
SET st1.indexId =
  NVL(FIRST v.indexId FOR v IN
        ( SELECT META(st).id, s.id AS indexId
          FROM N1QL_SYSTEM_BUCKET.N1QL_SYSTEM_SCOPE.N1QL_CBO_STATS st
          JOIN system:indexes s USE HASH(build)
            ON s.name = st.indexName AND
               CASE WHEN s.bucket_id IS VALUED
                    THEN CONCAT2(".", s.bucket_id, s.scope_id, s.keyspace_id)
                    ELSE CONCAT2(".", s.keyspace_id, "_default", "_default")
               END = CONCAT2(".", st.`bucket`, st.`scope`, st.`collection`)
        )
      WHEN v.id = META(st1).id
      END, st1.indexId);
```

The index ID inside each document key only keeps the key unique. It does not have to match. The `indexId` field inside the document does have to match.

5. Run `EXPLAIN <query>`.

---

## B. 7.6+ (`_system._query` per bucket)

`N1QL_SYSTEM_BUCKET` is gone. During upgrade the stats migrate into each bucket's `_system._query` collection. **Repeat everything below for each bucket the problem query references.**

### Export (customer side, once per bucket)

```bash
cbexport json -c couchbase://<host>:8091 -u Administrator -p password \
  -b <bucket_name> -f list -o /tmp/<bucket_name>_cbostats_export.json \
  --include-key docKey --include-data _system._query \
  --scope-field system_scope --collection-field system_collection
```

If cbexport isn't available, use a query and save the output:

```sql
SELECT META(d).id AS docKey, d.* FROM <bucket_name>._system._query AS d;
```

Keep **only the `results` array** from the output, or the file won't be valid list-format JSON.

### Import (local, once per bucket)

You cannot import directly into a system collection, so stage the data in a temporary bucket. Inserting into `_system._query` needs full admin privileges, so this **does not work on Capella**.

The temporary bucket below is named `default`. If the customer's bucket is itself called `default`, use a different staging name.

**1. Create the target bucket.** Use the same name as the customer's bucket. It starts empty.

**2. Create the staging area.** Create bucket `default` with scope `my_system` and collection `my_query`.

**3. Import into staging:**

```bash
cbimport json -c couchbase://localhost:8091 -u Administrator -p password \
  -b default -g %docKey% -d file:///tmp/<bucket_name>_cbostats_export.json \
  -f list --scope-collection-exp my_system.my_query
```

**4. List the referenced scopes and collections, with the customer's scope IDs:**

```sql
SELECT `scope` || "." || `collection` AS `path`,
       SPLIT(docKey, "::")[1] AS scope_id, COUNT(*) AS cnt
FROM default.my_system.my_query
GROUP BY `scope` || "." || `collection`, SPLIT(docKey, "::")[1];
```

Create each of these scopes and collections in `<bucket_name>`.

**5. Get the local scope IDs.** The scope ID is part of the document key (`cbo::<8-hex scope id>...`). Two ways to get it:

- From the collections manifest. Each scope's `uid` is in hex; zero-pad it to 8 digits:
  ```bash
  curl -s -u Administrator:password http://localhost:8091/pools/default/buckets/<bucket_name>/scopes | jq '.scopes[] | {name, uid}'
  ```
- Or patch `system:keyspaces` locally to expose it. In `datastore/system/system_keyspace_keyspaces.go`, add `"scope_uid": scope.Uid(),` to the collection doc map, rebuild, and run:
  ```sql
  SELECT * FROM system:keyspaces WHERE `bucket` = "<bucket_name>";
  ```
  Don't commit this patch.

**6. Rewrite the document keys to use the new scope IDs.** Do this once per non-`_default` scope. In the example, `0000000e` is the customer's scope ID and `00000009` is the local one:

```sql
UPDATE default.my_system.my_query
SET docKey = REPLACE(docKey, "cbo::0000000e", "cbo::00000009")
WHERE `scope` = "<scope_name>";
```

The original doc uses `` `scope` != "_default" ``. That only works when there is a single non-default scope. With several scopes, filter on the specific scope.

**7. Find the collections that have index stats:**

```sql
SELECT DISTINCT RAW `scope` || "." || `collection`
FROM default.my_system.my_query
WHERE type = "index";
```

For each path returned:

```sql
SELECT `scope` || "." || `collection` AS `path`,
       SPLIT(docKey, "(index)")[1] AS indexId, indexName
FROM default.my_system.my_query
WHERE type = "index" AND `scope` = "<scope>" AND `collection` = "<collection>";
```

Then create each of those indexes in `<bucket_name>` from the customer's definitions. **Remove `"defer_build": true`** so the indexes actually build.

**8. Remap the index IDs.** Set the named parameter first. In cbq that's `\set -$bucket_name "<bucket_name>";`. In the REST API it's `$bucket_name=...`.

```sql
UPDATE default.my_system.my_query st1
SET st1.indexId =
  NVL(FIRST v.indexId FOR v IN
        ( SELECT META(st).id, s.id AS indexId
          FROM default.my_system.my_query st
          JOIN system:indexes s USE HASH(build)
            ON s.name = st.indexName AND
               CASE WHEN s.bucket_id IS VALUED
                    THEN CONCAT2(".", s.bucket_id, s.scope_id, s.keyspace_id)
                    ELSE CONCAT2(".", s.keyspace_id, "_default", "_default")
               END = CONCAT2(".", $bucket_name, st.`scope`, st.`collection`)
        )
      WHEN v.id = META(st1).id
      END, st1.indexId)
WHERE type = "index";
```

```sql
UPDATE default.my_system.my_query st1
SET st1.docKey = REPLACE(st1.docKey, SPLIT(st1.docKey, "(index)")[1], st1.indexId)
WHERE st1.type = "index";
```

**9. Clear the stats that local CREATE INDEX generated.** Otherwise you can get duplicate keys, especially for `_default`:

```sql
DELETE FROM system:dictionary WHERE `bucket` = $bucket_name;
```

**10. Copy the staged documents into the system collection:**

```sql
INSERT INTO <bucket_name>._system._query (KEY _k, VALUE _v)
SELECT d.docKey AS _k, d AS _v FROM default.my_system.my_query AS d;
```

**11. Flush the dictionary cache** so the stats reload from disk. Restarting the query service also works.

```sql
DELETE FROM system:dictionary_cache WHERE `bucket` = $bucket_name;
```

**12. Drop the staging bucket** (`default`).

After every referenced bucket is done, run `EXPLAIN <query>` and compare it with the customer's plan.

---

## Checks when the plan doesn't match

- `SELECT COUNT(*) FROM <bucket>._system._query;` should equal the number of exported docs for that bucket.
- `SELECT * FROM system:dictionary WHERE `bucket` = "<bucket>";` should show the customer's stats (distribution counts, avgDocSize, and so on).
- Every index document should have an `indexId` that exists in `system:indexes`. If any document still has the customer's ID, its index name or keyspace didn't match a local index.
- Every index must be **online** (not deferred) and must match the customer's definition exactly: keys, WHERE clause, partitioning.
- Check that CBO is on (`use_cbo`) and that you're on the same server version as the customer.
