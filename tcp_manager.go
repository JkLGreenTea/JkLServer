package jklserver

import (
	"bufio"
	"encoding/hex"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"math/big"
	"net"
	"strconv"
	"strings"
	"time"
)

type TCPManager struct {
	tcpListener					*net.TCPListener
	Port						uint16
	MaxNumberOfConnections		uint16
	Connections					map[int64]*TCPConnection
	Condition					*condition

	segmentManager				*segmentManager
	sessions					map[int64]*session
	serverCondition				*condition
	statistic					*statistic

	config						*Config
	logger						*zap.Logger
	sync						*syncing
}

func (tcpMng *TCPManager) ListenTCP() {
	defer func() {
		tcpMng.sync.wg.Done()

		if tcpMng.tcpListener != nil {
			err := tcpMng.tcpListener.Close()
			if err != nil {
				tcpMng.logger.Error(err.Error())
			}
		}

		tcpMng.logger.Info("|" + tcpMng.config.IP + ":" + strconv.Itoa(int(tcpMng.Port)) + "|: TCP listener is closed.")
	}()

	tcpMng.sync.wg.Add(1)

	tcpMng.Condition.IsStarted.Store(true)
	tcpMng.Condition.IsAllowConn.Store(true)

	addr := net.TCPAddr{
		Port: int(tcpMng.Port),
		IP:   net.ParseIP(tcpMng.config.IP),
	}

	tcpListener, err := net.ListenTCP("tcp", &addr)
	if err != nil {
		tcpMng.logger.Error(err.Error())

		return
	}

	tcpMng.tcpListener = tcpListener

	tcpMng.logger.Info("|" + tcpMng.config.IP + ":" + strconv.Itoa(int(tcpMng.Port)) + "|: TCP listener started.")

	tcpConnChan := make(chan net.Conn)

	go func(tcpConnChan chan net.Conn) {
		defer close(tcpConnChan)

		for {
			if !tcpMng.Condition.IsStarted.Load() || !tcpMng.serverCondition.IsStarted.Load() {
				return
			}

			tcpConn, err := tcpMng.tcpListener.Accept()
			if err != nil {
				tcpMng.logger.Error(err.Error())

				continue
			}

			if tcpMng.serverCondition.IsAllowConn.Load() {
				if tcpMng.Condition.IsAllowConn.Load() {
					select {
					case tcpConnChan <- tcpConn:

					}
				} else {
					tcpMng.logger.Error("|" + tcpMng.config.IP + ":" + strconv.Itoa(int(tcpMng.Port)) + "| " + tcpConn.RemoteAddr().String() + " - attempt to connect to a closed TCP manager.")

					continue
				}
			} else {
				tcpMng.logger.Error("|" + tcpMng.config.IP + ":" + strconv.Itoa(int(tcpMng.Port)) + "| " + tcpConn.RemoteAddr().String() + " - attempt to connect to a closed server.")

				continue
			}
		}
	}(tcpConnChan)

	for {
		if !tcpMng.Condition.IsStarted.Load() || !tcpMng.serverCondition.IsStarted.Load() {
			return
		}

		select {
		case tcpConn := <-tcpConnChan:
			if tcpMng.serverCondition.IsAllowConn.Load() {
				if tcpMng.Condition.IsAllowConn.Load() {
					if tcpMng.MaxNumberOfConnections > uint16(len(tcpMng.Connections)) {
						stringIP := strings.Split(tcpConn.RemoteAddr().String(), ":")[0]
						i := new(big.Int)
						i.SetString(hex.EncodeToString([]byte(stringIP)), 16)
						stringIPHex := i.Int64()

						if _, ok := tcpMng.Connections[stringIPHex]; ok {
							tcpMng.logger.Error("|" + tcpMng.config.IP + ":" + strconv.Itoa(int(tcpMng.Port)) + "|: " + tcpConn.RemoteAddr().String() + " - IP magically already exists in the list of connections")

							continue
						}

						session_ :=  newSession()

						tcpMng.sync.mutex.Lock()
						tcpMng.sessions[stringIPHex] = session_
						tcpMng.sync.mutex.Unlock()

						conn := &TCPConnection{
							Session: session_,
							BD: tcpMng.segmentManager.bd,
							tcpConn: tcpConn,

							IsDisconnect: atomic.NewBool(false),
							IsLogin: atomic.NewBool(false),

							tcpManagerCondition: tcpMng.Condition,
							serverCondition: tcpMng.serverCondition,
							Statistic: tcpMng.statistic,

							config: tcpMng.config,
							logger: tcpMng.logger,
							sync: tcpMng.sync,
						}

						conn.Condition = &condition{
							IsStarted: atomic.NewBool(true),
							IsRestarting: atomic.NewBool(false),
							IsAllowConn: atomic.NewBool(true),
							IsShutdownTimer: atomic.NewBool(false),
						}

						tcpMng.sync.mutex.Lock()
						tcpMng.Connections[stringIPHex] = conn
						tcpMng.sync.mutex.Unlock()

						go tcpMng.connectionHandler(conn)
					} else {
						tcpMng.logger.Error("|" + tcpMng.config.IP + ":" + strconv.Itoa(int(tcpMng.Port)) + "|: " + tcpConn.RemoteAddr().String() + " - attempt to connect to a populated TCP manager.")

						err := tcpConn.Close()
						if err != nil {
							tcpMng.logger.Error(err.Error())

							continue
						}
					}
				} else {
					tcpMng.logger.Error("|" + tcpMng.config.IP + ":" + strconv.Itoa(int(tcpMng.Port)) + "|: " + tcpConn.RemoteAddr().String() + " - attempt to connect to a closed TCP manager.")

					continue
				}
			} else {
				tcpMng.logger.Error("|" + tcpMng.config.IP + ":" + strconv.Itoa(int(tcpMng.Port)) + "|: attempt to connect to a closed server.")

				continue
			}
		default:

		}

		time.Sleep(300 * time.Millisecond)
	}
}

