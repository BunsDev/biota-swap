package evm

import (
	"biota_swap/gl"
	"biota_swap/tokens"
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func (ei *EvmIota) StartListen(ch chan *tokens.SwapOrder) {
	nodeUrl := "wss://" + ei.url

	//Set the query filter
	query := ethereum.FilterQuery{
		Addresses: []common.Address{ei.contract},
		Topics:    [][]common.Hash{{EventUnWrap}},
	}

	errOrder := &tokens.SwapOrder{Type: 0}

	//Create the ethclient
	c, err := ethclient.Dial(nodeUrl)
	if err != nil {
		errOrder.Error = fmt.Errorf("The EthWssClient redial error. %v\nThe EthWssClient will be redialed later...\n", err)
		ch <- errOrder
		return
	}
	eventLogChan := make(chan types.Log)
	sub, err := c.SubscribeFilterLogs(context.Background(), query, eventLogChan)
	if err != nil || sub == nil {
		errOrder.Error = fmt.Errorf("Get event logs from eth wss client error. %v\n", err)
		ch <- errOrder
		return
	}
	for {
		select {
		case err := <-sub.Err():
			errOrder.Error = fmt.Errorf("Event wss sub error. %v\nThe EthWssClient will be redialed later...\n", err)
			ch <- errOrder
			return
		case vLog := <-eventLogChan:
			ei.dealTransferEvent(ch, &vLog)
		}
	}
}

func (ei *EvmIota) dealTransferEvent(ch chan *tokens.SwapOrder, vLog *types.Log) {
	errOrder := &tokens.SwapOrder{Type: 1}
	tx := vLog.TxHash.Hex()
	if len(vLog.Data) == 0 {
		errOrder.Error = fmt.Errorf("UnWrap event data is nil. %s, %s, %s\n", tx, vLog.Address.Hex(), vLog.Topics[1].Hex())
		ch <- errOrder
		return
	}
	data := new(big.Int).SetBytes(vLog.Data)
	account := common.HexToAddress(vLog.Topics[1].Hex()).Hex()
	gl.OutLogger.Info("UnWrap token. %s : %s : %s", tx, account, data.String())

	order := &tokens.SwapOrder{
		TxID:   tx,
		From:   common.BytesToAddress(vLog.Topics[1][:]).Hex(),
		To:     common.Bytes2Hex(vLog.Topics[2][:]),
		Amount: data.String(),
		Error:  nil,
	}
	ch <- order
}
