package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

var NodeEndpoint = ""
var eth *ethclient.Client

func ethClient() *ethclient.Client {
	ethClient, err := ethclient.Dial(NodeEndpoint)
	if err != nil {
		panic(err)
	}
	return ethClient
}

func init() {
	NodeEndpoint = os.Getenv("NODE_ENDPOINT")
	eth = ethClient()
}

var Routers = [3]common.Address{
	common.HexToAddress("0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D"), // uniswap v2
	// common.HexToAddress("0x8ad599c3A0ff1De082011EFDDc58f1908eb6e6D8"), // uniswap v3
	common.HexToAddress("0xd9e1cE17f2641f24aE83637ab66a2cca9C378B9F"), // sushiswap
}

func isRouter(address *common.Address) bool {
	if address == nil {
		return false
	}
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

func GetLocalABI(path string) string {
	abiFile, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer abiFile.Close()

	result, err := io.ReadAll(abiFile)
	if err != nil {
		log.Fatal(err)
	}
	return string(result)
}

func DecodeTransactionInputData(contractABI *abi.ABI, data []byte) {
	// The first 4 bytes of the t represent the ID of the method in the ABI
	// https://docs.soliditylang.org/en/v0.5.3/abi-spec.html#function-selector
	methodSigData := data[:4]
	method, err := contractABI.MethodById(methodSigData)
	if err != nil {
		log.Fatal(err)
	}

	inputsSigData := data[4:]
	inputsMap := make(map[string]interface {
	})
	if err := method.Inputs.UnpackIntoMap(inputsMap, inputsSigData); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Method Name: %s\n", method.Name)
	inputs, _ := json.Marshal(inputsMap)
	fmt.Printf("Method inputs: %v\n", string(inputs))
}

func handleRouter(txn *types.Transaction) {
	fmt.Println("Router:", txn.To())
	fmt.Println("tx hash:", txn.Hash())

	contractABI, err := abi.JSON(strings.NewReader(GetLocalABI("abis/UniswapV2Router.json")))
	if err != nil {
		log.Fatal(err)
	}

	DecodeTransactionInputData(&contractABI, txn.Data())
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
			go handleRouter(txn)
		}
	}
}
