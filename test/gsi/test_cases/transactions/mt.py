"""
Copyright 2021-Present Couchbase, Inc.

Use of this software is governed by the Business Source License included in
the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
file, in accordance with the Business Source License, use of this software will
be governed by the Apache License, Version 2.0, included in the file
licenses/APL.txt.
"""

#/usr/bin/python

import sys
import os
import math
import json
import time
import random
import requests
import urllib3
import decimal
import multiprocessing


cfg = {
       "n1ql_connstr":'http://127.0.0.1:8093',
       "keyspaces": [
                    {"bname":"b1"},
                    {"bname":"b1", "cname":"c1"},
                    {"bname":"b2", "sname":"s1","cname":"c1"},
                    {"bname":"b2", "sname":"s1","cname":"c2"},
                    {"bname":"b2", "sname":"s2","cname":"c3"}],
       "nrecords":10000,
       "cluster": {"qnodes":[{"connstr":"http://127.0.0.1:8093"}
                            ],
                   "docsperqnode": 20000,
                   "nclients": 4,
                   "docsperclient": 5000,
                   "nthreads": 20,
                   "docsperthread": 50,
                   "docsperstmt": [2,8]
                   },
       "allocateddocspertrans":100,
       "reuseinterval":5,
       "ARGS": None,
       "prefix":"",
       "docfields": ["total", "c0", "c1", "c2", "c3", "c4", "c5", "c6", "c7", "c8", "c9"],
       "ufields": [],
       "update_nfields": [2,8],
       "debug": True,
       "ndebug": 10,
       "ntrans":20000,
       "nstatements":5,
       "docsperstmt": 4,
       "loaddata": True,
       "conflict": False,
       "trans": True,
       "begin":"BEGIN WORK;",
       "rollback":"ROLLBACK WORK;",
       "commit":"COMMIT WORK;",
       "stmts":[
                "BEGIN WORK;",
                "COMMIT WORK;",
                "ROLLBACK WORK;",
                "",
                "",
                "INSERT INTO $1 AS i VALUES ($2,$3);",
                "UPSERT INTO $1 AS i VALUES ($2,$3);",
                "DELETE FROM $1 AS d USE KEYS $2;",
                "",
                "",
                "SELECT META().id AS k, s AS doc FROM %s AS s WHERE uid BETWEEN $1 AND $2;",
                "SELECT META().id, uid, nuid FROM %s AS s WHERE uid BETWEEN $1 AND $2;",
                "UPDATE %s AS u SET nuid = uid + 1000000 WHERE uid BETWEEN $1 AND $2;",
                "DELETE FROM %s AS d WHERE uid BETWEEN $1 AND $2 RETURNING RAW META(d).id;",
                "MERGE INTO %s AS t USING %s AS s ON t.uid = s.uid AND s.uid BETWEEN $1 AND $2 AND META(s).id = META(t).id WHEN MATCHED THEN UPDATE SET t.nuid = s.uid + 1000000;",
                "INSERT INTO %s AS i (KEY v.ikey, VALUE v) SELECT DISTINCT t AS v FROM %s AS t WHERE uid BETWEEN $1 AND $2 RETURNING RAW META(i).id;",
                "UPSERT INTO %s AS u (KEY v.ukey, VALUE v) SELECT DISTINCT t AS v FROM %s AS t WHERE uid BETWEEN $1 AND $2 RETURNING RAW META(u).id;"
               ],
}


txtimeout = "1m"

def n1ql_connection(url):
    conn = urllib3.connection_from_url(url)
    return conn

def n1ql_generate_request(stmt, posparam, txid, txtimeout, tximplicit=False):
#    stmt['max_parallelism'] = 1
    stmt['txtimeout'] = txtimeout
    stmt['creds'] = '[{"user":"Administrator","pass":"password"}]'
    if posparam:
        stmt['args'] = json.JSONEncoder().encode(posparam)

    if tximplicit:
        stmt['tximplicit'] = tximplicit
    elif txid != "":
        stmt['txid'] = txid
    return stmt

def n1ql_execute(conn, stmt, posparam, txid, txtimeout, tximplicit):
    query = n1ql_generate_request(stmt, posparam, txid, txtimeout, tximplicit)
    response = conn.request('POST', '/query/service', fields=query, encode_multipart=False)
    response.read(cache_content=False)
    body = json.loads(response.data.decode('utf8'))
