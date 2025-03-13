package chainget

import (
	"context"
	"encoding/hex"
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

var lock = make(chan struct{}, 1)

func main() {
	ctx := context.Background()
	wsURL := "wss://ethereum-rpc.publicnode.com"
	client, err := ethclient.Dial(wsURL)
	if err != nil {
		log.Fatalf("无法连接到 WebSocket 节点: %v", err)
	}
	defer client.Close()

	go func() {
		for {
			number, _ := client.BlockNumber(context.Background())
			fmt.Printf("当前区块高度：%d\n", number)
			time.Sleep(1000 * time.Second)
		}
	}()

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

	// 创建通道接收待处理交易哈希
	txHashes := make(chan common.Hash)
	subPending, err := client.Client().Subscribe(ctx, "eth", txHashes, "newPendingTransactions")
	if err != nil {
		log.Fatalf("Failed to subscribe to newPendingTransactions: %v", err)
	}
	defer subPending.Unsubscribe()

	fmt.Println("开始监听 USDT Transfer 事件...")
	for {
		select {
		case err := <-sub.Err():
			log.Printf("订阅错误: %v", err)
			//实现重连
			return
		case vLog := <-logs:
			if len(vLog.Topics) == 3 {
				value := new(big.Int).SetBytes(vLog.Data).Int64() / 1000000
				if value < int64(200000) {
					//continue
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
		case txHash := <-txHashes:
			fmt.Printf("New pending transaction: %s\n", txHash.Hex())
			tx, isPending, err := client.TransactionByHash(context.Background(), txHash)
			if err != nil {
				log.Printf("Failed to fetch transaction %s: %v", txHash.Hex(), err)
				continue
			}
			if !isPending {
				fmt.Printf("Transaction %s is no longer pending\n", txHash.Hex())
				continue
			}
			// 打印交易详情
			fmt.Printf("From: %s\n", getTransactionSender(tx))
			fmt.Printf("To: %s\n", tx.To().Hex())
			fmt.Printf("Value: %s\n", tx.Value().String())
			fmt.Printf("Gas: %d\n", tx.Gas())
			fmt.Printf("GasPrice: %s\n", tx.GasPrice().String())
			fmt.Printf("Data: %x\n", string(tx.Data()))
		}
	}
}

// getTransactionSender 获取交易发送者地址
func getTransactionSender(tx *types.Transaction) common.Address {
	signer := types.LatestSignerForChainID(tx.ChainId())
	sender, err := types.Sender(signer, tx)
	if err != nil {
		return common.Address{}
	}
	return sender
}

// 解析交易内容
func rawParseParams(data []byte) {
	if len(data) == 0 {
		fmt.Println("交易数据为空，当前交易为 ETH 转账")
		return
	}
	if len(data) < 8 {
		fmt.Println("方法参数不存在")
		return
	}
	methodID := data[:4] //交易方法 16进制
	params := data[4:]   //交易参数

	bytes, _ := hex.DecodeString(string(methodID))
	fmt.Println("方法ID:", bytes)

	fmt.Printf("MethodID: 0x%x\n", methodID)
	//解析 参数 32字节 为一组数据
	if len(params) < 32 {
		fmt.Println("方法参数长度不足 32 字节")
		return
	}

	countLen := len(params) / 32
	fmt.Println("一共有", countLen, "个参数")

	for i := 0; i < countLen; i++ {
		param := params[i*32 : (i+1)*32]
		fmt.Printf("参数 %d: %x\n", i, param)
	}

}
