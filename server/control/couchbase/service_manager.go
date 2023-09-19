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
	sync.Mutex
	state

	nodeInfo *service.NodeInfo
	waiters  waiters

	enabled bool
}

type state struct {
	rev       uint64
	changeID  string
	servers   []service.NodeID
	eject     []service.NodeID
	started   int
	completed int
}

type waiter chan state
type waiters map[waiter]bool

const _INIT_WARN_TIME = 5
const _MONITOR_POLL_INTERVAL = time.Second

func NewManager(uuid string) Manager {
	var mgr *ServiceMgr
	logging.Debugf("server::NewManager entry. UUID: %v", uuid)

	if uuid == "" {
		logging.Infof("No UUID passed.  Will not register for topology awareness.")
		logging.Debugf("server::NewManager exit: %v", mgr)
		return nil
	}

	mgr = &ServiceMgr{
		state: state{
			rev:       0,
			servers:   nil,
			eject:     nil,
			changeID:  "",
			started:   0,
			completed: 0,
		},
		nodeInfo: &service.NodeInfo{
			NodeID:   service.NodeID(uuid),
			Priority: service.Priority(0),
		},
	}

	mgr.waiters = make(waiters)

	go mgr.setInitialNodeList() // don't wait for cluster node list else registration may be too late
	go mgr.registerWithServer() // Note: doesn't complete unless an error occurs

	logging.Debugf("server::NewManager exit: %v", mgr)
	return mgr
}

func (m *ServiceMgr) setInitialNodeList() {
	if logging.Logging(logging.DEBUG) {
		logging.Debugf("ServiceMgr::setInitialNodeList entry")
		defer logging.Debugf("ServiceMgr::setInitialNodeList exit")
	}

	// wait for the node to be part of a cluster
	thisHost := distributed.RemoteAccess().WhoAmI()
	i := 0
	for distributed.RemoteAccess().Starting() && thisHost == "" {
		if i >= _INIT_WARN_TIME {
			logging.Warnf("Cluster initialisation taking longer than expected.")
			i = 0
		}
		time.Sleep(time.Second)
		thisHost = distributed.RemoteAccess().WhoAmI()
		i++
	}
	if thisHost == "" {
		// we won't get a server list so exit
		logging.Debugf("ServiceMgr::setInitialNodeList exit")
		return
	}

	// our topology is just the list of nodes in the cluster (or ourselves)
	topology := distributed.RemoteAccess().GetNodeNames()

	nodeList := make([]service.NodeID, 0)
	for _, nn := range topology {
		uuid := distributed.RemoteAccess().NodeUUID(nn)
		i := 0
		for uuid == "" {
			if i%10 == 0 {
				logging.Warnf("Unable to resolve node ID for [%v]. Retrying.", nn)
			}
			time.Sleep(time.Second)
			if m.state.servers != nil {
				// server list was updated in PrepareTopology change so abort this operation
				logging.Debugf("ServiceMgr::setInitialNodeList exit - topology already set")
				return
			}
			uuid = distributed.RemoteAccess().NodeUUID(nn)
			if uuid != "" {
				logging.Infof("Resolved node ID '%v' for [%v]", uuid, nn)
			}
			i++
		}
		nodeList = append(nodeList, service.NodeID(uuid))
	}

	m.Lock()
	set := false
	m.updateStateLOCKED(func(s *state) {
		if s.servers == nil {
			s.servers = nodeList
			set = true
		}
	})
	m.enabled = true
	m.Unlock()

	if set {
		if len(nodeList) == 0 {
			logging.Infof("Initial topology: no active nodes")
		} else {
			for i, n := range nodeList {
				logging.Infof("Initial topology %d/%d: %s ", i+1, len(nodeList), n)
			}
		}
	} else {
		logging.Infof("Topology already set.")
	}
	logging.Debugf("ServiceMgr::setInitialNodeList exit")
}

func (m *ServiceMgr) registerWithServer() {
	err := service.RegisterManager(m, nil)
	if err != nil {
		logging.Errorf("ServiceMgr::registerWithServer error %v", err)
		m.Shutdown()
	}
}

func (m *ServiceMgr) GetNodeInfo() (*service.NodeInfo, error) {
	logging.Debugf("ServiceMgr::GetNodeInfo: %v", m.nodeInfo)
	return m.nodeInfo, nil
}

func (m *ServiceMgr) Shutdown() error {
	logging.Severef("ServiceMgr::Shutdown")
	os.Exit(0)
	return nil
}

// There are only active tasks on the master node and only whilst others are being gracefully stopped.