#    print json.JSONEncoder().encode(body)
    return body

def prepare_query(conn, stmtObj):
    stmt = {'statement': 'PREPARE ' + stmtObj["name"] + " FROM " + stmtObj["stmt"] }
    body = n1ql_execute(conn, stmt, None, "", txtimeout, False)
#    print stmt, json.JSONEncoder().encode(body)
    name = str(body['results'][0]['name'])
    return {'prepared': '"' + name + '"'}

def generate_prepared_query (name):
    return {'prepared': '"' + name + '"'}

def get_txid(records):
    if len(records) == 1:
         return records[0]["txid"]
    return ""


def transaction_statements(coll_stmts):
    ts_stmts = []

    if cfg["trans"]:
        ts_stmts.append(coll_stmts[0])

    added = {}
    for nr in xrange(0,cfg["nstatements"]):
         rv = random.randint(10,len(coll_stmts)-1)
         stmt = coll_stmts[rv]
         if stmt["name"] not in added :
              ts_stmts.append(coll_stmts[rv])
              added[stmt["name"]] = True

    if cfg["trans"]:
         rv = random.randint(1,10)
         if rv != 2 :
            rv = 1
         ts_stmts.append(coll_stmts[rv])

    return ts_stmts

def n1ql_trans_execute(iter, tid, sid, conn, coll_stmts):
    ts_stmts = transaction_statements(coll_stmts)
    t0 = time.time()
    i = 0
    n = random.randint(cfg["cluster"]["docsperstmt"][0], cfg["cluster"]["docsperstmt"][1])
    s = random.randint(0, cfg["cluster"]["docsperthread"]-n-1)
    txid = ""
    posparam = []
    posparam.append(sid+s)
    posparam.append(sid+s+n)
    deleted = {}
    inserted = {}
    commit = False
    conflict = False
    if cfg["conflict"]:
        conflict = random.randint(0, 9) == 0

    expStatus = "success"

    for stmt in ts_stmts:
#         print stmt
         if stmt["name"] != "" :
              if conflict and "implicit" in stmt and stmt["implicit"]: 
                   body = n1ql_execute(conn, generate_prepared_query(stmt["name"]), posparam , "", txtimeout, True)
                   if body["status"] != "success" :
                       print "Iter # ", iter, "Thread # ", tid, "Implicit Stmt ", stmt, body
                   expStatus = "errors"

              if "ddocs" in stmt :
                   d1 = random.randint(0,n-1)
                   posparam[0] = posparam[0] + d1
                   posparam[1] = posparam[0] + d1
                   body = n1ql_execute(conn, generate_prepared_query(stmt["ddocs"]), posparam ,txid, txtimeout, False)
                   if body["status"] != "success" :
                       print "Iter # ", iter, "Thread # ", tid, "Query # ", i, "Stmts ", ts_stmts, "==sel==", body
                   else :
                       if stmt["collection"] not in deleted :
                             deleted[stmt["collection"]] = []
                       deleted[stmt["collection"]].extend(body["results"])

              body = n1ql_execute(conn, generate_prepared_query(stmt["name"]), posparam ,txid, txtimeout, False)
              status = body["status"]
              if stmt["name"] == "p001" :
                   if status == "success":
                       commit = True
                   elif status != expStatus :
                      print "Iter # ", iter, "Thread # ", tid, "Query # ", i, "Stmts ", ts_stmts, "====", status, body, posparam
                      quit()
              elif status != "success" :
                   print "Iter # ", iter, "Thread # ", tid, "Query # ", i, "Stmts ", ts_stmts, "====", status, body, posparam
                   quit()
              elif i == 0 and cfg["trans"]:
                   results = body["results"]
                   txid = get_txid(results)
                   if txid == "":
                       print "Iter # ", iter, "Thread # ", tid, "Query # ", i, "Stmts ", ts_stmts, "==no txid =="
              elif "iuundo" in stmt :
                   if stmt["collection"] not in inserted :
                       inserted[stmt["collection"]] = []
                   inserted[stmt["collection"]].extend(body["results"])

              i = i+1

    if commit :
