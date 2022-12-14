package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

var NodeEndpoint = ""

func init() {
	NodeEndpoint = os.Getenv("NODE_ENDPOINT")
}

func ethClient() *ethclient.Client {
	ethClient, err := ethclient.Dial(NodeEndpoint)
	if err != nil {
		panic(err)
	}
	return ethClient
}

var Routers = [3]common.Address{
	common.HexToAddress("0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D"), // uniswap v2
	common.HexToAddress("0x8ad599c3A0ff1De082011EFDDc58f1908eb6e6D8"), // uniswap v3
	common.HexToAddress("0xd9e1cE17f2641f24aE83637ab66a2cca9C378B9F"), // sushiswap
}

func isRouter(address *common.Address) bool {
	for _, router := range Routers {
		if router == *address {
			return true
		}
	}
	return false
}

func pendingTransactionsChannel() chan common.Hash {
	baseClient, err := rpc.Dial(NodeEndpoint)
	if err != nil {
		log.Fatalln(err)
	}

	txnsHash := make(chan common.Hash)

	subscriber := gethclient.New(baseClient)
	_, err = subscriber.SubscribePendingTransactions(context.Background(), txnsHash)
	if err != nil {
		log.Fatalln(err)
	}

	return txnsHash
}

func handleRouter(txn *types.Transaction) {
	fmt.Println("Router:", txn.Hash())
}

func main() {
	ethClient := ethClient()
	txnsHash := pendingTransactionsChannel()

	defer func() {
		ethClient.Close()
	}()

	for txnHash := range txnsHash {
		txn, _, err := ethClient.TransactionByHash(context.Background(), txnHash)
		if err != nil {
			continue
		}

		if isRouter(txn.To()) {
			handleRouter(txn)
		}
	}
}
