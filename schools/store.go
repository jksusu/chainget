package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"log"
	"math/big"
)

// 0x20f48b9d9d019ad7cb8113c85c99940e3b224ffc 合约地址
func subSlot() {
	rpcClient, err := rpc.Dial("https://sepolia.drpc.org")
	if err != nil {
		log.Fatalf("Failed to connect to RPC: %v", err)
	}
	client := ethclient.NewClient(rpcClient)
	defer rpcClient.Close()

	// 替换为实际的 esRNT 合约地址
	contractAddress := common.HexToAddress("0x20f48b9d9d019ad7cb8113c85c99940e3b224ffc")
	// 读取数组长度（插槽 0）
	lengthSlot := common.Big0
	lengthHex, err := client.StorageAt(context.Background(), contractAddress, common.BigToHash(lengthSlot), nil)
	if err != nil {
		log.Fatalf("Failed to read array length: %v", err)
	}
	length := new(big.Int).SetBytes(lengthHex[:]).Int64()
	fmt.Printf("Array length: %d\n", length)
	// 计算数组数据的起始插槽
	dataStartSlot := common.BytesToHash(common.BigToHash(lengthSlot).Bytes())
	dataStart := new(big.Int).SetBytes(dataStartSlot[:])
	// 遍历数组的每个元素
	for i := int64(0); i < length; i++ {
		// 每个元素占用 2 个插槽，计算当前元素的起始插槽
		elementStartSlot := new(big.Int).Add(dataStart, big.NewInt(i*2))

		// 读取 user（插槽 elementStartSlot）
		userSlot := common.BigToHash(elementStartSlot)
		userHex, err := client.StorageAt(context.Background(), contractAddress, userSlot, nil)
		if err != nil {
			log.Fatalf("Failed to read user at index %d: %v", i, err)
		}
		user := common.BytesToAddress(userHex[12:]) // 地址占用低 20 字节

		// 读取 startTime 和 amount（插槽 elementStartSlot+1）
		dataSlot := common.BigToHash(new(big.Int).Add(elementStartSlot, common.Big1))
		dataHex, err := client.StorageAt(context.Background(), contractAddress, dataSlot, nil)
		if err != nil {
			log.Fatalf("Failed to read data at index %d: %v", i, err)
		}
		startTime := new(big.Int).SetBytes(dataHex[24:32]).Uint64() // startTime 占用低 8 字节
		amount := new(big.Int).SetBytes(dataHex[:24])               // amount 占用高 24 字节
		// 按指定格式打印
		fmt.Printf("locks[%d]: user: %s, startTime: %d, amount: %s\n", i, user.Hex(), startTime, amount.String())
	}
}
