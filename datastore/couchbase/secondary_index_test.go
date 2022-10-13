//go:build ignore

/*
Copyright 2014-Present Couchbase, Inc.

Use of this software is governed by the Business Source License included in
the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
file, in accordance with the Business Source License, use of this software will
be governed by the Apache License, Version 2.0, included in the file
licenses/APL.txt.
*/

package couchbase

import "testing"
import "log"
import "reflect"

import "github.com/couchbase/query/expression/parser"
import "github.com/couchbase/query/expression"
import "github.com/couchbase/query/datastore"
import "github.com/couchbase/query/value"
import qp "github.com/couchbase/indexing/secondary/queryport"
import c "github.com/couchbase/indexing/secondary/common"
import "github.com/couchbase/indexing/secondary/protobuf"

type testingContext struct {
	t *testing.T
}

var testStatisticsResponse = &protobuf.StatisticsResponse{
	Stats: &protobuf.IndexStatistics{
		Count:      proto.Uint64(100),
		UniqueKeys: proto.Uint64(100),
		Min:        []byte(`"aaaaa"`),
		Max:        []byte(`"zzzzz"`),
	},
}
var testResponseStream = &protobuf.ResponseStream{
	Entries: []*protobuf.IndexEntry{
		&protobuf.IndexEntry{
			EntryKey: []byte(`["aaaaa"]`), PrimaryKey: []byte("key1"),
		},
		&protobuf.IndexEntry{
			EntryKey: []byte(`["aaaaa"]`), PrimaryKey: []byte("key2"),
		},
	},
}

var index *secondaryIndex
var qpServer *qp.Server

func init() {
	testUrl := server.GetIP(true) + ":9998"
	qpServer = startQueryport(testUrl, serverCallb)

	ns := &namespace{
		name:          "default",
		keyspaceCache: make(map[string]datastore.Keyspace),
	}
	ks := &keyspace{
		namespace: ns,
		name:      "default",
		indexes:   make(map[string]datastore.Index),
	}
	expr, err := parser.Parse(`gender`)
	if err != nil {
		log.Fatal(err)
	}
	equalKey := expression.Expressions{expr}
	expr, err = parser.Parse(`name`)
	if err != nil {
		log.Fatal(err)
	}
	rangeKey := expression.Expressions{expr}
	whereKey, err := parser.Parse("(30 < `age`)")
	if err != nil {
		log.Fatal(err)
	}
	index, _ = new2iIndex(
		"testindex", equalKey, rangeKey, whereKey, "gsi", ks)
	index.setHost([]string{testUrl})
}

func Test2iKeyspaceId(t *testing.T) {
	if index.KeyspaceId() != "default" {
		t.Fatal("failed KeyspaceId()")
	}
}

func Test2iId(t *testing.T) {
	if index.Id() != "testindex" {
		t.Fatal("failed Id()")
	}
}

func Test2iType(t *testing.T) {
	if index.Type() != "gsi" {
		t.Fatal("failed Type()")
	}
}

func Test2iSeekKey(t *testing.T) {
	equalKey := index.SeekKey()
	if len(equalKey) != 1 {
		t.Fatalf("failed SeekKey() - %v, expected 1", len(equalKey))
	} else if v := expression.NewStringer().Visit(equalKey[0]); v != "`gender`" {
		t.Fatalf("failed SeekKey() - %v, expected `gender`", v)
	}
}

func Test2iRangeKey(t *testing.T) {
	rangeKey := index.RangeKey()
	if len(rangeKey) != 1 {
		t.Fatalf("failed RangeKey() - %v, expected 1", len(rangeKey))
	} else if v := expression.NewStringer().Visit(rangeKey[0]); v != "`name`" {
		t.Fatalf("failed RangeKey() - %v, expected `name`")
	}
}

func Test2iCondition(t *testing.T) {
	whereKey := index.Condition()
	v := expression.NewStringer().Visit(whereKey)
	if v != "(30 < `age`)" {
		t.Fatalf("failed Condition() - %v, expected (30 < `age`)", v)
	}
}

func Test2iStatistics(t *testing.T) {
	c.LogIgnore()
	low, high := value.NewValue("aaaa"), value.NewValue("zzzz")
	span := &datastore.Span{
		Range: &datastore.Range{
			Low:       value.Values{low},
			High:      value.Values{high},
			Inclusion: datastore.BOTH,
		},
	}
	out, err := index.Statistics("", span)
	if err != nil {
		t.Fatal(err)
	}
	ref := &statistics{
		count:      100,
		uniqueKeys: 100,
		min:        []uint8{0x22, 0x61, 0x61, 0x61, 0x61, 0x61, 0x22},
		max:        []uint8{0x22, 0x7a, 0x7a, 0x7a, 0x7a, 0x7a, 0x22},
	}
	if reflect.DeepEqual(out, ref) == false {
		t.Fatalf("failed index.Statistics() %#v", out)
	}
}

func Test2iScanRange(t *testing.T) {
	c.LogIgnore()
	//c.SetLogLevel(c.LogLevelDebug)
	low, high := value.NewValue("aaaa"), value.NewValue("zzzz")
	span := &datastore.Span{
		Range: &datastore.Range{
			Low:       value.Values{low},
			High:      value.Values{high},
			Inclusion: datastore.BOTH,
		},
	}
	conn := datastore.NewIndexConnection(nil)

	go index.Scan("", span, false, 10000, conn)

	count := 0
loop:
	for {
		_, ok := conn.Sender.GetEntry()
		if !ok {
			break loop
		}
		count++
	}
	if count != 20000 {
		t.Fatal("failed ScanRange() - ", count)
	}
}

func Test2iScanEntries(t *testing.T) {
	c.LogIgnore()
	//c.SetLogLevel(c.LogLevelDebug)
	conn := datastore.NewIndexConnection(nil)

	go index.ScanEntries(10000, conn)

	count := 0
loop:
	for {
		_, ok := conn.Sender().GetEntry()
		if !ok {
			break loop
		}
		count++
	}
	if count != 20000 {
		t.Fatal("failed ScanEntries() - ", count)
	}
}

func Test2iClose(t *testing.T) {
	qpServer.Close()
}

func startQueryport(laddr string, callb qp.RequestHandler) *qp.Server {
	s, err := qp.NewServer(laddr, callb, c.SystemConfig.Clone())
	if err != nil {
		log.Fatal(err)
	}
	return s
}

func sendResponse(count int, respch chan<- interface{}, quitch <-chan interface{}) {
	i := 0
loop:
	for ; i < count; i++ {
		select {
		case respch <- testResponseStream:
		case <-quitch:
			break loop
		}
	}
}

func serverCallb(
	req interface{}, respch chan<- interface{}, quitch <-chan interface{}) {

	switch req.(type) {
	case *protobuf.StatisticsRequest:
		resp := testStatisticsResponse
		select {
		case respch <- resp:
			close(respch)

		case <-quitch:
			log.Fatal("unexpected quit", req)
		}

	case *protobuf.ScanRequest:
		sendResponse(10000, respch, quitch)
		close(respch)

	case *protobuf.ScanAllRequest:
		sendResponse(10000, respch, quitch)
		close(respch)

	default:
		log.Fatal("unknown request", req)
	}
}
