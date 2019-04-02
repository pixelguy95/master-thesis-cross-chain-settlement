package main

import (
	"log"

	"github.com/btcsuite/btcd/rpcclient"
	ltcrpc "github.com/ltcsuite/ltcd/rpcclient"
	"github.com/pixelguy95/master-thesis-cross-chain-settlement/src/offchain_swap/channel"
	"github.com/pixelguy95/master-thesis-cross-chain-settlement/src/offchain_swap/swap"
)

var connCfg = &rpcclient.ConnConfig{
	Host:         "localhost:18332",
	HTTPPostMode: true,
	DisableTLS:   true,
	User:         "pi",
	Pass:         "kebab",
}

var ltcConnCfg = &ltcrpc.ConnConfig{
	Host:         "localhost:19332",
	HTTPPostMode: true,
	DisableTLS:   true,
	User:         "pi",
	Pass:         "kebab",
}

func main() {
	log.Println("Start up")

	log.Println("Connecting to bitcoin node...")
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Connecting to litecoin node...")
	ltcClient, err := ltcrpc.New(ltcConnCfg, nil)
	if err != nil {
		log.Fatal(err)
	}

	client.Connect(1)
	ltcClient.Connect(1)

	log.Println("Creating Alice user for bitcoin channel")
	alice, _ := channel.GenerateNewUserFromWallet("Alice", "cj_wallet", true, false, client, ltcClient)
	alice.UserBalance = 100000

	log.Println("Creating Bob user for bitcoin channel")
	bob, _ := channel.GenerateNewUserFromWallet("Bob", "otherwallet", false, false, client, ltcClient)

	log.Print("Opening channel between Alice and Bob (bitcoin)")
	pc, err := channel.OpenNewChannel(alice, bob, false, client, ltcClient)
	if err != nil {
		log.Fatal(err)
	}
	pc.Settle(client)

	log.Println("Creating Bob user for litecoin channel")
	bobLtc, _ := channel.GenerateNewUserFromWallet("Bob", "doesn't matter either", true, true, client, ltcClient)
	bobLtc.UserBalance = 100000

	log.Println("Creating Alice user for litecoin channel")
	aliceLtc, _ := channel.GenerateNewUserFromWallet("Alice", "doesn't matter", false, true, client, ltcClient)

	log.Print("Opening channel between Alice and Bob (litecoin)")
	pcLtc, err := channel.OpenNewChannel(bobLtc, aliceLtc, true, client, ltcClient)

	//sd := &channel.SendDescriptor{
	//	Balance:  15000,
	//	Sender:   pc.Party1,
	//	Receiver: pc.Party2,
	//}

	//err = pc.SendCommit(sd)
	//if err != nil {
	//	log.Fatal(err)
	//	return
	//}

	//bob.UserBalance += 15000
	//pc.Settle(client)

	//buf := new(bytes.Buffer)
	//pc.FundingTx.Serialize(buf)
	//fmt.Printf("FUNDING TX:\n%x\n\n", buf)

	//buf = new(bytes.Buffer)
	//pc.Party1.Commits[2].CommitTx.Serialize(buf)
	//fmt.Printf("Commit TX:\n%x\n\n", buf)

	//sd := &channel.SendDescriptor{
	//	Balance:  15000,
	//	Sender:   pcLtc.Party1,
	//	Receiver: pcLtc.Party2,
	//}

	//err = pcLtc.SendCommit(sd)
	//if err != nil {
	//	log.Fatal(err)
	//}

	swap.GenerateAtomicSwap(pc, pcLtc, 20000)
}
