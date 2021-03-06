package jklserver

import (
	"bufio"
	"encoding/json"
	"errors"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"net"
	"time"
)

type TCPConnection struct {
	BD						interface{}
	tcpConn					net.Conn
	Session					*session

	IsDisconnect			*atomic.Bool
	IsLogin					*atomic.Bool

	Statistic				*statistic
	tcpManagerCondition		*condition
	serverCondition			*condition
	Condition				*condition

	deadline				<-chan time.Time
	writer                   *bufio.Writer
	scanner                  *bufio.Scanner

	config					*Config
	logger					*zap.Logger
	sync					*syncing
}

func (conn *TCPConnection) updateDeadline() {
	conn.deadline = time.After(conn.config.LifeTimeConnection)
}

func (conn *TCPConnection) Disconnect() {
	conn.IsDisconnect.Store(true)
}

func (conn *TCPConnection) ReadBytes() ([]byte, error) {
	buf := conn.scanner.Bytes()

	if len(buf) == 0 {
		conn.logger.Error("The segment has zero length.")

		return []byte{}, errors.New("The segment has zero length. ")
	}

	return buf, nil
}

func (conn *TCPConnection) WriteBytes(buf []byte) (int, error) {
	if conn.serverCondition.IsStarted.Load() {
		if len(buf) == 0 {
			return 0, errors.New("The segment has zero length. ")
		}

		conn.updateDeadline()
		conn.Statistic.NumberSentSegments.Inc()

		separator := []byte(conn.config.Separator)

		n, err := conn.writer.Write(append(buf, separator...))
		err = conn.writer.Flush()

		return n, err
	}

	return 0, errors.New("Attempt to send a segment from a disabled server. ")
}

func (conn *TCPConnection) Read() (*Segment, error) {
	segment := &Segment{
		Head: segmentHead{},
	}

	jsonByte, err := conn.ReadBytes()
	if err != nil {
		conn.logger.Error(err.Error())

		return &Segment{}, err
	}
	if len(jsonByte) == 0 {
		conn.logger.Error("The segment has zero length.")

		return segment, errors.New("The segment has zero length. ")
	}

	err = json.Unmarshal(jsonByte, &segment)
	if err != nil {
		conn.logger.Error(err.Error())

		return segment, err
	}

	return segment, nil

}

func (conn *TCPConnection) Write(segment *Segment) error {
	buf, err := json.Marshal(segment)
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