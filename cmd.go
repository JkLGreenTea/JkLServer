package jklserver

import (
	"bufio"
	"encoding/hex"
	"math/big"
	"os"
	"strconv"
	"strings"
)

type command struct {
	Title		string
	Func		func(srv *server)
}

func (srv *server) newCmd(pack string, title string, func_ func(srv *server)) {
	title = strings.ToLower(title)
	pack = strings.ToLower(pack)

	cmd := &command{
		Title: title,
		Func: func_,
	}

	i := new(big.Int)
	i.SetString(hex.EncodeToString([]byte(pack)), 16)
	packHex := i.Int64()

	i = new(big.Int)
	i.SetString(hex.EncodeToString([]byte(title)), 16)
	titleHex := i.Int64()

	srv.sync.mutex.Lock()
	srv.commands[packHex][titleHex] = cmd
	srv.sync.mutex.Unlock()
}

func (srv *server) createDefaultCmd() {
	i := new(big.Int)
	i.SetString(hex.EncodeToString([]byte("server")), 16)
	packHex := i.Int64()

	srv.sync.mutex.Lock()
	srv.commands = make(map[int64]map[int64]*command)
	srv.commands[packHex] = make(map[int64]*command)
	srv.sync.mutex.Unlock()

	srv.newCmd("server", "Statistic", cliGetStatisticServer)
	srv.newCmd("server", "Listen", cliListenServer)
	srv.newCmd("server", "Stop", cliStopServer)
	srv.newCmd("server", "Restart", cliRestartServer)
	srv.newCmd("server", "Status", cliGetConditionServer)
	srv.newCmd("server", "Disable", cliDisableConnectionServer)
	srv.newCmd("server", "Allow", cliAllowConnectionServer)
	srv.newCmd("server", "List", cliListManagers)
}

func (srv *server) readingCmd()  {
	defer srv.Statistic.NumberSystemGoroutines.Dec()

	srv.Statistic.NumberSystemGoroutines.Inc()

	for {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		cmd := scanner.Text()

		if cmd != "" {
			srv.logger.Info("CLI command: " + cmd)

			go srv.handleCmd(cmd)
		}
	}
}

