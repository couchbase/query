//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package control

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/couchbase/cbauth/service"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
)

type Manager interface {
}

type ServiceMgr struct {
	mu *sync.RWMutex
	state

	nodeInfo *service.NodeInfo
	waiters  waiters
}

type state struct {
	rev      uint64
	changeID string
	servers  []service.NodeInfo
	eject    []service.NodeInfo
}

type waiter chan state
type waiters map[waiter]struct{}

func NewManager() Manager {
	var mgr *ServiceMgr
	logging.Debugf("server::NewManager entry")
	defer logging.Debuga(func() string { return fmt.Sprintf("server::NewManager exit: %v", mgr) })

	mgr = &ServiceMgr{
		mu: &sync.RWMutex{},
		state: state{
			rev:      0,
			servers:  nil,
			eject:    nil,
			changeID: "",
		},
		nodeInfo: &service.NodeInfo{
			NodeID:   service.NodeID(distributed.RemoteAccess().NodeUUID(distributed.RemoteAccess().WhoAmI())),
			Priority: service.Priority(0),
		},
	}

	mgr.waiters = make(waiters)

	mgr.setInitialNodeList()
	go mgr.registerWithServer() // Note: doesn't complete unless an error occurs

	return mgr
}

func (m *ServiceMgr) setInitialNodeList() {
	logging.Debugf("ServiceMgr::setInitialNodeList entry")
	defer logging.Debugf("ServiceMgr::setInitialNodeList exit")

	// our topology is just the list of nodes in the cluster (or ourselves)
	topology := distributed.RemoteAccess().GetNodeNames()

	info := make([]rune, 0, len(topology)*32)
	nodeList := make([]service.NodeInfo, 0)
	for _, nn := range topology {
		ps := prepareOperation(nn, "ServiceMgr::setInitialNodeList")
		uuid := distributed.RemoteAccess().NodeUUID(nn)
		nodeList = append(nodeList, service.NodeInfo{service.NodeID(uuid), service.Priority(0), ps})
		info = append(info, []rune(uuid)...)
		info = append(info, '[')
		info = append(info, []rune(nn)...)
		info = append(info, ']')
		info = append(info, ' ')
	}

	m.updateState(func(s *state) {
		s.servers = nodeList
	})
	if len(info) == 0 {
		info = append(info, []rune("no active nodes")...)
	}
	logging.Infof("Initial topology: %v", string(info))
}

func (m *ServiceMgr) registerWithServer() {
	// ns_server is looking for "n1ql-service_api" but we are "cbq-engine" so the default API tries to register as
	// "cbq-engine-service_api" (if we don't "massage" the CBAUTH_REVRPC_URL).  Instead we'll make use of a new API allowing
	// us to massage as necessary here, leaving the environment variable well alone.
	orig := os.Getenv("CBAUTH_REVRPC_URL") + "-service_api"
	url := strings.Replace(orig, "cbq-engine", "n1ql", 1)
	logging.Debuga(func() string { return fmt.Sprintf("ServiceMgr::registerWithServer url: %v", url) })
	err := service.RegisterManagerWithURL(m, url, nil)
	if err != nil {
		logging.Infof("ServiceMgr::registerWithServer error %v", err)
		m.Shutdown()
	}
}

func (m *ServiceMgr) GetNodeInfo() (*service.NodeInfo, error) {
	logging.Debuga(func() string { return fmt.Sprintf("ServiceMgr::GetNodeInfo: %v", m.nodeInfo) })
	return m.nodeInfo, nil
}

func (m *ServiceMgr) Shutdown() error {
	logging.Infof("ServiceMgr::Shutdown")
	os.Exit(0)
	return nil
}

// There are only active tasks on the master node and only whilst others are being gracefully stopped.  Here we rely on the
// /admin/shutdown REST interface to obtain information on the progress of the remote graceful shutdown
// Using this saves us from having to establish and handle another communication mechanism to feed back state to the master

