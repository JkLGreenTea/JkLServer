package jklserver

import (
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"strconv"
)

type threadsManager struct {
	tcpManagers			map[uint16]*TCPManager
	udpManagers			map[uint16]*UDPManager
	segmentManager		*segmentManager
	packageManager		*packageManager
	sessions			map[int64]*session

	serverCondition		*condition
	statistic			*statistic

	config				*Config
	logger				*zap.Logger
	sync				*syncing
}

func (threadsMng *threadsManager) NewTCPManager(port, maxNumberOfConnections uint16) {
	threadsMng.sync.rwMutex.Lock()
	if _, ok := threadsMng.tcpManagers[port]; !ok {
		threadsMng.sync.rwMutex.Unlock()

		tcpMng := &TCPManager{
			Port: port,
			MaxNumberOfConnections: maxNumberOfConnections,
			Connections: make(map[int64]*TCPConnection),
			segmentManager: threadsMng.segmentManager,
			sessions: threadsMng.sessions,

			serverCondition: threadsMng.serverCondition,
			statistic: threadsMng.statistic,

			config: threadsMng.config,
			logger: threadsMng.logger,
			sync: threadsMng.sync,
		}

		tcpMng.Condition = &condition{
			IsStarted: atomic.NewBool(false),
			IsRestarting: atomic.NewBool(false),
			IsAllowConn: atomic.NewBool(false),
			IsShutdownTimer: atomic.NewBool(false),
		}

		threadsMng.sync.mutex.Lock()
		threadsMng.tcpManagers[port] = tcpMng
		threadsMng.sync.mutex.Unlock()

		threadsMng.logger.Info("|" + threadsMng.config.IP + ":" + strconv.Itoa(int(port)) + "|: TCP manager created.")
	} else {
		threadsMng.sync.rwMutex.Unlock()

		threadsMng.logger.Error("|" + threadsMng.config.IP + ":" + strconv.Itoa(int(port)) + "|: failed to create TCP manager.")
	}
}

func (threadsMng *threadsManager) NewUDPManager(port uint16) {
	threadsMng.sync.rwMutex.Lock()
	if _, ok := threadsMng.udpManagers[port]; !ok {
		threadsMng.sync.rwMutex.Unlock()
		udpMng := &UDPManager{
			Port: port,
			packageManager: threadsMng.packageManager,
			sessions: threadsMng.sessions,

			serverCondition: threadsMng.serverCondition,
			statistic: threadsMng.statistic,

			config: threadsMng.config,
			logger: threadsMng.logger,
			sync: threadsMng.sync,
		}

		udpMng.Condition = &condition{
			IsStarted: atomic.NewBool(false),
			IsRestarting: atomic.NewBool(false),
			IsAllowConn: atomic.NewBool(false),
			IsShutdownTimer: atomic.NewBool(false),
		}

		threadsMng.sync.mutex.Lock()
		threadsMng.udpManagers[port] = udpMng
		threadsMng.sync.mutex.Unlock()

		threadsMng.logger.Info("|" + threadsMng.config.IP + ":" + strconv.Itoa(int(port)) + "|: UDP manager created.")
	} else {
		threadsMng.sync.rwMutex.Unlock()

		threadsMng.logger.Error("|" + threadsMng.config.IP + ":" + strconv.Itoa(int(port)) + "|: failed to create UDP manager.")
	}
}

func (threadsMng *threadsManager) AddHandleFuncSegment(id uint32, funcHandler func(segment *Segment, conn *TCPConnection), isLogin bool) {
	if isLogin {
		threadsMng.sync.rwMutex.Lock()
		if threadsMng.segmentManager != nil {
			threadsMng.segmentManager.sync.mutex.Lock()
			threadsMng.segmentManager.handleFuncSegmentIsLogin[id] = &funcHandler
			threadsMng.segmentManager.sync.mutex.Unlock()
		}
		threadsMng.sync.rwMutex.Unlock()
	} else {
		threadsMng.sync.rwMutex.Lock()
		if threadsMng.segmentManager != nil {
			threadsMng.segmentManager.sync.mutex.Lock()
			threadsMng.segmentManager.handleFuncSegment[id] = &funcHandler
			threadsMng.segmentManager.sync.mutex.Unlock()
		}
		threadsMng.sync.rwMutex.Unlock()
	}
}

func (threadsMng *threadsManager) SetHandleFuncAuthorization(id uint32, funcHandler func(segment *Segment, conn *TCPConnection) bool) {
	threadsMng.sync.rwMutex.Lock()
	if threadsMng.segmentManager != nil {
		threadsMng.segmentManager.sync.mutex.Lock()
		threadsMng.segmentManager.handleFuncAuthorization = &handleFuncAuthorization{
			ID:      id,
			Handler: &funcHandler,
		}
		threadsMng.segmentManager.sync.mutex.Unlock()
	}
	threadsMng.sync.rwMutex.Unlock()
}

func (threadsMng *threadsManager) AddHandleFuncPackage(id uint32, funcHandler func(pack *Package, conn *UDPConnection)) {
	threadsMng.sync.rwMutex.Lock()
	if threadsMng.packageManager != nil {
		threadsMng.packageManager.sync.mutex.Lock()
		threadsMng.packageManager.handleFuncPackage[id] = &funcHandler
		threadsMng.packageManager.sync.mutex.Unlock()
	}
	threadsMng.sync.rwMutex.Unlock()
}

func (threadsMng *threadsManager) SetBD(bd interface{}) {
	threadsMng.segmentManager.bd = bd
	threadsMng.packageManager.bd = bd
}