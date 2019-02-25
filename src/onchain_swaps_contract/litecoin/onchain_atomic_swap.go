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

	receiver, _ := ltcutil.DecodeAddress("mz3n1EDLsV7ofS2b2GRAFCfwipMCb8Fea9", &chaincfg.TestNet4Params)
	contract, _ := hex.DecodeString("6382012088a82044e4e1ea490adba0ae309fbc40b8f1ec0c35e0bea0359b646377ca9b0bc5d9898876a914c68153c7f59420ac02589ad32c4cf3e3b02124336704e81a715cb17576a914db91c90d16b48d5090f87af6c54ce9c6844c031e6888ac")
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
