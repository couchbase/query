#/usr/bin/python

#
# This script validate results with the postgress 11
#    Download https://www.postgresql.org/download/linux/redhat/
#    Install server and clients
#    sudo -u postgres psql
#        CREATE USER root;
#        ALTER USER root SUPERUSER CREATEDB CREATEROLE;
#        CREATE DATABASE root;
#        CREATE DATABASE test;
#        du;
# Add following line in .bash_profile logout and login
#       export PATH=$PATH:/usr/pgsql-11/bin
#
#I used python 2.7.9
#   pip install --upgrade pip
#   pip install psycopg2-binary
#
#   modify cfg dictionary entires for load the data etc
#   install couchbase
#   create test bucket and create primary index or appropriate indexes

import sys
import math
import json
import time
import random
import requests
import urllib3
import decimal
import multiprocessing

cfg_aggs = [
# default "olaponly":False, "wframe": True, "distinct": True, "validate": True, "minargs":1, "maxargs":1, "optoby":True, "uoby": False
              { "name": "SUM",         },
              { "name": "COUNT",       },
              { "name": "MIN",         },
              { "name": "MAX",         },
              { "name": "AVG",         },
              { "name": "STDDEV",      },
              { "name": "STDDEV_POP",  },
              { "name": "STDDEV_SAMP", },
              { "name": "VARIANCE",    },
              { "name": "VAR_POP",     },
              { "name": "VAR_SAMP",    },
              { "name": "ROW_NUMBER",  "olaponly":True, "minargs":0, "maxargs":0, "distinct": False, "optoby":False, "uoby":True, "wframe": False},
              { "name": "RANK",        "olaponly":True, "minargs":0, "maxargs":0, "distinct": False, "optoby":False, "wframe": False},
              { "name": "DENSE_RANK",  "olaponly":True, "minargs":0, "maxargs":0, "distinct": False, "optoby":False, "wframe": False},
              { "name": "PERCENT_RANK","olaponly":True, "minargs":0, "maxargs":0, "distinct": False, "optoby":False, "wframe": False},
              { "name": "CUME_DIST",   "olaponly":True, "minargs":0, "maxargs":0, "distinct": False, "optoby":False, "wframe": False},
              { "name": "NTILE",       "olaponly":True, "minargs":1, "maxargs":1, "distinct": False, "optoby":False, "uoby":True, "wframe": False},
              { "name": "FIRST_VALUE", "olaponly":True, "minargs":1, "maxargs":1, "distinct": False, "uoby":True, "nulls":True},
              { "name": "LAST_VALUE",  "olaponly":True, "minargs":1, "maxargs":1, "distinct": False, "uoby":True, "nulls":True},
              { "name": "NTH_VALUE",   "olaponly":True, "minargs":2, "maxargs":2, "distinct": False, "uoby":True, "nulls":True, "from":True},
              { "name": "LAG",         "olaponly":True, "minargs":1, "maxargs":3, "distinct": False, "optoby":False, "uoby":True, "wframe": False, "nulls":True},
              { "name": "LEAD",        "olaponly":True, "minargs":1, "maxargs":3, "distinct": False, "optoby":False, "uoby":True, "wframe": False, "nulls":True},
              { "name": "RATIO_TO_REPORT","validate": False, "olaponly":True, "minargs":1, "maxargs":1, "distinct": False, "nooby":True, "wframe": False},
              { "name": "ARRAY_AGG",   "validate": False},
              { "name": "COUNTN",      "validate": False},
              { "name": "MEAN",        "validate": False},
              { "name": "MEDIAN",      "validate": False},
         ]

