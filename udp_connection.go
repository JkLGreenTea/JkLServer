package jklserver

import (
	"encoding/json"
	"errors"
	"go.uber.org/zap"
	"net"
)

type UDPConnection struct {
	BD						interface{}
	RemoteAddr 				*net.UDPAddr
	Session					*session
	Statistic				*statistic

	udpConn					*net.UDPConn

	config					*Config
	logger					*zap.Logger
	sync					*syncing
}

func (conn *UDPConnection) WriteBytes(buf []byte) (int, error) {
	if len(buf) == 0 {
		conn.logger.Error("The package has zero length.")

		return 0, errors.New("The package has zero length. ")
	}

	conn.Statistic.NumberSentPackages.Inc()

	return conn.udpConn.WriteTo(buf, conn.RemoteAddr)
}

func (conn *UDPConnection) Write(pack *Package) error {
	buf, err := json.Marshal(pack)
	if err != nil {
		conn.logger.Error(err.Error())

		return err
	}

	_, err = conn.WriteBytes(buf)
	if err != nil {
		conn.logger.Error(err.Error())

		return err
	}

	return nil
}

func (conn *UDPConnection) WriteBytesTo(buf []byte, addr net.Addr) (int, error) {
	if len(buf) == 0 {
		conn.logger.Error("The package has zero length.")

		return 0, errors.New("The package has zero length. ")
	}

	conn.Statistic.NumberSentPackages.Inc()

	return conn.udpConn.WriteTo(buf, addr)
}

func (conn *UDPConnection) WriteTo(pack *Package, addr net.Addr) error {
	buf, err := json.Marshal(pack)
	if err != nil {
		conn.logger.Error(err.Error())

		return err
	}

	_, err = conn.WriteBytesTo(buf, addr)
	if err != nil {
		return err
	}

	return nil
}
