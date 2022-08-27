//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package control

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
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

	thisHost string
	enabled  bool
}

// to reduce the occasions when host is looked up from node ID, cache it here
type queryServer struct {
	host     string
	nodeInfo service.NodeInfo
}

type state struct {
	rev      uint64
	changeID string
	servers  []queryServer
	eject    []queryServer
}

type waiter chan state
type waiters map[waiter]struct{}

func NewManager(uuid string) Manager {
	var mgr *ServiceMgr
	logging.Debugf("server::NewManager entry. UUID: %v", uuid)
	defer logging.Debuga(func() string { return fmt.Sprintf("server::NewManager exit: %v", mgr) })

	if uuid == "" {
		logging.Infof("No UUID passed.  Will not register for topology awareness.")
		return nil
	}

	mgr = &ServiceMgr{
		mu: &sync.RWMutex{},
		state: state{
			rev:      0,
			servers:  nil,
			eject:    nil,
			changeID: "",
		},
		nodeInfo: &service.NodeInfo{
			NodeID:   service.NodeID(uuid),
			Priority: service.Priority(0),
		},
	}

	mgr.waiters = make(waiters)

	go mgr.setInitialNodeList() // don't wait for cluster node list else registration may be too late
	go mgr.registerWithServer() // Note: doesn't complete unless an error occurs

	return mgr
}

func (m *ServiceMgr) setInitialNodeList() {
	logging.Debugf("ServiceMgr::setInitialNodeList entry")
	defer logging.Debugf("ServiceMgr::setInitialNodeList exit")

	// wait for the node to be part of a cluster
	m.thisHost = distributed.RemoteAccess().WhoAmI()
	for distributed.RemoteAccess().Starting() && m.thisHost == "" {
		time.Sleep(time.Second)
		m.thisHost = distributed.RemoteAccess().WhoAmI()
	}
	if m.thisHost == "" {
		m.thisHost = string(m.nodeInfo.NodeID)
		// we won't get a server list so exit
		return
	}

	m.enabled = true

	// our topology is just the list of nodes in the cluster (or ourselves)
	topology := distributed.RemoteAccess().GetNodeNames()

	info := make([]rune, 0, len(topology)*32)
	nodeList := make([]queryServer, 0)
	for _, nn := range topology {
		ps := prepareOperation(nn, "ServiceMgr::setInitialNodeList")
		uuid := distributed.RemoteAccess().NodeUUID(nn)
		nodeList = append(nodeList, queryServer{nn, service.NodeInfo{service.NodeID(uuid), service.Priority(0), ps}})
		info = append(info, []rune(uuid)...)
		info = append(info, '[')
		info = append(info, []rune(nn)...)
		info = append(info, ']')
		info = append(info, ' ')
	}

	// since preparation may take a short time it is technically possible to receive new topology from the orchestrator before
	// we're able to update with our initial understanding; don't overwrite if this is the case
	set := false
	m.updateState(func(s *state) {
		if s.servers == nil {
			s.servers = nodeList
			set = true
		}
	})

	if set {
		if len(info) == 0 {
			info = append(info, []rune("no active nodes")...)
		}
		logging.Infof("Initial topology: %v", string(info))
	}
}

func (m *ServiceMgr) registerWithServer() {
	err := service.RegisterManager(m, nil)
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
				if e.nodeInfo.Opaque == nil {
					e.nodeInfo.Priority = -1
				}
				if e.nodeInfo.Priority < 0 {
					continue
				}
				res, err := distributed.RemoteAccess().ExecutePreparedAdminOp(e.nodeInfo.Opaque, "GET", "", nil,
					distributed.NO_CREDS, "")
				if res != nil && err == nil {
					var status struct {
						Code int32 `json:"code"`
					}
					jerr := json.Unmarshal(res, &status)
					if jerr == nil && errors.ErrorCode(status.Code) != errors.E_SERVICE_SHUT_DOWN {
						running++
					} else {
						e.nodeInfo.Priority = -1
					}
				} else {
					e.nodeInfo.Priority = -1
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
				IsCancelable: true, // since it is ignored anyway and ns-server still tries to cancel the task...
			}
			tasks.Tasks = append(tasks.Tasks, task)
		}
	}

	logging.Debuga(func() string { return fmt.Sprintf("ServiceMgr::GetTaskList exit: %v", tasks) })

	return tasks, nil
}

