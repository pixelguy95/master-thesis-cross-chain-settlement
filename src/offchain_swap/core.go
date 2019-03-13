package main

import (
	"fmt"

	rpcutils "github.com/pixelguy95/btcd-rpcclient-extension/bitcoin"

	"./channel"
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
	fmt.Println("Start up")

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

	clientWraper.GetNewP2PKHAddress()

	cj, _ := channel.GenerateNewUserFromWallet("cj_wallet", client)
	cj.PrintUser()

	fmt.Println()

	other, _ := channel.GenerateNewUserFromWallet("otherwallet", client)
	other.PrintUser()

	fmt.Println()
	fmt.Println()

	cj.UserBalance = 100000

	pc, error := channel.OpenNewChannel(cj, other, client)

	if error != nil {
		fmt.Println(error)
	}

	fmt.Println(pc.FundingTx)
}