func (srv *server) handleCmd(cmd string) {
	cmd = strings.Trim(cmd, " ")
	cmd = strings.ToLower(cmd)

	cmdSplit := strings.Split(cmd, ".")

	if len(cmdSplit) == 0 {
		return
	}

	cmdTitle := strings.Split(cmdSplit[1], " ")[0]
	pack := cmdSplit[0]

	i := new(big.Int)
	i.SetString(hex.EncodeToString([]byte(pack)), 16)
	packHex := i.Int64()

	i = new(big.Int)
	i.SetString(hex.EncodeToString([]byte(cmdTitle)), 16)
	cmdTitleHex := i.Int64()

	switch pack {
	case "tcp": {
		switch cmdTitle {
		case "list": {
			srv.sync.rwMutex.Lock()
			for _, mng := range srv.ThreadsManager.tcpManagers {
				var condition string

				if mng.Condition.IsStarted.Load() {
					condition = "running..."
				} else {
					condition = "stopped..."
				}

				srv.logger.Info("|" + srv.Config.IP + ":" + strconv.Itoa(int(mng.Port)) + "|: TCP Listener " + condition)
			}
			srv.sync.rwMutex.Unlock()
		}
		case "new":
			attributes := strings.Split(cmd, "$")

			attr1, err := strconv.Atoi(strings.Trim(attributes[1], " "))
			if err != nil {
				srv.logger.Error("* Unknown command.")
				return
			}
			attr2, err := strconv.Atoi(strings.Trim(attributes[2], " "))
			if err != nil {
				srv.logger.Error("* Unknown command.")
				return
			}

			srv.ThreadsManager.NewTCPManager(uint16(attr1), uint16(attr2))
		case "stop": {
			attributes := strings.Split(cmd, "$")

			attr1, err := strconv.Atoi(strings.Trim(attributes[1], " "))
			if err != nil {
				srv.logger.Error("* Unknown command.")
				return
			}

			srv.ThreadsManager.sync.rwMutex.Lock()
			srv.ThreadsManager.tcpManagers[uint16(attr1)].Stop()
			srv.ThreadsManager.sync.rwMutex.Unlock()
		}
		case "restart": {
			attributes := strings.Split(cmd, "$")

			attr1, err := strconv.Atoi(strings.Trim(attributes[1], " "))
			if err != nil {
				srv.logger.Error("* Unknown command.")
				return
			}

			srv.ThreadsManager.sync.rwMutex.Lock()
			srv.ThreadsManager.tcpManagers[uint16(attr1)].Restart()
			srv.ThreadsManager.sync.rwMutex.Unlock()
		}
		case "delete": {
			attributes := strings.Split(cmd, "$")

			attr1, err := strconv.Atoi(strings.Trim(attributes[1], " "))
			if err != nil {
				srv.logger.Error("* Unknown command.")
				return
			}

			srv.sync.mutex.Lock()
			delete(srv.ThreadsManager.tcpManagers, uint16(attr1))
			srv.sync.mutex.Unlock()
		}
		case "disable": {
			attributes := strings.Split(cmd, "$")

			attr1, err := strconv.Atoi(strings.Trim(attributes[1], " "))
			if err != nil {
				srv.logger.Error("* Unknown command.")
				return
			}

			srv.ThreadsManager.tcpManagers[uint16(attr1)].DisableConnection()
		}
		case "allow": {
			attributes := strings.Split(cmd, "$")

			attr1, err := strconv.Atoi(strings.Trim(attributes[1], " "))
			if err != nil {
				srv.logger.Error("* Unknown command.")
				return
			}

			srv.ThreadsManager.tcpManagers[uint16(attr1)].AllowConnection()
		}
		case "status": {
			attributes := strings.Split(cmd, "$")

			attr1, err := strconv.Atoi(strings.Trim(attributes[1], " "))
			if err != nil {
				srv.logger.Error("* Unknown command.")
				return
			}

			cliGetConditionTCPManager(uint16(attr1), srv)
		}
		case "listen": {
			attributes := strings.Split(cmd, "$")

			attr1, err := strconv.Atoi(strings.Trim(attributes[1], " "))
			if err != nil {
				srv.logger.Error("* Unknown command.")
				return
			}

			srv.sync.rwMutex.Lock()
			if mng, ok := srv.ThreadsManager.tcpManagers[uint16(attr1)]; !ok {
				mng.ListenTCP()
			} else {
				srv.logger.Error("TCP manager is already running.")
			}
			srv.sync.rwMutex.Unlock()
		}
		default: srv.logger.Error("* Unknown command.")
		}
	}
	case "udp": {
		switch cmdTitle {
		case "list": {
			srv.sync.rwMutex.Lock()
			for _, mng := range srv.ThreadsManager.udpManagers {
				var condition string

				if mng.Condition.IsStarted.Load() {
					condition = "running..."
				} else {
					condition = "stopped..."
				}

				srv.logger.Info("|" + srv.Config.IP + ":" + strconv.Itoa(int(mng.Port)) + "|: UDP Listener " + condition)
			}
			srv.sync.rwMutex.Unlock()
		}
		case "new": {
			attributes := strings.Split(cmd, "$")

			attr1, err := strconv.Atoi(strings.Trim(attributes[1], " "))
			if err != nil {
				srv.logger.Error("* Unknown command.")
				return
			}

			srv.ThreadsManager.NewUDPManager(uint16(attr1))
		}
		case "stop": {
			attributes := strings.Split(cmd, "$")

			attr1, err := strconv.Atoi(strings.Trim(attributes[1], " "))
			if err != nil {
				srv.logger.Error("* Unknown command.")
				return
			}

			srv.ThreadsManager.udpManagers[uint16(attr1)].Stop()
		}
		case "restart": {
			attributes := strings.Split(cmd, "$")

			attr1, err := strconv.Atoi(strings.Trim(attributes[1], " "))
			if err != nil {
				srv.logger.Error("* Unknown command.")
				return
			}

			srv.ThreadsManager.udpManagers[uint16(attr1)].Restart()
		}
		case "delete": {
			attributes := strings.Split(cmd, "$")

			attr1, err := strconv.Atoi(strings.Trim(attributes[1], " "))
			if err != nil {
				srv.logger.Error("* Unknown command.")
				return
			}

			srv.sync.mutex.Lock()
			delete(srv.ThreadsManager.udpManagers, uint16(attr1))
			srv.sync.mutex.Unlock()
		}
		case "disable": {
			attributes := strings.Split(cmd, "$")

			attr1, err := strconv.Atoi(strings.Trim(attributes[1], " "))
			if err != nil {
				srv.logger.Error("* Unknown command.")
				return
			}

			srv.ThreadsManager.udpManagers[uint16(attr1)].DisableConnection()
		}
		case "allow": {
			attributes := strings.Split(cmd, "$")

			attr1, err := strconv.Atoi(strings.Trim(attributes[1], " "))
			if err != nil {
				srv.logger.Error("* Unknown command.")
				return
			}

			srv.ThreadsManager.udpManagers[uint16(attr1)].AllowConnection()
		}
		case "status": {
			attributes := strings.Split(cmd, "$")

			attr1, err := strconv.Atoi(strings.Trim(attributes[1], " "))
			if err != nil {
				srv.logger.Error("* Unknown command.")
				return
			}

			cliGetConditionUDPManager(uint16(attr1), srv)
		}
		case "listen": {
			attributes := strings.Split(cmd, "$")

			attr1, err := strconv.Atoi(strings.Trim(attributes[1], " "))
			if err != nil {
				srv.logger.Error("* Unknown command.")
				return
			}

			srv.sync.rwMutex.Lock()
			if mng, ok := srv.ThreadsManager.udpManagers[uint16(attr1)]; !ok {
				mng.ListenUDP()
			} else {
				srv.logger.Error("UDP manager is already running.")
			}
			srv.sync.rwMutex.Unlock()
		}
		default: srv.logger.Error("* Unknown command.")
		}
	}
	default: {
		srv.sync.rwMutex.Lock()
		commands, ok := srv.commands[packHex]
		if ok {
			cmd, ok := commands[cmdTitleHex]
			if ok {
				go cmd.Func(srv)
			} else {
				srv.logger.Error("Command not found.")
			}
		} else {
			srv.logger.Error("Command package not found.")
		}
		srv.sync.rwMutex.Unlock()
	}
	}
}