func (m *ServiceMgr) GetTaskList(rev service.Revision, cancel service.Cancel) (*service.TaskList, error) {
	logging.Debuga(func() string { return fmt.Sprintf("ServiceMgr::GetTaskList entry: %v", DecodeRev(rev)) })

	curState, err := m.wait(rev, cancel)
	if err != nil {
		logging.Debuga(func() string { return fmt.Sprintf("ServiceMgr::GetTaskList exit: error: %v", err) })
		return nil, err
	}

	tasks := &service.TaskList{}

	tasks.Rev = EncodeRev(curState.rev)
	tasks.Tasks = make([]service.Task, 0)

	m.mu.Lock()
	changeID := m.state.changeID
	eject := m.eject
	m.mu.Unlock()
	if changeID != "" { // master
		running := 0
		if eject != nil {
			for _, e := range eject {
				if e.Opaque == nil {
					continue
				}
				res, err := distributed.RemoteAccess().ExecutePreparedAdminOp(e.Opaque, "GET", "", nil, distributed.NO_CREDS, "")
				if res != nil && err == nil {
					var status struct {
						Code int32 `json:"code"`
					}
					jerr := json.Unmarshal(res, &status)
					if jerr == nil && errors.ErrorCode(status.Code) != errors.E_SERVICE_SHUT_DOWN {
						running++
					} else {
						e.Opaque = nil
					}
				} else {
					e.Opaque = nil
				}
			}
		}
		if running == 0 {
			m.updateState(func(s *state) {
				s.changeID = ""
				s.eject = nil
			})
		} else {
			m.updateState(func(s *state) {
				s.eject = eject
			})
			progress := 0.1 // consider the initiation the starting 10%
			if len(eject) > 0 {
				progress += (float64(len(eject)-running) / float64(len(eject))) * 0.9
			}
			task := service.Task{
				Rev:          EncodeRev(0),
				ID:           fmt.Sprintf("shutdown/monitor/%s", curState.changeID),
				Type:         service.TaskTypeRebalance,
				Status:       service.TaskStatusRunning,
				Progress:     progress,
				IsCancelable: false,
			}
			tasks.Tasks = append(tasks.Tasks, task)
		}
	}

	logging.Debuga(func() string { return fmt.Sprintf("ServiceMgr::GetTaskList exit: %v", tasks) })

	return tasks, nil
}

// we don't support cancelling tasks
func (m *ServiceMgr) CancelTask(id string, rev service.Revision) error {
	logging.Debuga(func() string { return fmt.Sprintf("ServiceMgr::CancelTask entry %v %v", id, DecodeRev(rev)) })
	defer logging.Debugf("ServiceMgr::CancelTask exit")
	return service.ErrNotSupported
}

// return the current node list as understood by this process
func (m *ServiceMgr) GetCurrentTopology(rev service.Revision, cancel service.Cancel) (*service.Topology, error) {
	logging.Debuga(func() string { return fmt.Sprintf("ServiceMgr::GetCurrentTopology entry: rev = %v", DecodeRev(rev)) })

	state, err := m.wait(rev, cancel)
	if err != nil {
		logging.Debuga(func() string { return fmt.Sprintf("ServiceMgr::GetCurrentTopology exit: %v", err) })
		return nil, err
	}

	topology := &service.Topology{}

	topology.Rev = EncodeRev(state.rev)
	m.mu.Lock()
	if m.servers != nil && len(m.servers) != 0 {
		checkPrepareOperations(m.servers, "ServiceMgr::GetCurrentTopology")
		for _, s := range m.servers {
			topology.Nodes = append(topology.Nodes, s.NodeID)
		}
	} else {
		topology.Nodes = append(topology.Nodes, m.nodeInfo.NodeID)
	}
	m.mu.Unlock()
	topology.IsBalanced = true
	topology.Messages = nil

	logging.Debuga(func() string {
		return fmt.Sprintf("ServiceMgr::GetCurrentTopology exit: %v - %v eject: %v", DecodeRev(rev), topology, m.eject)
	})

	return topology, nil
}

func prepareOperation(node string, caller string) interface{} {
	host := distributed.RemoteAccess().UUIDToHost(node)
	ps, err := distributed.RemoteAccess().PrepareAdminOp(host, "shutdown", "", nil, distributed.NO_CREDS, "")
	if err != nil {
		logging.Debuga(func() string {
			return fmt.Sprintf("%v Failed to prepare admin operation for %v: %v", caller, host, err)
		})
	}
	return ps
}

