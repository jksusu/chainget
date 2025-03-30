package main

import (
	"chainget/pkg/helper"
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"
	"strings"
	"time"
)

type Ido struct{}

var (
	ctx             = context.Background()
	rpcUrl          = "wss://bsc-rpc.publicnode.com"
	contractAddress common.Address
	contractAbi     abi.ABI
)

func topic() [][]common.Hash {
	var err error
	contractAbi, err = abi.JSON(strings.NewReader(helper.ReadAbiJson("binance_ido_abi.json")))
	if err != nil {
		log.Fatalf("❌ Failed to parse ABI: %v", err)
	}
	//事件签名 创建 IDO 事件 设置池子事件
	NewIDOContractEvent := contractAbi.Events["NewIDOContract"]
	if NewIDOContractEvent.ID == (common.Hash{}) {
		log.Fatalf("❌ Event 'NewIDOContractEvent' not found in ABI")
	}
	PoolParametersSetEvent := contractAbi.Events["PoolParametersSet"]
	if PoolParametersSetEvent.ID == (common.Hash{}) {
		log.Fatalf("❌ Event 'PoolParametersSet' not found in ABI")
	}
	return [][]common.Hash{{
		NewIDOContractEvent.ID,
		PoolParametersSetEvent.ID,
	}}
}

func watch() {
	contractAddress = common.HexToAddress("0xe0C7897d48847b6916094bF5cD8216449Ea8fB86")
	client, err := ethclient.Dial(rpcUrl)
	if err != nil {
		log.Fatalf("connect rpc error: %v", err)
	}
	defer client.Close()

	number, _ := client.BlockNumber(context.Background())
	go func() {
		for {
			number, _ = client.BlockNumber(context.Background())
			log.Println("block number: ", number)
			time.Sleep(1000 * time.Second)
		}
	}()

	var (
		logsChannel = make(chan types.Log, 100)
		query       = ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(number)),
			Addresses: []common.Address{contractAddress},
			Topics:    topic(),
		}
	)
	// 订阅事件
	sub, err := client.SubscribeFilterLogs(ctx, query, logsChannel)
	if err != nil {
		log.Fatalf("❌ Failed to subscribe to logs: %v", err)
	}
	defer sub.Unsubscribe()

	fmt.Println("开始监听事件...")

	// 监听循环
	for {
		select {
		case errs := <-sub.Err():
			log.Fatalf("❌ Subscription error: %v", errs)
		case vLog := <-logsChannel:
			handleLog(vLog)
		}
	}
}

// handleLog 处理不同的事件日志
func handleLog(vLog types.Log) {
	var (
		eventID   = vLog.Topics[0]
		eventName string
	)
	// 查找事件名称
	for name, event := range contractAbi.Events {
		if event.ID == eventID {
			eventName = name
			break
		}
	}
	if eventName == "" {
		log.Printf("⚠️ Unknown event ID: %s", eventID.Hex())
		return
	}

	fmt.Printf("时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("块号: %d\n", vLog.BlockNumber)
	fmt.Printf("交易哈希: %s\n", vLog.TxHash.Hex())
	fmt.Printf("事件: %s\n", eventName)

	switch eventName {
	case "NewIDOContract":
		idoAddress := common.HexToAddress(vLog.Topics[1].Hex())
		fmt.Printf("idoAddress: %s\n", idoAddress.Hex())
	case "PoolParametersSet":
		if len(vLog.Topics) > 1 {
			param1 := common.HexToAddress(vLog.Topics[1].Hex())
			fmt.Printf("param1: %s\n", param1.Hex())
		}
	default:
		fmt.Println("未处理的事件")
	}
	fmt.Println("------------------------")
}
