package jklserver

import (
	"./tools"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"strconv"
	"sync"
	"time"
)

type server struct {
	Config				*Config
	ThreadsManager		*threadsManager
	Condition			*condition
	Statistic			*statistic

	commands			map[int64]map[int64]*command

	logger				*zap.Logger
	sync				*syncing
}

type syncing struct {
	wg					*tools.Wg
	mutex				*sync.Mutex
	rwMutex				*sync.RWMutex
}

type statistic struct {
	NumberReceivedSegments 	*atomic.Uint32
	NumberSentSegments     	*atomic.Uint32
	NumberReceivedPackages	*atomic.Uint32
	NumberSentPackages    	*atomic.Uint32
	NumberSystemGoroutines	*atomic.Uint32
	NumberGoroutines		*atomic.Uint32
	NumberConnections		*atomic.Uint32
}

type condition struct {
	IsStarted			*atomic.Bool
	IsRestarting		*atomic.Bool
	IsAllowConn			*atomic.Bool
	IsShutdownTimer		*atomic.Bool
}

func newServer() *server {
	return &server{}
}

func (srv *server) build() {
	wg := &tools.Wg {
		Wg: &sync.WaitGroup{},
		Count: atomic.NewUint32(0),
	}
	mutex := &sync.Mutex{}
	rwMutex := &sync.RWMutex{}

	srv.sync = &syncing {
		wg: wg,
		mutex: mutex,
		rwMutex: rwMutex,
	}

	srv.Condition = &condition {
		IsStarted: atomic.NewBool(false),
		IsRestarting: atomic.NewBool(false),
		IsAllowConn: atomic.NewBool(false),
		IsShutdownTimer: atomic.NewBool(false),
	}

	srv.Statistic = &statistic {
		NumberReceivedSegments: atomic.NewUint32(0),
		NumberSentSegments: atomic.NewUint32(0),
		NumberReceivedPackages: atomic.NewUint32(0),
		NumberSentPackages: atomic.NewUint32(0),
		NumberSystemGoroutines: atomic.NewUint32(0),
		NumberConnections: atomic.NewUint32(0),
		NumberGoroutines: wg.Count,
	}

	srv.ThreadsManager = &threadsManager{
		tcpManagers: make(map[uint16]*TCPManager),
		udpManagers: make(map[uint16]*UDPManager),
		sessions: make(map[int64]*session),

		serverCondition: srv.Condition,
		statistic: srv.Statistic,

		config: srv.Config,
		logger: srv.logger,
		sync: srv.sync,
	}

	srv.ThreadsManager.segmentManager = &segmentManager{
		handleFuncSegment: make(map[uint32]*func(segment *Segment, conn *TCPConnection)),
		handleFuncSegmentIsLogin: make(map[uint32]*func(segment *Segment, conn *TCPConnection)),

		config: srv.Config,
		logger: srv.logger,
		sync: srv.sync,
	}

	srv.ThreadsManager.packageManager = &packageManager{
		handleFuncPackage:	make(map[uint32]*func(pack *Package, conn *UDPConnection)),

		config: srv.Config,
		logger: srv.logger,
		sync: srv.sync,
	}
}

func (srv *server) listen() {
	defer srv.Statistic.NumberSystemGoroutines.Dec()

	srv.Statistic.NumberSystemGoroutines.Inc()

	if !srv.Condition.IsStarted.Load() && !srv.Condition.IsShutdownTimer.Load() && !srv.Condition.IsRestarting.Load() {
		srv.logger.Info("* Server starting...")

		srv.Condition.IsAllowConn.Store(true)
		srv.Condition.IsStarted.Store(true)

		for _, mng := range srv.ThreadsManager.tcpManagers {
			go mng.ListenTCP()
		}

		for _, mng := range srv.ThreadsManager.udpManagers {
			go mng.ListenUDP()
		}

		time.Sleep(1 * time.Second)

		srv.sync.wg.Wait()
	} else {
		srv.logger.Error("Unable to run on the server.")
	}
}

func (srv *server) Listen() {
	go srv.listen()

	srv.createDefaultCmd()
	srv.readingCmd()
}

func (srv *server) Stop() {
	if srv.Condition.IsStarted.Load() && !srv.Condition.IsShutdownTimer.Load() {
		srv.Condition.IsShutdownTimer.Store(true)

		if int(srv.Config.TimeShutdown.Seconds()) > 60 {
			srv.logger.Info("* The server will be shutdown after " + strconv.Itoa(int(srv.Config.TimeShutdown.Seconds())) + " seconds.")
			time.Sleep(srv.Config.TimeShutdown - 60)
		}

		srv.logger.Info("* The server will be shutdown after 60 seconds.")
		time.Sleep(30 * time.Second)

		srv.logger.Info("* The server will be shutdown after 30 seconds.")
		time.Sleep(20 * time.Second)

		srv.Condition.IsStarted.Store(false)

		for i := 10; i > 0; i-- {
			srv.logger.Info("* The server will be shutdown after " + strconv.Itoa(i) + " seconds.")
			time.Sleep(1 * time.Second)
		}

		srv.logger.Info("* Server stopped.")

		srv.Condition.IsShutdownTimer.Store(false)
	} else {
		srv.logger.Error("Unable to stop on the server.")
	}

	err := srv.logger.Sync()
	if err != nil {
		srv.logger.Error(err.Error())
	}
}

func (srv *server) Restarting() {
	if !srv.Condition.IsShutdownTimer.Load() && srv.Condition.IsStarted.Load() && !srv.Condition.IsRestarting.Load() {
		srv.Condition.IsRestarting.Store(true)
		srv.logger.Info("* Server restarting...")

		srv.Stop()
		srv.Condition.IsRestarting.Store(false)
		srv.listen()
	} else {
		srv.logger.Error("Unable to reboot on the server.")
	}
}

func (srv *server) DisableConnection() {
	if srv.Condition.IsAllowConn.Load() {
		srv.Condition.IsAllowConn.Store(false)
	} else {
		srv.logger.Error("Connection to the server is already prohibited.")
	}
}

func (srv *server) AllowConnection() {
	if !srv.Condition.IsAllowConn.Load() {
		srv.Condition.IsAllowConn.Store(true)
	} else {
		srv.logger.Error("Connection to the server is already allowed.")
	}
}