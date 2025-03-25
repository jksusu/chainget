package chainget

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
)

type EventSubClient struct {
	Client        *ethclient.Client
	Url           string
	Ctx           context.Context
	SubscribeChan chan common.Hash
}

func NewEventSubClient(wssUrl string) *EventSubClient {
	c, err := ethclient.Dial(wssUrl)
	if err != nil {
		log.Fatalf("event sub client error: %v", err)
	}
	client := &EventSubClient{
		Client:        c,
		Url:           wssUrl,
		Ctx:           context.Background(),
		SubscribeChan: make(chan common.Hash), //订阅事件通知
	}
	//读数据
	go client.Loop()

	return client
}

func (c *EventSubClient) Loop() {
	for {
		select {
		case txHash := <-c.SubscribeChan:
			fmt.Printf("New pending transaction: %s\n", txHash.Hex())
			tx, isPending, err := c.Client.TransactionByHash(context.Background(), txHash)
			if err != nil {
				log.Printf("Failed to fetch transaction %s: %v", txHash.Hex(), err)
				continue
			}
			if !isPending {
				fmt.Printf("Transaction %s is no longer pending\n", txHash.Hex())
				continue
			}
			fmt.Printf("From: %s\n", getTransactionSender(tx))
			fmt.Printf("To: %s\n", tx.To().Hex())
			fmt.Printf("Value: %s\n", tx.Value().String())
			fmt.Printf("Gas: %d\n", tx.Gas())
			fmt.Printf("GasPrice: %s\n", tx.GasPrice().String())
			fmt.Printf("Data: %x\n", string(tx.Data()))
		}
	}
}

func (c *EventSubClient) SubNewPendingTransactions() {
	sub, err := c.Client.Client().Subscribe(c.Ctx, "eth", c.SubscribeChan, "newPendingTransactions")
	if err != nil {
		log.Fatalf("Failed to subscribe to newPendingTransactions: %v", err)
	}
	fmt.Println("Subscribed to newPendingTransactions success")
	defer sub.Unsubscribe()
}
