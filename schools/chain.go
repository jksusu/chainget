package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"
	"time"
)

func chain() {
	ctx := context.Background()
	wsURL := "wss://ethereum-rpc.publicnode.com"
	client, err := ethclient.Dial(wsURL)
	if err != nil {
		log.Fatalf("无法连接到 WebSocket 节点: %v", err)
	}
	defer client.Close()

	number, _ := client.BlockNumber(context.Background())
	fmt.Printf("当前区块高度：%d\n", number)

	var lastTime time.Time
	newBlockLog := make(chan *types.Header, 2048)
	newBlockSub, _ := client.SubscribeNewHead(ctx, newBlockLog)
	defer newBlockSub.Unsubscribe()

	contractAddress := common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7")
	topic := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
		Topics:    [][]common.Hash{{topic}},
	}

	logs := make(chan types.Log, 100)
	sub, err := client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		log.Fatalf("订阅失败: %v", err)
	}
	defer sub.Unsubscribe()

	for {
		select {
		case err := <-sub.Err():
			log.Printf("订阅错误: %v", err)
			return
		case newBlock := <-newBlockLog:
			currentTime := time.Unix(int64(newBlock.Time), 0)
			if !lastTime.IsZero() {
				timeDiff := currentTime.Sub(lastTime)
				fmt.Printf("Block #%d, Hash: %s, Time: %s\n", newBlock.Number, newBlock.Hash().Hex(), currentTime)
				fmt.Printf("Time since last block: %s\n", timeDiff)
			}
			fmt.Printf("区块高度:%d\n", newBlock.Number)
			lastTime = currentTime
		case vLog := <-logs:
			if len(vLog.Topics) == 3 {
				value := new(big.Int).SetBytes(vLog.Data).Int64() / 1000000
				if value < int64(200000) {
					continue
				}
				//获取 usdt 转账
				from := common.HexToAddress(vLog.Topics[1].Hex())
				to := common.HexToAddress(vLog.Topics[2].Hex())

				fmt.Printf("区块号: %d\n", vLog.BlockNumber)
				fmt.Printf("交易哈希: %s\n", vLog.TxHash.Hex())
				fmt.Printf("From: %s\n", from.Hex())
				fmt.Printf("To: %s\n", to.Hex())
				fmt.Printf("Value: %d usdt\n", value)
				fmt.Println("-------------------")
			}
		}
	}
}
