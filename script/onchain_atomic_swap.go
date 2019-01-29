package main

import (
	"fmt"
	"log"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil"
)

func main() {
	//unspent := basictransactions.GenerateInputIndex("27b2a5950bd67a96fcdb4c9dcc35e5af4bdcf215fed325dba86f804101ab5646", 0)

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
		OnAccountBalance: func(account string, balance btcutil.Amount, confirmed bool) {
			log.Printf("New balance for account %s: %v", account,
				balance)
		},
	}

	client, error := rpcclient.New(connCfg, &ntfnHandlers)
	fmt.Println(error)

	client.Connect(3)

	client.
}
