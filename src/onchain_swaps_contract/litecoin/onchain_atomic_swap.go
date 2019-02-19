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
	contract, _ := hex.DecodeString("6382012088a820a2303419b90d0aeec2bb4847bba4456a2bdaee182849b03e1236b2872f7291b58876a914b221e68c4e811390c1d491b0592fd99b2d2a253467045f036d5cb17576a9143a7afa827dce4bb67428336c75f1af4361f6d6436888ac")
	//0220e192922b445359ccee222d0e9ddc664e519e6cfbf2582725c1beacdd10df

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