#        print "deleted", deleted
#        print "inserted", inserted
        for d in deleted :
            args = []
            args.append(d)
            args.append("")
            args.append("")
            docs = deleted[d]
            for doc in docs :
                args[1] = doc["k"]
                args[2] = doc["doc"]
                body = n1ql_execute(conn, generate_prepared_query("p005"), args , "", txtimeout, False)
                if body["status"] != "success" :
                    print "Iter # ", iter, "Thread # ", tid, "Query # ", i, "Stmts ", ts_stmts, "==insert==",  body

        for d in inserted :
            args = []
            args.append(d)
            args.append("")
            args[1] = inserted[d]
            body = n1ql_execute(conn, generate_prepared_query("p007"), args , "", txtimeout, False)
            if body["status"] != "success" :
                print "Iter # ", iter, "Thread # ", tid, "Query # ", i, "Stmts ", ts_stmts, "==insert==", body

#    print "Iter # ", iter, "Thread # ", tid, "Query # ", i, "Stmts ", ts_stmts, "====", time.time() - t0

def run_queries(tid, sid, n1qlconn, count, coll_stmts, debug=False):
    t0 = time.time()
    t1 = time.time()
    for i in xrange (0, count):
        n1ql_trans_execute (i, tid, sid, n1qlconn, coll_stmts)
        if debug and i != 0 and (i%cfg["ndebug"]) == 0:
                print "Thread # ", tid, " Tx #", i, time.time() - t1
                t1 = time.time()
    if debug:
        print "Thread # ", tid, " Tx #", count, time.time() - t1

def run_tid(tid, sid, count, coll_stmts, debug):
    time.sleep(tid*0.1)
    random.seed()
    n1qlconn = n1ql_connection(cfg["n1ql_connstr"])
    run_queries(tid, sid, n1qlconn, count, coll_stmts, debug) 

def n1ql_load(conn, stmt, collections, start, ndocs, fields):
    for nr in range(0, ndocs):
        record = {}
        record["uid"] = start + nr
        record["nuid"] = start + nr
        for nc in xrange(0,len(fields)):
            colname = fields[nc]
            record[colname] = (nc * 100) + random.randint(1,100)
        record["comment"] = "comments".ljust(873, '-')

        args = []
        args.append("")
        args.append("")
        args.append("")

        for collection in collections:
             args[0] = collection
             args[1] = collection + "::k" + str(record["uid"]).zfill(9)
             record["ikey"] = collection + "::ik" + str(record["uid"]).zfill(9)
             record["ukey"] = collection + "::uk" + str(record["uid"]).zfill(9)
             args[2] = record
             n1ql_execute(conn, stmt, args, "", txtimeout, False)

def run_load_tid(tid, stmt, collections, ndocs, fields):
    conn = n1ql_connection(cfg["n1ql_connstr"])
    n1ql_load(conn, stmt, collections, tid*ndocs, ndocs, fields)

def create_collections(conn, buckets, scopes, collections):
    for c in collections:
        stmt = "DROP INDEX ix1 ON " + c + ";"
        n1ql_execute(conn, {"statement":stmt} , None, "", txtimeout, False)
    time.sleep(5)

    for b in buckets:
        os.system("couchbase-cli bucket-delete -c localhost -u Administrator -p password --bucket " + b )
        cmd = "couchbase-cli bucket-create -c localhost -u Administrator -p password --bucket "
        cmd += b
        cmd += " --bucket-type couchbase --bucket-ramsize 250 --bucket-replica 0 --enable-flush 1"
        os.system(cmd)
#        os.system("couchbase-cli bucket-flush -c localhost -u Administrator -p password --bucket " + b + " --force")

    os.system("couchbase-cli rebalance -c localhost -u Administrator -p password")

    time.sleep(5)
    for s in scopes:
        stmt = "CREATE SCOPE " + s + ";"
        n1ql_execute(conn, {"statement":stmt} , None, "", txtimeout, False)

    time.sleep(1)
    for c in collections:
        stmt = "CREATE COLLECTION " + c +  ";"
        n1ql_execute(conn, {"statement":stmt} , None, "", txtimeout, False)

    time.sleep(15)
    for c in collections:
        stmt = "CREATE INDEX ix1 ON " + c +  " (uid);"
        body = n1ql_execute(conn, {"statement":stmt} , None, "", txtimeout, False)
        if body["status"] != "success" :
            print stmt, body
    time.sleep(5)