func (m *ServiceMgr) CancelTask(id string, rev service.Revision) error {
	logging.Debuga(func() string { return fmt.Sprintf("ServiceMgr::CancelTask entry %v %v", id, DecodeRev(rev)) })

	timeout := time.Duration(0)
	data := "cancel=true"
	m.mu.Lock()
	defer m.mu.Unlock()

	currentTask := fmt.Sprintf("shutdown/monitor/%s", m.changeID)
	if currentTask == id {
		if m.eject != nil {
			servers := make([]queryServer, 0, len(m.eject))
			servers = append(servers, m.servers...)
			timedOut := false
			info := make([]rune, 0, len(m.eject)*33)
			done := util.WaitCount{}
			mutex := &sync.Mutex{}
			// in parallel in case some take time to reach
			for _, e := range m.eject {
				go func() {
					if e.nodeInfo.Opaque == nil {
						// if we failed to prepare it before now, we could well be too late but try again anyway
						e.nodeInfo.Opaque = prepareOperation(e.host, "ServiceMgr::CancelTask")
					}
					_, err := distributed.RemoteAccess().ExecutePreparedAdminOp(e.nodeInfo.Opaque, "POST", data, nil,
						distributed.NO_CREDS, "")
					if err == nil {
						mutex.Lock()
						if !timedOut {
							e.nodeInfo.Priority = 0
							servers = appendInOrder(servers, e)
							info = append(info, []rune(e.nodeInfo.NodeID)...)
							info = append(info, ' ')
						}
						mutex.Unlock()
						logging.Debuga(func() string {
							return fmt.Sprintf("ServiceMgr::CancelTask cancelled shutdown down on '%s'", string(e.nodeInfo.NodeID))
						})
					} else {
						logging.Infof("ServiceMgr::CancelTask failed to cancel shutdown on '%s' (op:%v): %v",
							string(e.nodeInfo.NodeID), e.nodeInfo.Opaque, err)
					}
					done.Incr()
				}()
			}
			// wait for completion
			if !done.Until(int32(len(m.eject)), timeout) {
				logging.Errorf("ServiceMgr::CancelTask failed to cancel shutdown on all ejected nodes within the time limit.")
				mutex.Lock()
				timedOut = true
				mutex.Unlock()
			}
			if len(info) > 0 {
				logging.Infof("Topology change: shutdown cancelled on: %s", string(info))
			}
			m.updateStateLOCKED(func(s *state) {
				s.changeID = ""
				s.eject = nil
				if servers != nil {
					s.servers = servers
				}
			})
			if len(info) < len(m.servers)*64 {
				info = make([]rune, 0, len(m.servers)*64)
			} else {
				info = info[:0]
			}
			for _, n := range m.servers {
				info = append(info, []rune(n.nodeInfo.NodeID)...)
				info = append(info, '[')
				info = append(info, []rune(n.host)...)
				info = append(info, ']')
				info = append(info, ' ')
			}
			logging.Infof("Current topology: %s", string(info))
			logging.Debugf("ServiceMgr::CancelTask exit")
			return nil
		} else {
			logging.Debugf("ServiceMgr::CancelTask exit: ejected nodes list is empty.")
			return service.ErrConflict
		}
	}
	logging.Debuga(func() string { return fmt.Sprintf("ServiceMgr::CancelTask exit: unknown task (%v).", id) })
	return service.ErrNotFound
}

