package main

import (
	"bytes"
	"fmt"

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

	cj, _ := channel.GenerateNewUserFromWallet("cj_wallet", true, client)
	cj.PrintUser()

	fmt.Println()

	other, _ := channel.GenerateNewUserFromWallet("otherwallet", false, client)
	other.PrintUser()

	fmt.Println()
	fmt.Println()

	cj.UserBalance = 100000

	pc, error := channel.OpenNewChannel(cj, other, client)

	if error != nil {
		fmt.Println(error)
	}

	fmt.Println()
	buf := new(bytes.Buffer)
	pc.FundingTx.Serialize(buf)
	fmt.Printf("FUNDING TX:\n%x\n\n", buf)

	fmt.Println()
	buf = new(bytes.Buffer)
	pc.Party1.Commits[0].CommitTx.Serialize(buf)
	fmt.Printf("COMMIT TX:\n%x\n\n", buf)

	fmt.Println()
	buf = new(bytes.Buffer)
	pc.Party1.CommitSpends[0].CommitSpend.Serialize(buf)
	fmt.Printf("SPEND COMMIT TX:\n%x\n\n", buf)
}
