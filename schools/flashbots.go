package main

import (
	"chainget/global"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/metachris/flashbotsrpc"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type FlashBotsClient struct{}

var (
	lock         = make(chan struct{}, 1)
	watchAddress = "0x332C7bF94F4aBBF784F0081c2E7b182d9bDD7e15"
	watchNFTAbi  = `[{"type": "function","name": "enablePresale","inputs": [],"outputs": [],"stateMutability": "nonpayable"}]`
	presaleAbi   = `[{"type": "function","name": "presale","inputs": [{"name": "amount","type": "uint256","internalType": "uint256"}],"outputs": [],"stateMutability": "payable"}]`
	webSocketUrl = "wss://late-small-hexagon.ethereum-sepolia.quiknode.pro/6d67ce9bdbe524c4b75db82dd3a23739801a325a/"
	relayURL     = "https://relay-sepolia.flashbots.net"
	ethRpcUrl    = "https://late-small-hexagon.ethereum-sepolia.quiknode.pro/6d67ce9bdbe524c4b75db82dd3a23739801a325a"
	privateKey   *ecdsa.PrivateKey
	contractABI  abi.ABI
	address      = common.HexToAddress(watchAddress)
)

func (f FlashBotsClient) initData() {
	var err error
	if privateKey, err = crypto.HexToECDSA(global.GetPrivateKey()); err != nil {
		log.Fatalf("解析私钥失败: %v", err)
	}
	//解析合约 ABI
	if contractABI, err = abi.JSON(strings.NewReader(presaleAbi)); err != nil {
		log.Fatalf("解析 ABI 失败: %v", err)
	}
}
func (f FlashBotsClient) Run() {
	f.initData()

	f.Push()
	//go f.WatchPending()

	<-lock
}

func (f FlashBotsClient) GetEnablePresaleSelector() []byte {
	contractABIs, err := abi.JSON(strings.NewReader(watchNFTAbi))
	if err != nil {
		log.Fatal("解析ABI失败:", err)
	}
	return contractABIs.Methods["enablePresale"].ID
}

// 监听
func (f FlashBotsClient) WatchPending() {
	rpcClient, err := rpc.DialContext(context.Background(), webSocketUrl)
	if err != nil {
		log.Fatal("连接RPC失败:", err)
	}
	defer rpcClient.Close()

	client := ethclient.NewClient(rpcClient)

	txChan := make(chan common.Hash)
	sub, err := client.Client().Subscribe(context.Background(), "eth", txChan, "newPendingTransactions")
	if err != nil {
		log.Fatal("订阅pending交易失败:", err)
	}
	defer sub.Unsubscribe()

	//监控合约地址,合约方法选择器获取
	contractAddress := common.HexToAddress(watchAddress)
	enablePresaleSelector := f.GetEnablePresaleSelector()

	fmt.Println("开始监控合约pending：", contractAddress)

	for {
		select {
		case err := <-sub.Err():
			log.Fatal("订阅错误:", err)
		case txHash := <-txChan:
			tx, isPending, err := client.TransactionByHash(context.Background(), txHash)
			if err != nil || !isPending {
				continue
			}
			if tx.To() != nil && *tx.To() == contractAddress && len(tx.Data()) >= 4 {
				if string(tx.Data()[:4]) == string(enablePresaleSelector) {
					//调用 flashbots 打包
					fmt.Printf("检测到pending enablePresale交易: %s\n", tx.Hash().Hex())
				}
			}
		}
	}
}

// 交易签名
func (f FlashBotsClient) signedTx(client *ethclient.Client, gasFeeCap *big.Int) []string {
	var (
		chainID = big.NewInt(11155111)
		sender  = crypto.PubkeyToAddress(privateKey.PublicKey)
	)
	nonce, err := client.PendingNonceAt(context.Background(), sender)
	if err != nil {
		log.Fatalf("获取 nonce 失败: %v", err)
	}
	// 估算 gas
	gasLimit := uint64(200000)
	gasLimit = gasLimit * 12 / 10 // 增加 20% 余量
	amount := big.NewInt(1)
	priceETH := "0.01"
	priceFloat, ok := new(big.Float).SetString(priceETH)
	if !ok {
		log.Fatalf("无效的 NFT 价格: %s", priceETH)
	}
	price, _ := new(big.Float).Mul(priceFloat, big.NewFloat(1)).Int(nil)
	weiPerEth := big.NewInt(1e18)
	price = new(big.Int).Mul(price, weiPerEth) // 转换为 wei
	data, _ := contractABI.Pack("presale", amount)

	signedTx1, _ := types.SignTx(types.NewTx(&types.LegacyTx{Nonce: nonce, To: &address, Value: amount, Gas: gasLimit, GasPrice: gasFeeCap, Data: data}), types.NewEIP155Signer(chainID), privateKey)
	rawTxABytes1, _ := signedTx1.MarshalBinary()
	rawTxHex1 := "0x" + hex.EncodeToString(rawTxABytes1)

	signedTx2, _ := types.SignTx(types.NewTx(&types.LegacyTx{Nonce: nonce + 1, To: &address, Value: amount, Gas: gasLimit, GasPrice: gasFeeCap, Data: data}), types.NewEIP155Signer(chainID), privateKey)
	rawTxABytes2, _ := signedTx2.MarshalBinary()
	rawTxHex2 := "0x" + hex.EncodeToString(rawTxABytes2)

	return []string{
		rawTxHex1,
		rawTxHex2,
	}
}

func (f FlashBotsClient) Push() {
	client, err := ethclient.Dial(ethRpcUrl)
	if err != nil {
		log.Fatalf("连接 RPC 失败: %v", err)
	}
	defer client.Close()

	gasTipCap, err := client.SuggestGasTipCap(context.Background())
	if err != nil {
		log.Fatalf("获取 gas tip cap 失败: %v", err)
	}
	gasFeeCap, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatalf("获取 gas fee cap 失败: %v", err)
	}
	gasFeeCap = new(big.Int).Add(gasFeeCap, gasTipCap)
	fmt.Printf("Gas Tip Cap: %s, Gas Fee Cap: %s\n", gasTipCap.String(), gasFeeCap.String())

	blockNumber, err := client.BlockNumber(context.Background())
	if err != nil {
		log.Fatalf("获取区块号失败: %v", err)
	}
	targetBlock := big.NewInt(int64(blockNumber + 1))

	fmt.Println("区块号码：", targetBlock.String())
	//获取
	signedTxArr := f.signedTx(client, gasFeeCap)

	f.sendBundle(signedTxArr, targetBlock.String())

	// 发送交易
	//err = client.SendTransaction(context.Background(), signedTx)
	//if err != nil {
	//	log.Fatalf("发送交易失败: %v", err)
	//}
	//fmt.Printf("交易已发送，Tx Hash: %s\n", signedTx.Hash().Hex())
	//
	//// 等待交易确认（可选）
	//receipt, err := bind.WaitMined(context.Background(), client, signedTx)
	//if err != nil {
	//	log.Printf("等待交易确认失败: %v", err)
	//} else {
	//	fmt.Printf("交易已确认，Block Number: %d\n", receipt.BlockNumber.Uint64())
	//}
}
func (f FlashBotsClient) GetChainIdAndNonce() (*big.Int, uint64) {
	client, err := ethclient.Dial(ethRpcUrl)
	if err != nil {
		log.Fatalf("连接 RPC 失败: %v", err)
	}
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatalf("获取链ID失败: %v", err)
	}

	// 从私钥获取地址
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("无法获取公钥")
	}
	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	// 获取当前账户的nonce
	nonce, err := client.PendingNonceAt(context.Background(), address)
	if err != nil {
		log.Fatalf("获取nonce失败: %v", err)
	}

	return chainID, nonce
}