cfg_wframe = [
# default "validate": True, "nvals":0
             {"name": "ROWS UNBOUNDED PRECEDING"},
             {"name": "ROWS CURRENT ROW"},
             {"name": "ROWS %d PRECEDING", "nvals":1},

             {"name": "ROWS BETWEEN UNBOUNDED PRECEDING AND %d PRECEDING", "nvals":1},
             {"name": "ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW"},
             {"name": "ROWS BETWEEN UNBOUNDED PRECEDING AND %d FOLLOWING", "nvals":1},
             {"name": "ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING"},
             {"name": "ROWS BETWEEN CURRENT ROW AND CURRENT ROW"},
             {"name": "ROWS BETWEEN CURRENT ROW AND %d FOLLOWING", "nvals":1},
             {"name": "ROWS BETWEEN CURRENT ROW AND UNBOUNDED FOLLOWING"},
             {"name": "ROWS BETWEEN %d PRECEDING AND %d PRECEDING", "nvals":2},
             {"name": "ROWS BETWEEN %d PRECEDING AND CURRENT ROW", "nvals":1},
             {"name": "ROWS BETWEEN %d PRECEDING AND %d FOLLOWING", "nvals":2},
             {"name": "ROWS BETWEEN %d PRECEDING AND UNBOUNDED FOLLOWING", "nvals":1},
             {"name": "ROWS BETWEEN %d FOLLOWING AND %d FOLLOWING", "nvals":2},
             {"name": "ROWS BETWEEN %d FOLLOWING AND UNBOUNDED FOLLOWING", "nvals":1},

             {"name": "RANGE UNBOUNDED PRECEDING"},
             {"name": "RANGE CURRENT ROW"},
             {"name": "RANGE %d PRECEDING", "nvals":1, "validate": 110000, "noby":1},
             {"name": "RANGE BETWEEN UNBOUNDED PRECEDING AND %d PRECEDING", "nvals":1, "validate": 110000, "noby":1},
             {"name": "RANGE BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW"},
             {"name": "RANGE BETWEEN UNBOUNDED PRECEDING AND %d FOLLOWING", "nvals":1, "validate": 110000, "noby":1},
             {"name": "RANGE BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING"},
             {"name": "RANGE BETWEEN CURRENT ROW AND CURRENT ROW"},
             {"name": "RANGE BETWEEN CURRENT ROW AND %d FOLLOWING", "nvals":1, "validate": 110000, "noby":1},
             {"name": "RANGE BETWEEN CURRENT ROW AND UNBOUNDED FOLLOWING"},
             {"name": "RANGE BETWEEN %d PRECEDING AND %d PRECEDING", "nvals":2, "validate": 110000, "noby":1},
             {"name": "RANGE BETWEEN %d PRECEDING AND CURRENT ROW", "nvals":1, "validate": 110000, "noby":1},
             {"name": "RANGE BETWEEN %d PRECEDING AND %d FOLLOWING", "nvals":2, "validate": 110000, "noby":1},
             {"name": "RANGE BETWEEN %d PRECEDING AND UNBOUNDED FOLLOWING", "nvals":1, "validate": 110000, "noby":1},
             {"name": "RANGE BETWEEN %d FOLLOWING AND %d FOLLOWING", "nvals":2, "validate": 110000, "noby":1},
             {"name": "RANGE BETWEEN %d FOLLOWING AND UNBOUNDED FOLLOWING", "nvals":1, "validate": 110000, "noby":1},

             {"name": "GROUPS UNBOUNDED PRECEDING", "validate": 110000},
             {"name": "GROUPS CURRENT ROW", "validate": 110000},
             {"name": "GROUPS %d PRECEDING", "nvals":1, "validate": 110000},
             {"name": "GROUPS BETWEEN UNBOUNDED PRECEDING AND %d PRECEDING", "nvals":1, "validate": 110000},
             {"name": "GROUPS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW", "validate": 110000},
             {"name": "GROUPS BETWEEN UNBOUNDED PRECEDING AND %d FOLLOWING", "nvals":1, "validate": 110000},
             {"name": "GROUPS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING", "validate": 110000},
             {"name": "GROUPS BETWEEN CURRENT ROW AND CURRENT ROW", "validate": 110000},
             {"name": "GROUPS BETWEEN CURRENT ROW AND %d FOLLOWING", "nvals":1, "validate": 110000},
             {"name": "GROUPS BETWEEN CURRENT ROW AND UNBOUNDED FOLLOWING", "validate": 110000},
             {"name": "GROUPS BETWEEN %d PRECEDING AND %d PRECEDING", "nvals":2, "validate": 110000},
             {"name": "GROUPS BETWEEN %d PRECEDING AND CURRENT ROW", "nvals":1, "validate": 110000},
             {"name": "GROUPS BETWEEN %d PRECEDING AND %d FOLLOWING", "nvals":2, "validate": 110000},
             {"name": "GROUPS BETWEEN %d PRECEDING AND UNBOUNDED FOLLOWING", "nvals":1, "validate": 110000},
             {"name": "GROUPS BETWEEN %d FOLLOWING AND %d FOLLOWING", "nvals":2, "validate": 110000},
             {"name": "GROUPS BETWEEN %d FOLLOWING AND UNBOUNDED FOLLOWING", "nvals":1, "validate": 110000}

             ]

