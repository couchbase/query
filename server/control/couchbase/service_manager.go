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

	"github.com/couchbase/cbauth/service"
	"github.com/couchbase/query/clustering"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/server"
)

type Manager interface {
}

type ServiceMgr struct {
	mu *sync.RWMutex
	state

	nodeInfo *service.NodeInfo
	waiters  waiters

	managedServer *server.Server
	cluster       clustering.Cluster
}

type state struct {
	rev      uint64
	servers  []service.NodeID
	changeID string
	eject    []string
}

type waiter chan state
type waiters map[waiter]struct{}

func NewManager(server *server.Server) (Manager, error) {
	var mgr *ServiceMgr
	logging.Debugf("server::NewManager entry")
	defer logging.Debuga(func() string { return fmt.Sprintf("server::NewManager exit: %v", mgr) })

	if server == nil {
		return nil, fmt.Errorf("Invalid server.")
	}

	c, err := server.ConfigurationStore().Cluster()
	if err != nil || c == nil {
		return nil, fmt.Errorf("Unable to access cluster information.")
	}

	mu := &sync.RWMutex{}

	mgr = &ServiceMgr{
		mu: mu,
		state: state{
			rev:      0,
			servers:  nil,
			eject:    nil,
			changeID: "",
		},
		managedServer: server,
		cluster:       c,
	}

	mgr.waiters = make(waiters)

	mgr.nodeInfo = &service.NodeInfo{
		NodeID:   service.NodeID(distributed.RemoteAccess().WhoAmI()),
		Priority: service.Priority(0),
	}

	mgr.setInitialNodeList()
	go mgr.registerWithServer() // Note: doesn't complete unless an error occurs

	return mgr, nil
}

func (m *ServiceMgr) setInitialNodeList() {
	logging.Debugf("ServiceMgr::setInitialNodeList entry")
	defer logging.Debugf("ServiceMgr::setInitialNodeList exit")

	// our topology is just the list of nodes in the cluster (or ourselves)
	topology := distributed.RemoteAccess().GetNodeNames()

	nodeList := make([]service.NodeID, 0)
	for _, nn := range topology {
		nodeList = append(nodeList, service.NodeID(nn))
	}

	m.updateState(func(s *state) {
		s.servers = nodeList
	})
	logging.Debuga(func() string { return fmt.Sprintf("ServiceMgr::setInitialNodeList list: %v", nodeList) })
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

var ssd = errors.NewServiceShutDownError() // specific shutdown completed error code

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
	eject := m.eject
	m.mu.Unlock()
	if eject != nil { // we are the master
		running := 0
		res, errs := distributed.RemoteAccess().DoAdminOps(eject, "shutdown", "GET", "", "", nil, distributed.NO_CREDS, "")
		for i, r := range res {
			remove := true
			if r != nil && errs[i] == nil {
				var status struct {
					Code int32 `json:"code"`
				}
				err = json.Unmarshal(r, &status)
				if err == nil && status.Code != ssd.Code() {
					running++
					remove = false
				}
			}
			if remove {
				eject[i] = ""
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
		topology.Nodes = append([]service.NodeID(nil), m.servers...)
	} else {
		topology.Nodes = append([]service.NodeID(nil), m.nodeInfo.NodeID)
	}
	m.mu.Unlock()
	topology.IsBalanced = true
	topology.Messages = nil

	logging.Debuga(func() string {
		return fmt.Sprintf("ServiceMgr::GetCurrentTopology exit: %v - %v eject: %v",
			DecodeRev(rev), topology, m.eject)
	})

	return topology, nil
}

// when preparing all we're doing is updating the cached nodes list from the list of retained nodes
func (m *ServiceMgr) PrepareTopologyChange(change service.TopologyChange) error {
	logging.Debuga(func() string { return fmt.Sprintf("ServiceMgr::PrepareTopologyChange entry: %v", change) })
	defer logging.Debugf("ServiceMgr::PrepareTopologyChange exit")

	if change.Type != service.TopologyChangeTypeFailover && change.Type != service.TopologyChangeTypeRebalance {
		return service.ErrNotSupported
	}

	servers := make([]service.NodeID, 0)
	for _, n := range change.KeepNodes {
		servers = append(servers, n.NodeInfo.NodeID)
	}

	m.updateState(func(s *state) {
		s.servers = servers
	})

	logging.Infof("Topology change: %v ", servers)
	return nil
}

// This is only invoked on the master which is then responsible for initiating changes on other nodes
func (m *ServiceMgr) StartTopologyChange(change service.TopologyChange) error {
	logging.Debuga(func() string { return fmt.Sprintf("ServiceMgr::StartTopologyChange %v", change) })
	defer logging.Debugf("ServiceMgr::StartTopologyChange exit")

	if change.Type != service.TopologyChangeTypeFailover &&
		change.Type != service.TopologyChangeTypeRebalance {
		return service.ErrNotSupported
	}

	if len(change.EjectNodes) == 0 {
		logging.Debugf("ServiceMgr::StartTopologyChange no nodes to eject")
		return nil
	}

	m.mu.Lock()

	candidates := make([]string, 0, len(change.EjectNodes))
	for _, n := range change.EjectNodes {
		candidates = append(candidates, string(n.NodeID))
	}

	_, errs := distributed.RemoteAccess().DoAdminOps(candidates, "shutdown", "POST", "", "", nil, distributed.NO_CREDS, "")
	eject := make([]string, 0, len(errs)) // errs will be the same length as candidates
	for i, _ := range errs {
		if errs[i] == nil {
			eject = append(eject, candidates[i])
			logging.Debuga(func() string {
				return fmt.Sprintf(
					"ServiceMgr::StartTopologyChange initiated shutdown down on '%s'", candidates[i])
			})
		} else {
			logging.Infof("ServiceMgr::StartTopologyChange failed start shudown on '%s': %v", candidates[i], errs[i])
		}
	}
	eject = eject[:len(eject):len(eject)]
	logging.Infof("Topology change: shutdown initiated on: %v", eject)
	m.updateStateLOCKED(func(s *state) {
		if len(eject) > 0 {
			s.changeID = change.ID
			s.eject = eject
		} else {
			s.changeID = ""
			s.eject = nil
		}
	})
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