func cliGetStatisticServer(srv *server) {
	srv.logger.Info("NumberReceivedSegments: " + srv.Statistic.NumberReceivedSegments.String())
	srv.logger.Info("NumberReceivedPackages: " + srv.Statistic.NumberReceivedPackages.String())
	srv.logger.Info("NumberSentSegments: " + srv.Statistic.NumberSentSegments.String())
	srv.logger.Info("NumberSentPackages: " + srv.Statistic.NumberSentPackages.String())
	srv.logger.Info("NumberSystemGoroutines: " + srv.Statistic.NumberSystemGoroutines.String())
	srv.logger.Info("NumberGoroutines: " + srv.Statistic.NumberGoroutines.String())
	srv.logger.Info("NumberConnections: " + srv.Statistic.NumberConnections.String())
}

func cliListenServer(srv *server) {
	srv.listen()
}

func cliStopServer(srv *server) {
	srv.Stop()
}

func cliRestartServer(srv *server) {
	srv.Restarting()
}

func cliGetConditionServer(srv *server) {
	var condition string

	if srv.Condition.IsRestarting.Load() {
		condition = "Rebooting..."
	} else if srv.Condition.IsShutdownTimer.Load() {
		condition = "Stop..."
	} else if srv.Condition.IsStarted.Load() {
		condition = "Running..."
	} else if !srv.Condition.IsStarted.Load() {
		condition = "Stopped..."
	}

	srv.logger.Info("Status: " + condition)
	srv.logger.Info("Status connection: " + srv.Condition.IsAllowConn.String())
}

func cliDisableConnectionServer(srv *server) {
	srv.DisableConnection()
}

func cliAllowConnectionServer(srv *server) {
	srv.AllowConnection()
}

func cliListManagers(srv *server) {
	srv.sync.rwMutex.Lock()
	for _, mng := range srv.ThreadsManager.tcpManagers {
		var condition string

		if mng.Condition.IsStarted.Load() {
			condition = "running..."
		} else {
			condition = "stopped..."
		}

		srv.logger.Info("|" + srv.Config.IP + ":" + strconv.Itoa(int(mng.Port)) + "|: TCP Listener " + condition)
	}

	for _, mng := range srv.ThreadsManager.udpManagers {
		var condition string

		if mng.Condition.IsStarted.Load() {
			condition = "running..."
		} else {
			condition = "stopped..."
		}

		srv.logger.Info("|" + srv.Config.IP + ":" + strconv.Itoa(int(mng.Port)) + "|: UDP Listener " + condition)
	}
	srv.sync.rwMutex.Unlock()
}

func cliGetConditionTCPManager(id uint16, srv *server) {
	var condition string

	if srv.ThreadsManager.tcpManagers[id].Condition.IsShutdownTimer.Load() {
		condition = "Stop..."
	} else if srv.ThreadsManager.tcpManagers[id].Condition.IsStarted.Load() {
		condition = "Running..."
	} else if !srv.ThreadsManager.tcpManagers[id].Condition.IsStarted.Load() {
		condition = "Stopped..."
	}

	srv.logger.Info("Status: " + condition)
	srv.logger.Info("Status connection: " + srv.ThreadsManager.tcpManagers[id].Condition.IsAllowConn.String())
}

func cliGetConditionUDPManager(id uint16, srv *server) {
	var condition string

	if srv.ThreadsManager.udpManagers[id].Condition.IsShutdownTimer.Load() {
		condition = "Stop..."
	} else if srv.ThreadsManager.udpManagers[id].Condition.IsStarted.Load() {
		condition = "Running..."
	} else if !srv.ThreadsManager.udpManagers[id].Condition.IsStarted.Load() {
		condition = "Stopped..."
	}

	srv.logger.Info("Status: " + condition)
	srv.logger.Info("Status connection: " + srv.ThreadsManager.udpManagers[id].Condition.IsAllowConn.String())
}