func (tcpMng *TCPManager) Stop() {
	tcpMng.Condition.IsStarted.Store(false)
}

func (tcpMng *TCPManager) Restart() {
	tcpMng.logger.Info("|" + tcpMng.config.IP + ":" + strconv.Itoa(int(tcpMng.Port)) + "|: TCP listener restarting...")
	tcpMng.Stop()
	time.Sleep(3 * time.Second)
	tcpMng.ListenTCP()
}

func (tcpMng *TCPManager) DisableConnection() {
	if tcpMng.Condition.IsAllowConn.Load() {
		tcpMng.Condition.IsAllowConn.Store(false)
		tcpMng.logger.Info("|" + tcpMng.config.IP + ":" + strconv.Itoa(int(tcpMng.Port)) + "|: TCP listener connection prohibited.")
	} else {
		tcpMng.logger.Error("Connection to the TCP manager is already prohibited.")
	}
}

func (tcpMng *TCPManager) AllowConnection() {
	if !tcpMng.Condition.IsAllowConn.Load() {
		tcpMng.Condition.IsAllowConn.Store(true)
		tcpMng.logger.Info("|" + tcpMng.config.IP + ":" + strconv.Itoa(int(tcpMng.Port)) + "|: TCP listener connection allowed.")
	} else {
		tcpMng.logger.Error("Connection to the TCP manager is already allowed.")
	}
}

func (tcpMng *TCPManager) connectionHandler(conn *TCPConnection) {
	defer tcpMng.disconnect(conn)

	conn.sync.wg.Add(1)
	conn.updateDeadline()

	tcpMng.logger.Info("|" + tcpMng.config.IP + ":" + strconv.Itoa(int(tcpMng.Port)) + "|: " + conn.tcpConn.RemoteAddr().String() + " - connected.")

	conn.writer = bufio.NewWriter(conn.tcpConn)
	conn.scanner = bufio.NewScanner(conn.tcpConn)

	conn.Statistic.NumberConnections.Inc()

	for {
		if conn.IsDisconnect.Load() {
			return
		}

		if !tcpMng.Condition.IsStarted.Load() || !tcpMng.serverCondition.IsStarted.Load() {
			return
		}

		select {
		case <-conn.deadline:
			return
		default:

		}

		scanned := conn.scanner.Scan()
		if scanned {
			conn.Statistic.NumberReceivedSegments.Inc()

			go tcpMng.segmentManager.handle(conn)
		}

		if err := conn.scanner.Err(); err != nil {
			tcpMng.logger.Info("|" + tcpMng.config.IP + ":" + strconv.Itoa(int(tcpMng.Port)) + "|: " + conn.tcpConn.RemoteAddr().String() + " - an existing connection was forcibly closed by the remote host.")

			return
		}
	}
}

func (tcpMng *TCPManager) disconnect(conn *TCPConnection) {
	err := conn.tcpConn.Close()
	if err != nil {
		tcpMng.logger.Error(err.Error())

	}

	stringIP := strings.Split(conn.tcpConn.RemoteAddr().String(), ":")[0]

	tcpMng.sync.mutex.Lock()

	i := new(big.Int)
	i.SetString(hex.EncodeToString([]byte(stringIP)), 16)
	stringIPHex := i.Int64()

	delete(tcpMng.Connections, stringIPHex)
	delete(tcpMng.sessions, stringIPHex)

	tcpMng.sync.mutex.Unlock()

	tcpMng.sync.wg.Done()

	tcpMng.logger.Info("|" + tcpMng.config.IP + ":" + strconv.Itoa(int(tcpMng.Port)) + "|: " + conn.tcpConn.RemoteAddr().String() + " - disconnected.")
	conn.Statistic.NumberConnections.Dec()
}