cfg_wframe_exclude = [
             {"name": "", "validate":True},
             {"name": "EXCLUDE NO OTHERS", "validate":110000},
             {"name": "EXCLUDE CURRENT ROW", "validate":110000},
             {"name": "EXCLUDE TIES", "validate":110000},
             {"name": "EXCLUDE GROUP", "validate":110000}
             ]

cfg = {
       "nqueries": 1000000,
       "pgsql_connstr": "dbname='test' user='root'",
       "n1ql_connstr":'http://127.0.0.1:8093',
       "tabname":"test",
       "nrecords":20,
       "ARGS": None,
       "prefix":"",
       "aggs": cfg_aggs,
       "collation" : [""," ASC", " DESC"],
       "nullspos" : [""," NULLS FIRST", " NULLS LAST"],
       "docfields": ["c0", "c1", "c2", "c3", "c4", "c5", "c6", "c7", "c8", "c9"],
       "ufields": ["uc"],
       "loaddata": False,
       "onestmt": "",
       "debug": True,
       "ndebug": 1000,
       "query_nfields": [2,9],
       "query_naggs": 10,
       "round":6,
       "validate": True,
       "wframe": True,
       "cfg_wframe": cfg_wframe,
       "wframe_range": [0,4],
       "wframe_exclude": True,
       "cfg_wframe_exclude": cfg_wframe_exclude,
       "defarg":9999,
       "nth":9
      }

if cfg["validate"]:
     import psycopg2
     import psycopg2.extras

def generate_docs(n, fields, ufields):
    records = []
    for nr in range(0,n):
        record = {}
        for nc in range(0,len(fields)):
            colname = fields[nc]
            record[colname] = (nc * 100) + random.randint(1,n/4)
        for nc in range(0,len(ufields)):
            colname = ufields[nc]
            record[colname] = nr+1
        records.append(record)
    return records

def order_by(p, o, fields, ufields):
    collation = cfg["collation"]
    nullspos = cfg["nullspos"]
    n = random.randint(1,o)
    if n == 0:
        return ""
    s = " ORDER BY "
    for i in range (0, n):
        if i != 0 :
            s += ", "
        s += fields[p+i]
 
        s += collation[random.randint(0,len(collation)-1)]
        s += nullspos[random.randint(0,len(nullspos)-1)]

    for f in ufields:
        s += ", " + f
    return s

def partition_by(p, fields):
    n = random.randint(0,p)
    if n == 0:
        return ""
    s = "PARTITION BY "
    for i in range (0, n):
        if i != 0 :
            s += ", "
        s += fields[i]
    return s

def window_frame_exclude(qvalidate):
    if cfg.get("wframe_exclude") != False and random.randint(0, 5) != 0 :
         wfexclude = cfg["cfg_wframe_exclude"]
         wfe = wfexclude[random.randint(0, len(wfexclude)-1)]
         wvalidate = wfe.get("validate")
         if qvalidate != True or wvalidate == None or wvalidate == True or cfg["pgsqlver"] >= wvalidate:
              return " " + wfe["name"] + " "

    return ""

