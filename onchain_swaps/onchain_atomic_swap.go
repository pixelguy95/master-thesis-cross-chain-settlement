package main

import (
	"bytes"
	"fmt"

	"encoding/hex"

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

	clientWraper.GetNewPubKey()
	return

	clientWraper.UnloadAllWallets()
	clientWraper.LoadWallet("cj_wallet")

	tx, error := customtransactions.GenerateP2PKHTransaction("mhqovpK2dcjUvdmn75ZgUcMi5FCmviVHmb", 10000, client)
	if error != nil {
		fmt.Println(error)
	}

	//clientWraper.SignRawTransactionWithWallet(tx)

	//client.DumpPrivKey()

	buf := new(bytes.Buffer)
	tx.SerializeNoWitness(buf)
	fmt.Println(hex.EncodeToString(buf.Bytes()))

	client.Disconnect()
}
