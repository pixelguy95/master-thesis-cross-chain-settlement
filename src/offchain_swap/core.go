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

	cj, _ := channel.GenerateNewUserFromWallet("Alice", "cj_wallet", true, client)
	cj.PrintUser()

	fmt.Println()

	other, _ := channel.GenerateNewUserFromWallet("Bob", "otherwallet", false, client)
	other.PrintUser()

	fmt.Println()
	fmt.Println()

	cj.UserBalance = 100000

	pc, error := channel.OpenNewChannel(cj, other, client)

	if error != nil {
		fmt.Println(error)
	}

	sd := &channel.SendDescriptor{
		Balance:  15000,
		Sender:   pc.Party1,
		Receiver: pc.Party2,
	}

	err := pc.SendCommit(sd)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println()
	buf := new(bytes.Buffer)
	pc.FundingTx.Serialize(buf)
	fmt.Printf("FUNDING TX:\n%x\n\n", buf)

	fmt.Println()
	buf = new(bytes.Buffer)
	pc.Party2.Commits[1].CommitTx.Serialize(buf)
	fmt.Printf("RECEIVER COMMIT TX:\n%x\n\n", buf)

	fmt.Println()
	buf = new(bytes.Buffer)
	pc.Party1.HTLCOutputTxs[1].ReceiverCommitTimeoutTx.Serialize(buf)
	fmt.Printf("RECEIVER COMMIT TIMEOUT TX:\n%x\n\n", buf)

	fmt.Println()
	buf = new(bytes.Buffer)
	pc.Party1.HTLCOutputTxs[1].ReceiverCommitTimeoutRedeemTx.Serialize(buf)
	fmt.Printf("RECEIVER COMMIT TIMEOUT REDEEM TX:\n%x\n\n", buf)

	fmt.Println()
	buf = new(bytes.Buffer)
	pc.Party2.HTLCOutputTxs[1].ReceiverCommitTimeoutRevokeTx.Serialize(buf)
	fmt.Printf("RECEIVER COMMIT TIMEOUT REVOKE TX:\n%x\n\n", buf)
}
