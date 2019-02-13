package main

import (
	"bytes"
	"fmt"

	"./customtransactions"
	"github.com/ltcsuite/ltcd/chaincfg"
	"github.com/ltcsuite/ltcd/rpcclient"
	"github.com/ltcsuite/ltcutil"
)

func main() {

	connCfg := &rpcclient.ConnConfig{
		Host:         "localhost:19332",
		HTTPPostMode: true,
		DisableTLS:   true,
		User:         "pi",
		Pass:         "kebab",
	}

	client, error := rpcclient.New(connCfg, nil)
	if error != nil {
		fmt.Println(error)
	}

	fmt.Println(client.GetBlockCount())

	receiver, _ := ltcutil.DecodeAddress("mpkEXru48AA3k5Z6zU1hmUrzra1WwzHuj3", &chaincfg.TestNet4Params)
	contractDetails, _ := customtransactions.GenerateNewAtomicSwapContract(receiver, 100000, client)

	fmt.Printf("Secret hex: %x\n\n", contractDetails.Secret)

	fmt.Println("Contract:")
	fmt.Printf("%x\n\n", contractDetails.Contract)

	fmt.Println("Contract transaction:")
	buf := new(bytes.Buffer)
	contractDetails.ContractTx.SerializeNoWitness(buf)
	fmt.Printf("%x\n\n", buf)

	fmt.Println("Refund transaction:")
	buf = new(bytes.Buffer)
	contractDetails.Refund.SerializeNoWitness(buf)
	fmt.Printf("%x\n\n", buf)
}
