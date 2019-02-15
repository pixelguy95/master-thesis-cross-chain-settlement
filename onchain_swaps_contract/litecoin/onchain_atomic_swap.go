package main

import (
	"bytes"
	"encoding/hex"
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

	receiver, _ := ltcutil.DecodeAddress("msAAFBn8WD8T4aFsqLKe8LpEA5DaT7Wzwg", &chaincfg.TestNet4Params)
	contract, _ := hex.DecodeString("6382012088a8208ecae302621c1b9f73e49d076a3b52d59e9b1fd553ac15b9b46773e4aa25bc4d8876a914b221e68c4e811390c1d491b0592fd99b2d2a253467047915685cb17576a914d749fe1d9c31aa51d6babecaa32ae3c568952c346888ac")

	contractDetails, _ := customtransactions.GenerateAtomicSwapFromContract(receiver, 100000, contract, client)

	//fmt.Printf("Secret hex: %x\n\n", contractDetails.Secret)

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
