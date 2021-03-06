package jklserver

import (
	"./status"
	"encoding/json"
	"errors"
	"go.uber.org/zap"
)

type Package struct {
	Head	packageHead				`json:"head"`
	Body	map[string]interface{}	`json:"body"`
}

type packageHead struct {
	ID		uint32				`json:"id"`
	Status	status.Status		`json:"status"`
}

type packageManager struct {
	bd					interface{}
	handleFuncPackage	map[uint32]*func(pack *Package, conn *UDPConnection)

	config				*Config
	logger				*zap.Logger
	sync				*syncing
}

func (packageMng *packageManager) handle(pack *Package, conn *UDPConnection) {
	defer packageMng.sync.wg.Done()

	packageMng.sync.wg.Add(1)

	if segmentHandler, ok := packageMng.handleFuncPackage[pack.Head.ID]; ok {
		segmentHandlerFunc := *segmentHandler
		segmentHandlerFunc(pack, conn)

		return
	}
}

func (packageMng *packageManager) read(buf []byte) (*Package, error){
	pack := &Package{
		Head: packageHead{},
	}

	if len(buf) == 0 {
		return pack, errors.New("The package has zero length. ")
	}

	err := json.Unmarshal(buf, &pack)
	if err != nil {
		return pack, err
	}


	return pack, nil
}