func (m *ServiceMgr) GetTaskList(rev service.Revision, cancel service.Cancel) (*service.TaskList, error) {
	logging.Debuga(func() string { return fmt.Sprintf("ServiceMgr::GetTaskList entry: %v", DecodeRev(rev)) })

	curState, err := m.wait(rev, cancel)
	if err != nil {
		logging.Debugf("ServiceMgr::GetTaskList exit: error: %v", err)
		return nil, err
	}

	tasks := &service.TaskList{}
	tasks.Rev = EncodeRev(curState.rev)
	tasks.Tasks = make([]service.Task, 0)

	if curState.changeID != "" { // master
		if curState.completed >= curState.started {
			m.updateState(func(s *state) {
				s.changeID = ""
				s.eject = nil
				s.started = 0
				s.completed = 0
			})
		} else {
			task := service.Task{
				Rev:          EncodeRev(0),
				ID:           fmt.Sprintf("shutdown/monitor/%s", curState.changeID),
				Type:         service.TaskTypeRebalance,
				Status:       service.TaskStatusRunning,
				Progress:     float64(curState.completed) / float64(curState.started),
				IsCancelable: true, // since it is ignored anyway and ns-server still tries to cancel the task...
			}
			tasks.Tasks = append(tasks.Tasks, task)
		}
	}

	logging.Debugf("ServiceMgr::GetTaskList exit: %v", tasks)

	return tasks, nil
}

// Here we rely on the /admin/shutdown REST interface to obtain information on the progress of the remote graceful shutdown
// Using this saves us from having to establish and handle another communication mechanism to feed back state to the master

func (m *ServiceMgr) monitorShutdown(changeID string, host string, eject service.NodeID) {
	logging.Debuga(func() string {
		return fmt.Sprintf("ServiceMgr::monitorShutdown entry: %v(%v[%v])", changeID, eject, host)
	})
	for {
		m.Lock()
		cancelled := m.state.changeID != changeID
		m.Unlock()
		if cancelled {
			logging.Debugf("ServiceMgr::monitorShutdown exit: %v(%v[%v]) - cancelled", changeID, eject, host)
			return
		}
		var err errors.Error
		var code errors.ErrorCode
		logging.Debugf("ServiceMgr::monitorShutdown: %v(%v[%v]) - polling", changeID, eject, host)
		distributed.RemoteAccess().GetRemoteDoc(host, "", "shutdown", "GET", func(doc map[string]interface{}) {
			if i, ok := doc["code"]; ok {
				if c, ok := i.(float64); ok {
					code = errors.ErrorCode(c)
				}
			}
			logging.Debugf("ServiceMgr::monitorShutdown: %v(%v[%v]) - document: %v", changeID, eject, host, doc)
		}, func(e errors.Error) {
			err = e
		}, distributed.NO_CREDS, "", nil)
		if err == nil {
			if code == errors.E_SERVICE_SHUT_DOWN {
				break
			}
		} else {
			logging.Errorf("ServiceMgr::monitorShutdown: %v(%v[%v]) - error: %v", changeID, eject, host, err)
			break
		}
		m.Lock()
		cancelled = m.state.changeID != changeID
		m.Unlock()
		if cancelled {
			logging.Debugf("ServiceMgr::monitorShutdown exit: %v(%v[%v]) - cancelled", changeID, eject, host)
			return
		}
		time.Sleep(_MONITOR_POLL_INTERVAL)
	}
	logging.Infof("Topology change %v: %v shut down", changeID, eject)
	logging.Debugf("ServiceMgr::monitorShutdown exit: %v(%v[%v])", changeID, eject, host)
}

