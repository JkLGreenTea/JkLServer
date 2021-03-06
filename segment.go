package jklserver

import (
	"./status"
	"go.uber.org/zap"
)

type Segment struct {
	Head	segmentHead				`json:"head"`
	Body	map[string]interface{}	`json:"body"`
}

type segmentHead struct {
	ID		uint32				`json:"id"`
	Status	status.Status		`json:"status"`
}

type segmentManager struct {
	bd							interface{}
	handleFuncSegment		  	map[uint32]*func(segment *Segment, conn *TCPConnection)
	handleFuncSegmentIsLogin  	map[uint32]*func(segment *Segment, conn *TCPConnection)
	handleFuncAuthorization		*handleFuncAuthorization

	config						*Config
	logger						*zap.Logger
	sync						*syncing
}

type handleFuncAuthorization struct {
	ID			uint32
	Handler		*func(segment *Segment, conn *TCPConnection) bool
}

func (segmentMng *segmentManager) handle(conn *TCPConnection) {
	defer segmentMng.sync.wg.Done()

	segmentMng.sync.wg.Add(1)

	if segment, err := conn.Read(); err == nil {
		if segment != nil {
			if segmentMng.handleFuncAuthorization != nil {
				if segmentMng.handleFuncAuthorization.ID == segment.Head.ID {
					func_ := *segmentMng.handleFuncAuthorization.Handler
					conn.IsLogin.Store(func_(segment, conn))
					conn.updateDeadline()

					return
				}
			}
			segmentMng.sync.rwMutex.Lock()
			if segmentHandler, ok := segmentMng.handleFuncSegment[segment.Head.ID]; ok {
				segmentMng.sync.rwMutex.Unlock()

				segmentHandlerFunc := *segmentHandler
				segment, err := conn.Read()
				if err != nil {
					segmentMng.logger.Error(err.Error())
					return
				}

				conn.updateDeadline()
				segmentHandlerFunc(segment, conn)

				return
			} else if conn.IsLogin.Load() {
				if segmentHandlerIsLogin, ok := segmentMng.handleFuncSegmentIsLogin[segment.Head.ID]; ok {
					segmentMng.sync.rwMutex.Unlock()

					segmentHandlerIsLoginFunc := *segmentHandlerIsLogin
					segment, err := conn.Read()
					if err != nil {
						segmentMng.logger.Error(err.Error())
						return
					}

					conn.updateDeadline()
					segmentHandlerIsLoginFunc(segment, conn)

					return
				}

				segmentMng.sync.rwMutex.Unlock()
			}
		}
	} else {
		segmentMng.logger.Error(err.Error())
	}
}
