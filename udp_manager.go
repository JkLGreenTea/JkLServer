package jklserver

import (
	"encoding/hex"
	"go.uber.org/zap"
	"math/big"
	"net"
	"strconv"
	"time"
)

type UDPManager struct {
	Port						uint16

	Condition					*condition
	packageManager				*packageManager
	serverCondition				*condition
	udpListener					*net.UDPConn
	sessions					map[int64]*session
	statistic					*statistic

	config						*Config
	logger						*zap.Logger
	sync						*syncing
}

func (udpMng *UDPManager) ListenUDP() {
	defer func() {
		udpMng.sync.wg.Done()

		udpMng.Condition.IsStarted.Store(false)
		udpMng.Condition.IsAllowConn.Store(false)

		udpMng.logger.Info("|" + udpMng.config.IP + ":" + strconv.Itoa(int(udpMng.Port)) + "|: UDP listener is closed.")

		udpMng.udpListener = nil
	}()

	udpMng.sync.wg.Add(1)

	udpMng.Condition.IsStarted.Store(true)
	udpMng.Condition.IsAllowConn.Store(true)

	addr := net.UDPAddr{
		Port: int(udpMng.Port),
		IP:   net.ParseIP(udpMng.config.IP),
	}

	udpListener, err := net.ListenUDP("udp", &addr)
	if err != nil {
		udpMng.logger.Error(err.Error())

		return
	}

	udpMng.udpListener = udpListener

	udpMng.logger.Info("|" + udpMng.config.IP + ":" + strconv.Itoa(int(udpMng.Port)) + "|: UDP listener started.")

	go func() {
		defer func() {
			udpMng.sync.wg.Done()

			if udpMng.udpListener != nil {
				err := udpMng.udpListener.Close()
				if err != nil {
					udpMng.logger.Error(err.Error())
				}
			}
		}()

		udpMng.sync.wg.Add(1)

		for {
			if !udpMng.Condition.IsStarted.Load() || !udpMng.serverCondition.IsStarted.Load() {
				return
			}

			time.Sleep(1 * time.Second)
		}
	}()

	for {
		if !udpMng.Condition.IsStarted.Load() || !udpMng.serverCondition.IsStarted.Load() {
			return
		}
		buf := make([]byte, udpMng.config.BufSize)

		n, remoteAddr, err := udpMng.udpListener.ReadFromUDP(buf)
		if err != nil {
			udpMng.logger.Error(err.Error())

			continue
		}

		if udpMng.serverCondition.IsAllowConn.Load() {
			if udpMng.Condition.IsAllowConn.Load() {

					i := new(big.Int)
					i.SetString(hex.EncodeToString([]byte(remoteAddr.IP.String())), 16)
					remoteAddrHex := i.Int64()

					udpMng.sync.rwMutex.Lock()
					_, ok := udpMng.sessions[remoteAddrHex]
					udpMng.sync.rwMutex.Unlock()

					if  ok {
						pack, err := udpMng.packageManager.read(buf[:n])
						if err == nil {
							udpMng.sync.rwMutex.Lock()
							session_ := udpMng.sessions[remoteAddrHex]
							udpMng.sync.rwMutex.Unlock()

							conn := &UDPConnection{
								BD: udpMng.packageManager.bd,

								RemoteAddr: remoteAddr,
								udpConn: udpMng.udpListener,
								Session: session_,

								Statistic: udpMng.statistic,

								config: udpMng.config,
								logger: udpMng.logger,
								sync: udpMng.sync,
							}

							if remoteAddr != nil {
								conn.Statistic.NumberReceivedPackages.Inc()

								go udpMng.packageManager.handle(pack, conn)
							}
						} else {
							udpMng.logger.Error("|" + udpMng.config.IP + ":" + strconv.Itoa(int(udpMng.Port)) + "|: " + err.Error())

							continue
						}
					} else {
						continue
					}
			} else {
				udpMng.logger.Error("|" + udpMng.config.IP + ":" + strconv.Itoa(int(udpMng.Port)) + "|: attempt to connect to a closed UDP manager.")

				continue
			}
		} else {
			udpMng.logger.Error("|" + udpMng.config.IP + ":" + strconv.Itoa(int(udpMng.Port)) + "|: attempt to connect to a closed server.")

			continue
		}
	}
}

func (udpMng *UDPManager) Stop() {
	udpMng.Condition.IsStarted.Store(false)
}

func (udpMng *UDPManager) Restart() {
	udpMng.logger.Info("|" + udpMng.config.IP + ":" + strconv.Itoa(int(udpMng.Port)) + "|: UDP listener restarting...")
	udpMng.Stop()
	time.Sleep(3 * time.Second)
	udpMng.ListenUDP()
}

func (udpMng *UDPManager) DisableConnection() {
	if udpMng.Condition.IsAllowConn.Load() {
		udpMng.Condition.IsAllowConn.Store(false)
		udpMng.logger.Info("|" + udpMng.config.IP + ":" + strconv.Itoa(int(udpMng.Port)) + "|: UDP listener connection prohibited.")
	} else {
		udpMng.logger.Error("Connection to the UDP manager is already prohibited.")
	}
}

func (udpMng *UDPManager) AllowConnection() {
	if !udpMng.Condition.IsAllowConn.Load() {
		udpMng.Condition.IsAllowConn.Store(false)
		udpMng.logger.Info("|" + udpMng.config.IP + ":" + strconv.Itoa(int(udpMng.Port)) + "|: UDP listener connection allowed.")
	} else {
		udpMng.logger.Error("Connection to the UDP manager is already allowed.")
	}
}