package main

import (
	"chainget/global"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/params"
	"github.com/lmittmann/w3/module/eth"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/lmittmann/flashbots"
	"github.com/lmittmann/w3"
)

type ItmFlashBot struct {
	Sender          common.Address
	PrivateKey      *ecdsa.PrivateKey
	ETHRpcUrl       string
	FlashBotsRpcUrl string
	ContractAddress common.Address
	ContractABI     abi.ABI
	ChainID         *big.Int
	Lock            chan int
	Debug           bool
}

var ctx = context.Background()
var abiJson = `[{"type": "function","name": "presale","inputs": [{"name": "amount","type": "uint256","internalType": "uint256"}],"outputs": [],"stateMutability": "payable"},{"type": "function","name": "enablePresale","inputs": [],"outputs": [],"stateMutability": "nonpayable"}]`

func NewItmClient() *ItmFlashBot {
	pk, err := crypto.HexToECDSA(global.Viper.GetString("key.privateKey"))
	if err != nil {
		log.Fatalf("❌ 解析私钥失败: %v", err)
	}
	abiAnalysis, err := abi.JSON(strings.NewReader(abiJson))
	if err != nil {
		log.Fatalf("❌ ABI解析失败: %v", err)
	}
	return &ItmFlashBot{
		Sender:          crypto.PubkeyToAddress(pk.PublicKey),
		PrivateKey:      pk,
		ETHRpcUrl:       global.Viper.GetString("rpc.ethRpcUrl"),
		FlashBotsRpcUrl: global.Viper.GetString("rpc.flashBotsRpcUrl"),
		ContractAddress: common.HexToAddress(global.Viper.GetString("key.contractAddress")),
		ContractABI:     abiAnalysis,
		ChainID:         big.NewInt(global.Viper.GetInt64("chainId")),
		Lock:            make(chan int),
		Debug:           global.Viper.GetBool("debug"),
	}
}

func (f ItmFlashBot) Run() {
	//监控

	//打包执行
	f.StartFlashBots()

	<-f.Lock
}

// StartFlashBots 开始执行发送到 flashbots 流程
func (f ItmFlashBot) StartFlashBots() {
	var (
		nonce, gasPrice, lastBlock = f.GetParams()
		txArr                      = f.GetTxsArr(nonce, gasPrice)
	)
	//发送交易到本地执行
	log.Printf("✔️ nonce: %v", nonce)
	log.Printf("✔️ gasPrice: %v", gasPrice)
	log.Printf("✔️ lastBlock: %v", lastBlock)
	log.Printf("✔️ txArr: %v", txArr)

	hash, err := f.SendBundle(txArr, lastBlock)
	if err != nil {
		log.Fatalf("❌ SendBundle 失败: %v", err)
	}
	log.Printf("✔️ hash: %v", hash)
}

func (f ItmFlashBot) Client() *w3.Client {
	client := w3.MustDial(f.ETHRpcUrl)
	defer client.Close()
	return client
}

// 获取必要参数
func (f ItmFlashBot) GetParams() (uint64, *big.Int, *big.Int) {
	var (
		nonce       uint64
		gasPrice    *big.Int
		latestBlock *big.Int
	)
	if err := f.Client().Call(
		eth.Nonce(f.Sender, nil).Returns(&nonce),
		eth.GasPrice().Returns(&gasPrice),
		eth.BlockNumber().Returns(&latestBlock),
	); err != nil {
		log.Fatalf("❌ params 获取失败: %v", err)
	}
	return nonce, gasPrice, latestBlock
}

// 预先签名交易
func (f ItmFlashBot) GetTxsArr(nonce uint64, gasPrice *big.Int) []*types.Transaction {
	amount := big.NewInt(1)
	txArr := types.Transactions{
		f.CreateSignedTx(amount, nonce, gasPrice),
		//f.CreateSignedTx(amount, nonce+1, gasPrice),
	}
	return txArr
}

// 签名交易详情
func (f ItmFlashBot) CreateSignedTx(amount *big.Int, nonce uint64, gasPrice *big.Int) *types.Transaction {
	data, err := f.ContractABI.Pack("presale", amount)
	if err != nil {
		log.Fatalf("❌ ContractABI.Pack Error: %v", err)
	}
	fmt.Println("✔️ 签名交易数据：", hex.EncodeToString(data))
	tx := &types.LegacyTx{
		Nonce:    nonce,
		To:       &f.ContractAddress,
		Value:    big.NewInt(1e16),
		Gas:      250_000,
		GasPrice: gasPrice,
		Data:     data,
	}
	s := types.LatestSigner(params.SepoliaChainConfig)
	//signedTx, err := types.SignNewTx(f.PrivateKey, types.NewEIP155Signer(f.ChainID), tx)
	signedTx, err := types.SignNewTx(f.PrivateKey, s, tx)
	if err != nil {
		log.Fatalf("❌ types.SignNewTx Error: %v", err)
	}
	log.Println("✔️ 签名交易HASH：", signedTx.Hash().Hex())
	return signedTx
}

// 发送交易
func (f ItmFlashBot) SendBundle(bundle types.Transactions, latestBlock *big.Int) (common.Hash, error) {
	var (
		client     = flashbots.MustDial(f.FlashBotsRpcUrl, f.PrivateKey)
		bundleHash common.Hash
		err        error
	)
	defer client.Close()

	if f.Debug {
		var callBundle *flashbots.CallBundleResponse
		err = client.Call(flashbots.CallBundle(&flashbots.CallBundleRequest{
			Transactions: bundle,
			BlockNumber:  new(big.Int).Add(latestBlock, w3.Big1),
		}).Returns(&callBundle))
		log.Printf("✔️ CallBundleResponse: %v", callBundle)
	} else {
		err = client.Call(flashbots.SendBundle(&flashbots.SendBundleRequest{
			Transactions: bundle,
			BlockNumber:  new(big.Int).Add(latestBlock, w3.Big1),
		}).Returns(&bundleHash))
	}
	return bundleHash, err
}
