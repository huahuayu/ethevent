package client

import (
	"context"
	"errors"
	"ethevent/http"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"math/big"
	"strings"
	"time"
)

var (
	EthClient   *ethclient.Client
	abiApi      = "https://api.etherscan.io/api?module=contract&action=getabi&address=$address&apikey=$apikey"
	contractAbi string
)

const (
	usdtContract  = "0xdac17f958d2ee523a2206206994597c13d831ec7"
	apikey        = "UYD2G767RI7EGMW7S22XMQCSXAC9XIDM1I"
	transferTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
)

func Init() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	var err error
	//EthClient, err = ethclient.DialContext(ctx, "https://mainnet.infura.io/v3/e9d43fcc8b60466c9b8c6c5b8215475c")
	EthClient, err = ethclient.DialContext(ctx, "wss://mainnet.infura.io/ws/v3/e9d43fcc8b60466c9b8c6c5b8215475c")
	defer cancel()
	if err != nil {
		log.Fatal(err)
	}
}

func FilterLog() {
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(11324683),
		ToBlock:   big.NewInt(11324783),
		Addresses: []common.Address{
			common.HexToAddress(usdtContract),
		},
		Topics: [][]common.Hash{{common.HexToHash(transferTopic)}, nil},
	}

	if contractAbi == ""{
		abiStr, err := getAbiFromEtherscan()
		if err != nil{
			log.Fatal(err)
		}
		contractAbi = abiStr
	}

	poolAbi, err := abi.JSON(strings.NewReader(contractAbi))
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	eventLogs, err := EthClient.FilterLogs(ctx, query)
	if err != nil {
		log.Fatal(err)
	}

	for _, eventLog := range eventLogs {
		if eventLog.Removed {
			continue
		}
		var transferEvent struct {
			From  common.Address
			To    common.Address
			Value *big.Int
		}
		err := poolAbi.UnpackIntoInterface(&transferEvent, "Transfer", eventLog.Data)
		if err != nil {
			log.Error("Failed to unpack")
			continue
		}

		log.Println("txid:", eventLog.TxHash.Hex())
		log.Println("from:", common.BytesToAddress(eventLog.Topics[1].Bytes()).Hex())
		log.Println("to:", common.BytesToAddress(eventLog.Topics[2].Bytes()).Hex())
		log.Println("value:", transferEvent.Value)
	}
}

func SubEvent() {
	type transferEvent struct {
		From  common.Address
		To    common.Address
		Value *big.Int
	}

	contractAddress := common.HexToAddress(usdtContract)

	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
	}

	var ch = make(chan types.Log)
	ctx := context.Background()

	sub, err := EthClient.SubscribeFilterLogs(ctx, query, ch)
	if err != nil {
		log.Fatal(err)
	}

	tokenAbi, err := abi.JSON(strings.NewReader("[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"constant\":true,\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"))
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case err := <-sub.Err():
			log.Fatal(err)
		case eventLog := <-ch:
			event := new(transferEvent)
			err := tokenAbi.UnpackIntoInterface(event, "Transfer", eventLog.Data)
			if err != nil {
				log.Println("Failed to unpack")
				continue
			}
			from := common.BytesToAddress(eventLog.Topics[1].Bytes())
			to := common.BytesToAddress(eventLog.Topics[2].Bytes())
			//log.Println("From", event.From.Hex())
			//log.Println("To", event.To.Hex())
			log.Println("From", from.Hex())
			log.Println("To", to.Hex())
			log.Println("Value", event.Value)
		}
	}
}

func getAbiFromEtherscan() (abiStr string, err error) {
	tmp := strings.Replace(abiApi, "$address", usdtContract, 1)
	url := strings.Replace(tmp, "$apikey", apikey, 1)
	var abiRes struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  string `json:"result"`
	}
	err = http.Get(url, &abiRes)
	if err != nil {
		log.Error(err)
		return "",err
	}
	if abiRes.Message != "OK" {
		log.Error("get abi failed: ",abiRes.Result)
		return "",errors.New(abiRes.Result)
	}
	return abiRes.Result, nil
}