func checkPrepareOperations(servers []service.NodeInfo, caller string) {
	for i := range servers {
		if servers[i].Opaque == nil {
			servers[i].Opaque = prepareOperation(string(servers[i].NodeID), caller)
		}
	}
}

const _PREPARE_RETRY_DELAY = 5 * time.Second

// when preparing all we're doing is updating the cached nodes list from the list of retained nodes
func (m *ServiceMgr) PrepareTopologyChange(change service.TopologyChange) error {
	logging.Debuga(func() string { return fmt.Sprintf("ServiceMgr::PrepareTopologyChange entry: %v", change) })
	defer logging.Debugf("ServiceMgr::PrepareTopologyChange exit")

	if change.Type != service.TopologyChangeTypeFailover && change.Type != service.TopologyChangeTypeRebalance {
		return service.ErrNotSupported
	}

	// for each node we know about, cache its shutdown URL
	info := make([]rune, 0, len(change.KeepNodes)*32)
	servers := make([]service.NodeInfo, 0)
	retry := false
	for _, n := range change.KeepNodes {
		ps := prepareOperation(string(n.NodeInfo.NodeID), "ServiceMgr::PrepareTopologyChange")
		if ps == nil {
			retry = true
		}
		servers = append(servers, service.NodeInfo{n.NodeInfo.NodeID, service.Priority(0), ps})
		info = append(info, []rune(n.NodeInfo.NodeID)...)
		info = append(info, '[')
		info = append(info, []rune(distributed.RemoteAccess().UUIDToHost(string(n.NodeInfo.NodeID)))...)
		info = append(info, ']')
		info = append(info, ' ')
	}

	if retry {
		// some failed.  This is likely a synchronisation issue where we've been invoked ahead of the cluster information
		// changing.  This is a blunt delay before retrying any that have failed.  If they still fail, we'll try again whenever
		// polled for the topologly and again prior to invocation.
		time.Sleep(_PREPARE_RETRY_DELAY)
		logging.Debuga(func() string { return "Retrying failed admin operation prepares" })
		checkPrepareOperations(servers, "ServiceMgr::PrepareTopologyChange")
	}

	// always keep a local list of servers that are no longer present; only the master will ever act upon this list
	var eject []service.NodeInfo
	m.mu.Lock()
	s := m.servers
	m.mu.Unlock()
	for _, o := range s {
		found := false
		for _, n := range servers {
			if o.NodeID == n.NodeID {
				found = true
				break
			}
		}
		if !found {
			eject = append(eject, o)
		}
	}
	if len(eject) == 0 {
		eject = nil
	} else {
		eject = eject[0:len(eject):len(eject)]
	}

	m.updateState(func(s *state) {
		s.servers = servers
		s.eject = eject
	})

	if len(info) == 0 {
		info = append(info, []rune("no active nodes")...)
	}
	logging.Infof("Topology changed to: %s", string(info))
	return nil
}

//const _FAILOVER_LIMIT = 120 * time.Second