func appendInOrder(list []queryServer, item queryServer) []queryServer {
	for i := range list {
		if list[i].nodeInfo.NodeID > item.nodeInfo.NodeID {
			if i == 0 {
				return append([]queryServer{item}, list...)
			}
			return append(append(list[:i], item), list[i:]...)
		}
	}
	return append(list, item)
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
		if m.enabled {
			checkPrepareOperations(m.servers, "ServiceMgr::GetCurrentTopology")
		}
		for _, s := range m.servers {
			topology.Nodes = append(topology.Nodes, s.nodeInfo.NodeID)
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

func prepareOperation(host string, caller string) interface{} {
	ps, err := distributed.RemoteAccess().PrepareAdminOp(host, "shutdown", "", nil, distributed.NO_CREDS, "")
	if err != nil {
		logging.Debuga(func() string {
			return fmt.Sprintf("%v Failed to prepare admin operation for %v: %v", caller, host, err)
		})
	}
	return ps
}

func checkPrepareOperations(servers []queryServer, caller string) {
	for i := range servers {
		if servers[i].nodeInfo.Opaque == nil {
			servers[i].nodeInfo.Opaque = prepareOperation(servers[i].host, caller)
		}
	}
}

// when preparing all we're doing is updating the cached nodes list from the list of retained nodes
func (m *ServiceMgr) PrepareTopologyChange(change service.TopologyChange) error {
	logging.Debuga(func() string { return fmt.Sprintf("ServiceMgr::PrepareTopologyChange entry: %v", change) })

	if change.Type != service.TopologyChangeTypeFailover && change.Type != service.TopologyChangeTypeRebalance {
		logging.Infof("ServiceMgr::PrepareTopologyChange exit [type: %v]", change.Type)
		return service.ErrNotSupported
	}

	if !m.enabled && m.thisHost == "" {
		logging.Infof("ServiceMgr::PrepareTopologyChange waiting for initialisation...")
		// we're here ahead of initialisation; wait...
		for m.thisHost == "" {
			time.Sleep(time.Second)
		}
	}
	if !m.enabled {
		logging.Debugf("ServiceMgr::PrepareTopologyChange exit [not enabled]")
		return nil
	}

	logging.Infof("Preparing for possible topology change")

	// for each node we know about, cache its shutdown URL
	info := make([]rune, 0, len(change.KeepNodes)*64)
	servers := make([]queryServer, 0)
	m.mu.Lock()
	s := m.servers
	m.mu.Unlock()
	for _, n := range change.KeepNodes {
		var ps interface{}
		var host string
		ps = nil
		// see if we can reuse the prepared operation
		// note: this means we may take less time here but are susceptible to stale data
		for _, o := range s {
			if o.nodeInfo.NodeID == n.NodeInfo.NodeID {
				ps = o.nodeInfo.Opaque
				host = o.host
				break
			}
		}
		if ps == nil {
			host = distributed.RemoteAccess().UUIDToHost(string(n.NodeInfo.NodeID))
			ps = prepareOperation(host, "ServiceMgr::PrepareTopologyChange")
		}
		servers = append(servers, queryServer{host, service.NodeInfo{n.NodeInfo.NodeID, service.Priority(0), ps}})
		info = append(info, []rune(n.NodeInfo.NodeID)...)
		info = append(info, '[')
		info = append(info, []rune(host)...)
		info = append(info, ']')
		info = append(info, ' ')
	}

	// always keep a local list of servers that are no longer present; only the master will ever act upon this list
	var eject []queryServer
	for _, o := range s {
		found := false
		for _, n := range servers {
			if o.nodeInfo.NodeID == n.nodeInfo.NodeID {
				found = true
				break
			}
		}
		if !found {
			eject = append(eject, o)
		}
	}
	if len(eject) != 0 {
		eject = eject[0:len(eject):len(eject)]
	}

	m.updateState(func(s *state) {
		s.servers = servers
		s.eject = eject
	})

	if len(info) == 0 {
		info = append(info, []rune("no active nodes")...)
	}
	logging.Infof("Topology: %s", string(info))
	logging.Debugf("ServiceMgr::PrepareTopologyChange exit")
	return nil
}

//const _FAILOVER_LIMIT = 120 * time.Second

// This is only invoked on the master which is then responsible for initiating changes on other nodes
func (m *ServiceMgr) StartTopologyChange(change service.TopologyChange) error {
	logging.Debuga(func() string { return fmt.Sprintf("ServiceMgr::StartTopologyChange %v", change) })

	if !m.enabled {
		logging.Debuga(func() string { return "ServiceMgr::StartTopologyChange exit: not enabled" })
		return nil // do nothing, but don't fail
	}

	timeout := time.Duration(0)
	data := ""
	switch change.Type {
	case service.TopologyChangeTypeFailover:
		// if we want to implement a timeout, this is how we'd do it:
		// data = fmt.Sprintf("deadline=%v", time.Now().Add(_FAILOVER_LIMIT).Unix())
		// timeout = _FAILOVER_LIMIT
		m.updateState(func(s *state) {
			s.changeID = ""
			s.eject = nil
		})
		logging.Debugf("ServiceMgr::StartTopologyChange exit")
		return nil // we're doing nothing so just return
	case service.TopologyChangeTypeRebalance:
	default:
		logging.Debugf("ServiceMgr::StartTopologyChange exit")
		return service.ErrNotSupported
	}

	m.mu.Lock()
	if m.eject != nil {
		info := make([]rune, 0, len(m.eject)*33)
		eject := make([]queryServer, 0, len(m.eject))
		done := util.WaitCount{}
		mutex := &sync.Mutex{}
		// in parallel in case some take time to reach
		for i := range m.eject {
			go func(i int) {
				if m.eject[i].nodeInfo.Opaque == nil {
					// if we failed to prepare it before now, we could well be too late but try again anyway
					m.eject[i].nodeInfo.Opaque = prepareOperation(m.eject[i].host, "ServiceMgr::StartTopologyChange")
				}
				_, err := distributed.RemoteAccess().ExecutePreparedAdminOp(m.eject[i].nodeInfo.Opaque, "POST", data, nil,
					distributed.NO_CREDS, "")
				if err == nil {
					mutex.Lock()
					if eject != nil {
						eject = append(eject, m.eject[i])
						info = append(info, []rune(m.eject[i].nodeInfo.NodeID)...)
						info = append(info, '[')
						info = append(info, []rune(m.eject[i].host)...)
						info = append(info, ']')
						info = append(info, ' ')
					}
					mutex.Unlock()
					logging.Debuga(func() string {
						return fmt.Sprintf("ServiceMgr::StartTopologyChange initiated shutdown down on '%s'",
							string(m.eject[i].nodeInfo.NodeID))
					})
				} else {
					logging.Infof("ServiceMgr::StartTopologyChange failed to start shutdown on '%s' (op:%v): %v",
						string(m.eject[i].nodeInfo.NodeID), m.eject[i].nodeInfo.Opaque, err)
				}
				done.Incr()
			}(i)
		}
		// wait for completion
		if !done.Until(int32(len(m.eject)), timeout) {
			mutex.Lock()
			eject = nil
			mutex.Unlock()
			logging.Infof("ServiceMgr::StartTopologyChange failed to initiate shutdown on all ejected nodes within the time limit.")
		}
		if len(info) > 0 {
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

	logging.Debugf("ServiceMgr::StartTopologyChange exit")
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