def collection_statements(conn, collections):
    seqno = -1
    stmts = cfg["stmts"]
    pstmts = []
    for i in xrange(0, 9):
          s = stmts[i]
          seqno = seqno + 1
          name = "p" + str(seqno).zfill(3)
          if s == "" :
               stmt = {"stmt": s, "name": "", }
          else:
               stmt = {"stmt": s, "name": name, }
               prepare_query(conn, stmt)
          pstmts.append(stmt)

    for collection in collections:
          startcol = seqno + 1
          for i in xrange(10, len(stmts)):
              s = stmts[i]
              seqno = seqno + 1
              name = "p" + str(seqno).zfill(3)
              if i < 14 :
                  stmt = {"stmt": s % (collection) }
                  if i == 12 :
                      stmt["implicit"] = True
                  if i == 13 :
                      stmt["dundo"] = "p005"
                      stmt["ddocs"] = "p" + str(startcol).zfill(3)
              else:
                  stmt = {"stmt": s % (collection, collection) }
                  if i >= 15 :
                      stmt["iuundo"] = "p007"
              stmt["validate"] = "p" + str(startcol+1).zfill(3)
              stmt["name"] = name
              stmt["collection"] = collection
              prepare_query(conn, stmt)
              pstmts.append(stmt)

#    for p in pstmts:
#        print p, "\n"
    return pstmts

def run_init(nthreads):
    conn = n1ql_connection(cfg["n1ql_connstr"])
    buckets = []
    scopes = []
    collections = []

    for k in cfg["keyspaces"]:
          bname = k["bname"]
          buckets.append(bname)
          collection = bname
          if "cname" in k:
              if "sname" in k :
                   collection += "." + k["sname"]
                   scopes.append(collection)
              else:
                   collection += "._default"
              collection += "." + k["cname"]
          else:
              if "sname" in k:
                   collection += "." + k["sname"]
                   scopes.append(collection)
                   collection += "._default"
          collections.append(collection)

    if cfg["loaddata"]:
          create_collections(conn, buckets, scopes, collections)

    stmt = "DELETE FROM system:prepareds;"
    n1ql_execute(conn, {"statement":stmt} , None, "", txtimeout, False)
    time.sleep(1)

    coll_stmts = collection_statements(conn, collections)

    if cfg["loaddata"]:
         ndocs = cfg["nrecords"]/nthreads + 1
         stmt = generate_prepared_query("p006")
         fields = cfg["docfields"]
         jobs = []
         for i in xrange(0,nthreads):
             j = multiprocessing.Process(target=run_load_tid, args=(i, stmt, collections, ndocs, fields))
             jobs.append(j)
             j.start()

         for j in jobs:
             j.join()

    return collections, coll_stmts

if __name__ == "__main__":
    if len(sys.argv) > 2:
         cfg["nstatements"] = int(sys.argv[2])
    if len(sys.argv) > 1:
         cfg["cluster"]["nthreads"] = int(sys.argv[1])

    nthreads = cfg["cluster"]["nthreads"]
    nstmts = cfg["nstatements"]
    ndocs =  cfg["docsperstmt"]

    print "\nUsage: python {0} <num of threads> <num of statements per tran> ".format(sys.argv[0])
    print "\nThreads: {0}".format(nthreads), ", Statements: {0}".format(nstmts)


    collections, coll_stmts = run_init(nthreads)

    tcount = int(math.ceil(cfg["ntrans"]/float(nthreads)))
    jobs = []

    t0  = time.time()

    qid = 0
    cid = 0
    sid = qid * (cfg["cluster"]["docsperqnode"]) + cid * cfg["cluster"]["docsperclient"] 
    for tid in xrange(0,nthreads):
        sid =  sid + tid * cfg["cluster"]["docsperthread"]
        j = multiprocessing.Process(target=run_tid, args=(tid, sid, tcount, coll_stmts, cfg["debug"]))
        jobs.append(j)
        j.start()

    for j in jobs:
        j.join()

#    print "Ran # ", nthreads*tcount, " Txs (" + str(nstmts) + " Queries " + str(ndocs) + " updates per query)", " Time (secs) #", (time.time() -t0 )
    print "Ran # ", nthreads*tcount, " Txs " + " Time (secs) #", (time.time() -t0 )

