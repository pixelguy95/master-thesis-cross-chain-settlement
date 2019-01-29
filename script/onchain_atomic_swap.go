package main

import (
	"fmt"
	"log"

	"encoding/json"

	"./rpcutils"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil"
)

func main() {

	//02c8fc9f6644e56705bd8bb7d287482968e3da0dede91caa45243c4ee378eacca5
	//hexTransaction := basictransactions.GenerateP2PKHTransaction("0327ef5006745e8c135911b86ad42e73170ecdb4e5793949362a580eee59885c03", "02c8fc9f6644e56705bd8bb7d287482968e3da0dede91caa45243c4ee378eacca5", unspent, 5424000, 12848000)
	//fmt.Println(hexTransaction)

	connCfg := &rpcclient.ConnConfig{
		Host:         "localhost:18332",
		HTTPPostMode: true,
		DisableTLS:   true,
		User:         "pi",
		Pass:         "kebab",
	}

	ntfnHandlers := rpcclient.NotificationHandlers{
		OnClientConnected: func() {
			fmt.Println("Connected")
		},
		OnAccountBalance: func(account string, balance btcutil.Amount, confirmed bool) {
			log.Printf("New balance for account %s: %v", account,
				balance)
		},
		OnUnknownNotification: func(method string, params []json.RawMessage) {
			fmt.Println(method)
		},
	}

	client, error := rpcclient.New(connCfg, &ntfnHandlers)
	if error != nil {
		fmt.Println(error)
	}

	client.Connect(1)
	clientWraper := rpcutils.New(client)
	rawData, error := clientWraper.ListWallets()

	if error != nil {
		fmt.Println(error)
	}

	fmt.Println(rawData)

	error = clientWraper.UnloadAllWallets()
	if error != nil {
		fmt.Println(error)
	} else {
		fmt.Println("All wallets deleted")
	}

	newWalletMsg, error := clientWraper.CreateWallet("brandnew")
	if error != nil {
		fmt.Println(error)
	} else {
		fmt.Println(newWalletMsg)
	}

	client.Disconnect()
}