func (m *ServiceMgr) CancelTask(id string, rev service.Revision) error {
	logging.Debuga(func() string { return fmt.Sprintf("ServiceMgr::CancelTask entry %v %v", id, DecodeRev(rev)) })

	if !m.enabled {
		logging.Debugf("ServiceMgr::CancelTask exit: not enabled")
		return nil // do nothing, but don't fail
	}

	timeout := time.Duration(0)
	data := "cancel=true"
	m.Lock()

	currentTask := fmt.Sprintf("shutdown/monitor/%s", m.changeID)
	if currentTask == id {
		if m.eject != nil {
			servers := make([]service.NodeID, 0, len(m.eject))
			servers = append(servers, m.servers...)
			timedOut := false
			info := make([]rune, 0, len(m.eject)*33)
			done := util.WaitCount{}
			mutex := &sync.Mutex{}
			// in parallel in case some take time to reach
			for _, e := range m.eject {
				go func() {
					host := distributed.RemoteAccess().UUIDToHost(string(e))
					if host == "" {
						logging.Infof("ServiceMgr::CancelTask failed to cancel shutdown on %s: unable to resolve host", string(e))
					} else {
						var err errors.Error
						distributed.RemoteAccess().DoRemoteOps([]string{host}, "shutdown", "POST", "", data, func(e errors.Error) {
							err = e
						}, distributed.NO_CREDS, "")
						if err == nil {
							mutex.Lock()
							if !timedOut {
								servers = appendInOrder(servers, e)
								info = append(info, []rune(e)...)
								info = append(info, ' ')
							}
							mutex.Unlock()
							logging.Debugf("ServiceMgr::CancelTask cancelled shutdown down on %s[%s]", string(e), host)
						} else {
							logging.Infof("ServiceMgr::CancelTask failed to cancel shutdown on %s[%s]: %v", string(e), host, err)
						}
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
				logging.Infof("Topology change %v: shutdown cancelled on: %s", m.changeID, string(info))
			}
			m.updateStateLOCKED(func(s *state) {
				s.changeID = ""
				s.eject = nil
				if servers != nil {
					s.servers = servers
				}
			})
			for i, n := range m.servers {
				logging.Infof("Current topology %d/%d: %s ", i+1, len(m.servers), n)
			}
			m.Unlock()
			logging.Debugf("ServiceMgr::CancelTask exit")
			return nil
		} else {
			m.Unlock()
			logging.Debugf("ServiceMgr::CancelTask exit: ejected nodes list is empty.")
			return service.ErrConflict
		}
	}
	m.Unlock()
	logging.Debugf("ServiceMgr::CancelTask exit: unknown task (%v).", id)
	return service.ErrNotFound
}

func appendInOrder(list []service.NodeID, item service.NodeID) []service.NodeID {
	for i := range list {
		if list[i] > item {
			if i == 0 {
				return append([]service.NodeID{item}, list...)
			}
			return append(append(list[:i], item), list[i:]...)
		}
	}
	return append(list, item)
}

// return the current node list as understood by this process
func (m *ServiceMgr) GetCurrentTopology(rev service.Revision, cancel service.Cancel) (*service.Topology, error) {
	logging.Tracea(func() string { return fmt.Sprintf("ServiceMgr::GetCurrentTopology entry: rev = %v", DecodeRev(rev)) })

	state, err := m.wait(rev, cancel)
	if err != nil {
		logging.Tracef("ServiceMgr::GetCurrentTopology exit: %v", err)
		return nil, err
	}

	topology := &service.Topology{}

	topology.Rev = EncodeRev(state.rev)
	m.Lock()
	if m.servers != nil && len(m.servers) != 0 {
		for _, s := range m.servers {
			topology.Nodes = append(topology.Nodes, s)
		}
	} else {
		topology.Nodes = append(topology.Nodes, m.nodeInfo.NodeID)
	}
	m.Unlock()
	topology.IsBalanced = true
	topology.Messages = nil

	logging.Tracea(func() string {
		return fmt.Sprintf("ServiceMgr::GetCurrentTopology exit: %v - %v eject: %v", DecodeRev(rev), topology, m.eject)
	})

	return topology, nil
}

// when preparing all we're doing is updating the cached nodes list from the list of retained nodes
func (m *ServiceMgr) PrepareTopologyChange(change service.TopologyChange) error {
	logging.Debugf("ServiceMgr::PrepareTopologyChange entry: %v", change)

	if change.Type != service.TopologyChangeTypeFailover && change.Type != service.TopologyChangeTypeRebalance {
		logging.Infof("ServiceMgr::PrepareTopologyChange exit [type: %v]", change.Type)
		return service.ErrNotSupported
	}

	if m.servers != nil {
		logging.Infof("Preparing for possible topology change: %v (%v)", change.ID, change.Type)
	} else {
		logging.Infof("Preparing topology from: %v (%v)", change.ID, change.Type)
	}

	servers := make([]service.NodeID, 0, len(change.KeepNodes))
	m.Lock()
	s := m.servers
	m.Unlock()
	for _, n := range change.KeepNodes {
		servers = append(servers, n.NodeInfo.NodeID)
	}

	// always keep a local list of servers that are no longer present; only the master will ever act upon this list
	var eject []service.NodeID
	for _, o := range s {
		found := false
		for _, n := range servers {
			if o == n {
				found = true
				break
			}
		}
		if !found {
			eject = append(eject, o)
		}
	}
	// in the unlikely event a node is in the change's eject list but wasn't previously known, add it to the manager's eject list
	for _, e := range change.EjectNodes {
		found := false
		for i := range eject {
			if eject[i] == e.NodeID {
				found = true
				break
			}
		}
		if !found {
			eject = append(eject, e.NodeID)
		}
	}
	if len(eject) != 0 {
		eject = eject[0:len(eject):len(eject)]
		if logging.LogLevel() == logging.DEBUG {
			for i, n := range eject {
				logging.Debugf("Ejection candidate: %d/%d: %s", i+1, len(eject), n)
			}
		}
	}

	m.Lock()
	m.enabled = true
	m.updateStateLOCKED(func(s *state) {
		s.servers = servers
		s.eject = eject
	})
	m.Unlock()

	if len(servers) == 0 {
		logging.Infof("Topology: no active nodes")
	} else {
		for i, n := range servers {
			logging.Infof("Topology %d/%d: %s", i+1, len(m.servers), n)
		}
	}
	logging.Debugf("ServiceMgr::PrepareTopologyChange exit")
	return nil
}

//const _FAILOVER_LIMIT = 120 * time.Second

// This is only invoked on the master which is then responsible for initiating changes on other nodes
func (m *ServiceMgr) StartTopologyChange(change service.TopologyChange) error {
	logging.Infof("Topology change %v: (%v) initiated.", change.ID, change.Type)

	if !m.enabled {
		logging.Debugf("ServiceMgr::StartTopologyChange exit: not enabled")
		return nil // do nothing, but don't fail
	}

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

	m.Lock()
	if m.eject != nil {
		// in parallel in case some take time to reach
		for i := range m.eject {
			m.updateStateLOCKED(func(s *state) {
				s.started++
			})
			go func() {
				host := distributed.RemoteAccess().UUIDToHost(string(m.eject[i]))
				if host == "" {
					logging.Infof("Topology change %v: failed to start shutdown on %s: unable to resolve host", change.ID,
						string(m.eject[i]))
				} else {
					var err errors.Error
					distributed.RemoteAccess().DoRemoteOps([]string{host}, "shutdown", "POST", "", data, func(e errors.Error) {
						err = e
					}, distributed.NO_CREDS, "")
					if err == nil {
						logging.Debugf("ServiceMgr::StartTopologyChange initiated shutdown down on %s[%s]",
							string(m.eject[i]), host)

						// monitor for completion
						m.monitorShutdown(change.ID, host, m.eject[i])
					} else {
						logging.Infof("Topology change %v: failed to start shutdown on %s[%v]: %v", change.ID,
							string(m.eject[i]), host, err)
					}
				}
				m.updateState(func(s *state) {
					s.completed++
					if s.completed >= s.started {
						s.eject = nil
						s.changeID = ""
					}
				})
			}()
			logging.Infof("Topology change %v: %s shutdown initiated", change.ID, m.eject[i])
		}
		m.updateStateLOCKED(func(s *state) {
			s.changeID = change.ID
		})
	} else {
		logging.Infof("Topology change %v: no action necessary.", change.ID)
	}
	m.Unlock()

	logging.Debugf("ServiceMgr::StartTopologyChange exit (%v)", change.ID)
	return nil
}

func EncodeRev(rev uint64) service.Revision {
	ext := make(service.Revision, 8)
	binary.BigEndian.PutUint64(ext, rev)
	return ext
}

func DecodeRev(ext service.Revision) uint64 {
	if ext != nil {
		return binary.BigEndian.Uint64(ext)
	}
	return 0
}

func (m *ServiceMgr) updateState(body func(state *state)) {
	m.Lock()
	m.updateStateLOCKED(body)
	m.Unlock()
}

func (m *ServiceMgr) updateStateLOCKED(body func(state *state)) {
	body(&m.state)
	m.state.rev++

	// notify waiters
	s := m.state
	for ch := range m.waiters {
		if ch != nil {
			ch <- s
		}
	}
	m.waiters = make(waiters)
}

func (m *ServiceMgr) wait(rev service.Revision, cancel service.Cancel) (state, error) {

	m.Lock()

	currState := m.state

	if rev == nil {
		m.Unlock()
		return currState, nil
	}

	haveRev := DecodeRev(rev)
	if haveRev != m.rev {
		m.Unlock()
		return currState, nil
	}

	ch := make(waiter, 1)
	m.waiters[ch] = true
	m.Unlock()

	select {
	case <-cancel:
		m.Lock()
		delete(m.waiters, ch)
		m.Unlock()
		return state{}, service.ErrCanceled
	case newState := <-ch:
		return newState, nil
	}
}