func (f FlashBotsClient) getEthClient() *ethclient.Client {
	client, err := ethclient.Dial(ethRpcUrl)
	if err != nil {
		log.Fatalf("连接 RPC 失败: %v", err)
	}
	return client
}

func (f FlashBotsClient) sendBundle(txs []string, blockNumber string) {
	rpc := flashbotsrpc.New(relayURL)
	rpc.Debug = true
	sendBundleArgs := flashbotsrpc.FlashbotsSendBundleRequest{
		Txs:         txs,
		BlockNumber: fmt.Sprintf("0x%x", blockNumber),
	}
	result, err := rpc.FlashbotsSendBundle(privateKey, sendBundleArgs)
	if err != nil {
		if errors.Is(err, flashbotsrpc.ErrRelayErrorResponse) {
			fmt.Println(err.Error())
		} else {
			fmt.Printf("error: %+v\n", err)
		}
		return
	}
	fmt.Printf("%+v\n", result)
}

func (f FlashBotsClient) sendCall(txs []string, blockNumber string) {
	rpc := flashbotsrpc.New(relayURL)
	rpc.Debug = true // 启用调试模式获取更多信息

	fmt.Printf("发送交易到区块: %s, 交易数据长度: %d\n", blockNumber, len(txs[0]))
	opts := flashbotsrpc.FlashbotsCallBundleParam{
		Txs:              txs,
		BlockNumber:      fmt.Sprintf("0x%x", blockNumber),
		StateBlockNumber: "latest",
	}

	result, err := rpc.FlashbotsCallBundle(privateKey, opts)
	if err != nil {
		if errors.Is(err, flashbotsrpc.ErrRelayErrorResponse) {
			fmt.Println("Flashbots Relay错误:", err.Error())
		} else {
			fmt.Printf("发送交易错误: %+v\n", err)
		}
		return
	}
	// 打印详细结果
	fmt.Println("交易发送成功!")
	fmt.Printf("结果: %+v\n", result)
}