// This is only invoked on the master which is then responsible for initiating changes on other nodes
func (m *ServiceMgr) StartTopologyChange(change service.TopologyChange) error {
	logging.Debuga(func() string { return fmt.Sprintf("ServiceMgr::StartTopologyChange %v", change) })
	defer logging.Debugf("ServiceMgr::StartTopologyChange exit")

	timeout := time.Duration(0)
	data := ""
	switch change.Type {
	case service.TopologyChangeTypeFailover:
		// if we want to implement a timeout, this is how we'd do it:
		// data = fmt.Sprintf("deadline=%v", time.Now().Add(_FAILOVER_LIMIT).Unix())
		// timeout = _FAILOVER_LIMIT
	case service.TopologyChangeTypeRebalance:
	default:
		return service.ErrNotSupported
	}

	m.mu.Lock()
	if m.eject != nil {
		info := make([]rune, 0, len(m.eject)*32)
		eject := make([]service.NodeInfo, 0, len(m.eject))
		done := util.WaitCount{}
		mutex := &sync.Mutex{}
		// in parallel in case some take time to reach
		for _, e := range m.eject {
			go func() {
				if e.Opaque == nil {
					// if we failed to prepare it before now, we could well be too late but try again anyway
					e.Opaque = prepareOperation(string(e.NodeID), "ServiceMgr::StartTopologyChange")
				}
				_, err := distributed.RemoteAccess().ExecutePreparedAdminOp(e.Opaque, "POST", data, nil, distributed.NO_CREDS, "")
				if err == nil {
					mutex.Lock()
					if eject != nil {
						eject = append(eject, e)
						info = append(info, []rune(e.NodeID)...)
						info = append(info, ' ')
					}
					mutex.Unlock()
					logging.Debuga(func() string {
						return fmt.Sprintf("ServiceMgr::StartTopologyChange initiated shutdown down on '%s'", string(e.NodeID))
					})
				} else {
					logging.Infof("ServiceMgr::StartTopologyChange failed start shudown on '%s' (op:%v): %v", string(e.NodeID),
						e.Opaque, err)
				}
				done.Incr()
			}()
		}
		// wait for completion
		if !done.Until(int32(len(m.eject)), timeout) {
			mutex.Lock()
			eject = eject[:0] // don't report any tasks
			mutex.Unlock()
			logging.Infof("ServiceMgr::StartTopologyChange failed initiate shutdown on nodes within time limit.")
		}
		if len(eject) > 0 {
			logging.Infof("Topology change: shutdown initiated on: %s", string(info))
		} else {
			mutex.Lock()
			eject = nil
			mutex.Unlock()
		}
		m.updateStateLOCKED(func(s *state) {
			if len(eject) > 0 {
				s.changeID = change.ID
				s.eject = eject
			} else {
				s.changeID = ""
				s.eject = nil
			}
		})
	}
	m.mu.Unlock()

	return nil
}

type Cleanup struct {
	canceled bool
	f        func()
}

func NewCleanup(f func()) *Cleanup {
	return &Cleanup{
		canceled: false,
		f:        f,
	}
}

func (c *Cleanup) Run() {
	if !c.canceled {
		c.f()
		c.Cancel()
	}
}

func (c *Cleanup) Cancel() {
	c.canceled = true
}

func EncodeRev(rev uint64) service.Revision {
	ext := make(service.Revision, 8)
	binary.BigEndian.PutUint64(ext, rev)

	return ext
}

func DecodeRev(ext service.Revision) uint64 {
	if ext == nil {
		return 0
	}
	return binary.BigEndian.Uint64(ext)
}

func (m *ServiceMgr) notifyWaitersLOCKED() {
	s := m.copyStateLOCKED()
	for ch := range m.waiters {
		if ch != nil {
			ch <- s
		}
	}

	m.waiters = make(waiters)
}

func (m *ServiceMgr) addWaiterLOCKED() waiter {
	ch := make(waiter, 1)
	m.waiters[ch] = struct{}{}

	return ch
}

func (m *ServiceMgr) removeWaiter(w waiter) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.waiters, w)
}

func (m *ServiceMgr) updateState(body func(state *state)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updateStateLOCKED(body)
}

func (m *ServiceMgr) updateStateLOCKED(body func(state *state)) {
	body(&m.state)
	m.state.rev++

	m.notifyWaitersLOCKED()
}

func (m *ServiceMgr) wait(rev service.Revision, cancel service.Cancel) (state, error) {

	m.mu.Lock()

	unlock := NewCleanup(func() { m.mu.Unlock() })
	defer unlock.Run()

	currState := m.copyStateLOCKED()

	if rev == nil {
		return currState, nil
	}

	haveRev := DecodeRev(rev)
	if haveRev != m.rev {
		return currState, nil
	}

	ch := m.addWaiterLOCKED()
	unlock.Run()

	select {
	case <-cancel:
		m.removeWaiter(ch)
		return state{}, service.ErrCanceled
	case newState := <-ch:
		return newState, nil
	}
}

func (m *ServiceMgr) copyStateLOCKED() state {
	s := m.state

	return s
}