def window_frame(agg, qvalidate):
    if cfg.get("wframe") == False or agg.get("wframe") == False or random.randint(0, 10) == 0 :
        return "", None

    wframe = cfg["cfg_wframe"]
    for i in range(0,len(wframe)):
         wf = wframe[random.randint(0, len(wframe)-1)]
         wvalidate = wf.get("validate")
         if qvalidate == True and (wvalidate == False or (isinstance(wvalidate, int) and cfg["pgsqlver"] < wvalidate)):
               continue
         wfname = wf["name"]
         if wf.get("nvals") != None :
             nvals = wf["nvals"]
             if nvals == 1:
                 v1 = random.randint(cfg["wframe_range"][0],cfg["wframe_range"][1])
                 wfname = wfname %(v1)
             elif nvals == 2:
                 v1 = random.randint(cfg["wframe_range"][0],cfg["wframe_range"][1])
                 v2 = random.randint(cfg["wframe_range"][0],cfg["wframe_range"][1])
                 wfname = wfname %(v1, v2)
         return " " + wfname + " " + window_frame_exclude(qvalidate), wf.get("noby")

    return "", None
    

def over_clause(agg, fields, qvalidate):
    s = "OVER("
    so = 0
    if agg.get("optoby") == False:
        so = 1
    p = random.randint(0,len(fields)-so)
    o = random.randint(so,len(fields)-p)
    if agg.get("nooby") == True:
        o = 0
    s += partition_by(p, fields)
    if o > 0:
       wframe, o1 = window_frame(agg, qvalidate)
       if o1 == 1:
            o = 1
       ufields = []
       if wframe.startswith(" ROWS"):
           ufields = cfg["ufields"]
       elif agg.get("uoby") == True and wframe == "":
           ufields = cfg["ufields"]
       elif qvalidate == True and agg.get("uoby") == True and wframe != "":
           fields = cfg["ufields"]
           p=0
           o=1
       s += order_by(p, o, fields, ufields)
       s += wframe
    elif agg.get("uoby") == True:
       s += order_by(0, 1, cfg["ufields"], [])
    s += ") "
    return s
     
def get_agg(olap, qvalidate):
    aggs = cfg["aggs"]
    for i in range(0,len(aggs)):
         agg = aggs[random.randint(0, len(aggs)-1)]
         if not olap and agg.get("olaponly") == True:
             continue
         if qvalidate == False:
             return agg
         elif qvalidate == True and agg.get("validate") != False:
             return agg
         elif qvalidate == None and (agg.get("validate") == False or agg.get("distinct") != False):
             return agg
    return aggs[0]

def aggregate(n,fields, olap, distinct, qvalidate):
    agg = get_agg(olap, qvalidate)
    agg_modifier = ""
    minargs = agg.get("minargs")
    maxargs = agg.get("maxargs")
    if minargs == None:
        minargs = 1
    if maxargs == None:
        maxargs = 1
    
    if distinct and minargs == 1 and maxargs == 1 and agg.get("distinct") != False and (random.randint(0,1) == 1 or qvalidate == None):
        agg_modifier = "DISTINCT "
    
    s = agg["name"]
    s += " ("
    s += agg_modifier
    nargs = random.randint(minargs, maxargs)
    for i in range (0, nargs):
          if i == 0:
               if agg["name"] == "NTILE":
                   s += str(random.randint(1, 9))
               else:
                   s += fields[random.randint(0, len(fields)-1)]
          elif i == 1:
               s += "," + str (random.randint(1, cfg["nth"]))
          else:
               if agg["name"] != "NTH_VALUE" or not qvalidate:
                    s += "," +  str (cfg["defarg"])

    s += ") "

    alias = "AS " + "oagg_" + str(n)
    return agg, s, alias

def regular_aggregate(n, fields):
    agg, s, alias = aggregate(n, fields, False, True, qvalidate)
    return s  + alias

def analytic_aggregate(n, fields, qvalidate):
    agg, s, alias = aggregate(n, fields, True, (qvalidate != True), qvalidate)
    if agg.get("from") == True and qvalidate != True:
        rv = random.randint(0,2)
        if rv == 1:
            s += "FROM FIRST "
        elif rv == 2:
            s += "FROM LAST "

    if agg.get("nulls") == True and qvalidate != True:
        rv = random.randint(0,2)
        if rv == 1:
            s += "RESPECT NULLS "
        elif rv == 2:
            s += "IGNORE NULLS "

    return s + over_clause(agg, fields, qvalidate) + alias

