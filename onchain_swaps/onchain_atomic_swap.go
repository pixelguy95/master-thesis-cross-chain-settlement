package main

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/txscript"

	"github.com/btcsuite/btcd/chaincfg"

	"github.com/btcsuite/btcutil"

	"./customtransactions"
	"./rpcutils"
	"github.com/btcsuite/btcd/rpcclient"
)

var connCfg = &rpcclient.ConnConfig{
	Host:         "localhost:18332",
	HTTPPostMode: true,
	DisableTLS:   true,
	User:         "pi",
	Pass:         "kebab",
}

func main() {

	ntfnHandlers := rpcclient.NotificationHandlers{
		OnClientConnected: func() {
			fmt.Println("Connected")
		},
	}

	client, error := rpcclient.New(connCfg, &ntfnHandlers)
	if error != nil {
		fmt.Println(error)
	}

	client.Connect(1)
	clientWraper := rpcutils.New(client)

	clientWraper.UnloadAllWallets()
	clientWraper.LoadWallet("cj_wallet")

	receiver, _ := btcutil.DecodeAddress("mm6Xib7iCSuFffgEgQxHtr7idbs4PqzbCw", &chaincfg.TestNet3Params)
	contractDetails, _ := customtransactions.GenerateAtomicSwapContract(receiver, 100000, client)

	fmt.Println("Contract transaction:")
	buf := new(bytes.Buffer)
	contractDetails.Contract.SerializeNoWitness(buf)
	fmt.Println(hex.EncodeToString(buf.Bytes()))

	fmt.Println("Refund transaction:")
	buf = new(bytes.Buffer)
	contractDetails.Refund.SerializeNoWitness(buf)
	fmt.Println(hex.EncodeToString(buf.Bytes()))

	stuff, _ := txscript.ExtractAtomicSwapDataPushes(0, contractDetails.Contract.TxOut[1].PkScript)
	fmt.Println(stuff)

	//reciverAsBytes, _ := hex.DecodeString("0349c9b0c947ea937858106c2f1ac465d456dba7239875eac6bb78ee32d574c607")
	//reciver, _ := btcutil.NewAddressPubKey(reciverAsBytes, &chaincfg.TestNet3Params)
	//swap, error := customtransactions.GenerateAtomicSwapTransaction(reciver, 100000, client)

	//reciverAsBytes, _ := hex.DecodeString("0349c9b0c947ea937858106c2f1ac465d456dba7239875eac6bb78ee32d574c607")
	//bkey, _ := btcutil.NewAddressPubKey(reciverAsBytes, &chaincfg.TestNet3Params)

	//fmt.Println(bkey.EncodeAddress())

	//out := customtransactions.GenerateInputIndex("48d077ab5e98b29a3a54ffcc59e9f1ebc9ff802f5c9ce68efb3e626af1b00e6a", 1)
	//tx, _ := customtransactions.GenerateAtomicClaimWithSecret(out, 100000, client)

	client.Disconnect()
}