def projection(fields, qvalidate):
    n = random.randint(0, cfg["query_naggs"])+1
    s = ""
    for i in range (0, len(fields)):
        if i != 0 :
            s += ", "
        s += fields[i]

    for i in range (0, n):
        s += ", "
        s += analytic_aggregate(i, fields, qvalidate)
    return s
    
def generate_fields(ofields):
    n = random.randint(cfg["query_nfields"][0], cfg["query_nfields"][1])
    if n > len(ofields):
       n = len(ofields)

    thisdict={}
    while len(thisdict) < n:
        j =  random.randint(0,len(ofields)-1)
        thisdict[ofields[j]] = ofields[j]
    fields = []
    for w in thisdict:
        fields.append(w)
    return sorted(fields)

def generate_statement(tabname, ofields, validate):
    fields = generate_fields(ofields)
    qvalidate = validate
    if validate and random.randint(0,100) == 0:
        qvalidate = None

    s = cfg["prefix"]
    s += " SELECT " + projection(fields, qvalidate)
    s += " FROM " + tabname
    s += ";"
    return s, qvalidate == True

def compare(s, t, validate):
    if not validate:
         if len(s) != cfg["nrecords"]:
             return False, s
         else:
             return True, None

    t = list(t)
    try:
        for elem in s:
            t.remove(elem)
    except ValueError:
        return False, elem
    return not t, t

def pgsql_load(conn, tabname, records, validate):
    if not validate:
         return

    cursor = conn.cursor(cursor_factory=psycopg2.extras.DictCursor)
    cursor.execute("DROP TABLE IF EXISTS test")
    cols = sorted(records[0].keys())
    coldef = [col + " int" for col in cols]
    query = "CREATE TABLE " + tabname + " (" + ", ".join(coldef) + ")"
    try:
        cursor.execute(query)
    except:
        print (query, "execute failed")

    cols_str = ", ".join(cols)
    vals_str_list = ["%s"] * len(cols)
    vals_str = ", ".join(vals_str_list)
    for record in records:
        vals = [record[x] for x in cols]
        query = "INSERT INTO test ({cols}) VALUES ({vals_str})".format(cols = cols_str, vals_str = vals_str)
        try:
            cursor.execute(query, vals)
        except:
            print (query, "execute failed")
    conn.commit()

def pgsql_conn(connstr, validate):
    if not validate:
         return None

    try:
        conn=psycopg2.connect(connstr)
    except:
        print "I am unable to connect to the database."
    return conn

def pgsql_version(conn, validate):
    rows = pgsql_execute(conn,"SHOW server_version_num", validate)
    if rows != None:
        return int(rows[0].get("server_version_num"))
    else:
        return 0


def pgsql_execute(conn, query, validate):
    if not validate:
         return None

    cursor = conn.cursor(cursor_factory=psycopg2.extras.DictCursor)
    try:
        cursor.execute(query)
    except:
        print "cursor open failed"

    colnames=[]
    for c in cursor.description:
         colnames.append(c.name)
    rows = []

    pgsqlrows = cursor.fetchall()
    for pgsqlrow in pgsqlrows:
        row = {}
        for c in colnames:
             row[c]=pgsqlrow[c]
             if isinstance(row[c], decimal.Decimal):
                 row[c] = float(row[c])
             if isinstance(row[c], float) and cfg.get("round") != None:
                 row[c] = round(row[c], cfg["round"])
        rows.append(row)
    cursor.close()
    return rows

def n1ql_connection(url):
    conn = urllib3.connection_from_url(url)
    return conn

def n1ql_generate_request(stmt):
    stmt = {'statement': stmt}
    stmt['max_parallelism'] = 1
    stmt['creds'] = '[{"user":"Administrator","pass":"password"}]'
    stmt['timeout'] = '120s'
    if cfg["ARGS"]:
        stmt['args'] =cfg["ARGS"]
    return stmt

def n1ql_load(conn, tabname, records):
    i = 0
    n1ql_execute(conn, 'DELETE FROM '+ tabname+ ';')
    for record in records:
        i = i+1
        key = "k" + str(i).zfill(9)
        query = 'INSERT INTO ' + tabname + ' VALUES("' + key + '",' + json.JSONEncoder().encode(record) + ');'
        n1ql_execute(conn, query)

def n1ql_round_float(records, validate):
    if not validate:
         return records

    if cfg.get("round") == None:
       return records

    rows = []
    for record in records:
        row = {} 
        for k in record.keys():
             row[k]=record[k]
             if isinstance(row[k], float):
                 row[k]= round(row[k],cfg["round"])
        rows.append(row)
    return rows
     
    
def n1ql_execute(conn, stmt):
    query = n1ql_generate_request(stmt)
    response = conn.request('POST', '/query/service', fields=query, encode_multipart=False)
    response.read(cache_content=False)
    body = json.loads(response.data.decode('utf8'))
    return body["results"]

def run_queries(tid,n1qlconn, pgsqlconn, count, tabname, fields, validate, debug=False):
    if cfg["onestmt"] != "":
         stmt = cfg["onestmt"]
         count = 1

    for i in range (0, count):
        if cfg["onestmt"] == "":
            stmt, qvalidate = generate_statement(tabname, fields, validate)
        if tid == 0:
            xxx = 3
#            print stmt, "\n"
        pgsql_records = pgsql_execute(pgsqlconn, stmt, qvalidate)
        n1ql_records = n1ql_execute(n1qlconn, stmt)
        n1ql_records = n1ql_round_float(n1ql_records, qvalidate)
        ok, mismatch = compare(n1ql_records, pgsql_records, qvalidate)
        if debug and i != 0 and (i%cfg["ndebug"]) == 0:
                print "Thread # ", tid, " Query #", i
        if not ok:
            print "Thread # ", tid, "Query # ", i, "====", stmt, "---", mismatch
            print (n1ql_records, pgsql_records)
    if debug:
        print "Thread # ", tid, " Query #", count


def run_init(validate):
    pgsqlconn = pgsql_conn(cfg["pgsql_connstr"], validate)
    cfg["pgsqlver"] = pgsql_version(pgsqlconn, validate)
    if cfg["loaddata"]:
         tabname = cfg["tabname"]
         records = generate_docs(cfg["nrecords"], cfg["docfields"], cfg["ufields"])
         n1qlconn = n1ql_connection(cfg["n1ql_connstr"])
         n1ql_load(n1qlconn, tabname, records)
         pgsql_load(pgsqlconn, tabname, records, validate)
    pgsqlconn.close()

def run_tid(tid, count, tabname, fields, validate, debug):
    time.sleep(tid*0.1)
    random.seed()
    n1qlconn = n1ql_connection(cfg["n1ql_connstr"])
    pgsqlconn = pgsql_conn(cfg["pgsql_connstr"], validate)
    run_queries(tid, n1qlconn, pgsqlconn, count, tabname, fields, validate, debug)

if __name__ == "__main__":
    tids = 1
    count = cfg["nqueries"]
    validate = cfg["validate"]

    if len(sys.argv) > 3 and sys.argv[3].lower() == 'false':
           validate = False
    if len(sys.argv) > 2:
         count = int(sys.argv[2])
    if len(sys.argv) > 1:
         tids = int(sys.argv[1])

    run_init(validate)

    print "\nUsage: python {0} <num of threads> <num of queries> <validate> ".format(sys.argv[0])
    print "\nThreads: {0}".format(tids), ", QUERIES: {0}".format(count), ", Postgress Ver: {0}".format(cfg["pgsqlver"]), ", RESULT VALIDATION: {0}\n".format(cfg["validate"]) 

    tcount = int(math.ceil(count/float(tids)))
    jobs = []
    for i in range(tids):
        j = multiprocessing.Process(target=run_tid, args=(i, tcount, cfg["tabname"], cfg["docfields"], cfg["debug"], validate))
        jobs.append(j)
        j.start()

    for j in jobs:
        j.join()

    print "Ran # ", tids*tcount, " Queries"

