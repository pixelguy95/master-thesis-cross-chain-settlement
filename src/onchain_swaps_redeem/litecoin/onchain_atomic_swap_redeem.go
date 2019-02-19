package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"

	"github.com/ltcsuite/ltcd/chaincfg"
	"github.com/ltcsuite/ltcd/rpcclient"
	"github.com/ltcsuite/ltcd/txscript"
	"github.com/ltcsuite/ltcd/wire"
	"github.com/ltcsuite/ltcutil"

	"../../onchain_swaps_contract/litecoin/customtransactions"
)

var connCfg = &rpcclient.ConnConfig{
	Host:         "localhost:19332",
	HTTPPostMode: true,
	DisableTLS:   true,
	User:         "pi",
	Pass:         "kebab",
}

func main() {

	client, error := rpcclient.New(connCfg, nil)
	if error != nil {
		fmt.Println(error)
	}

	client.Connect(1)

	txid, _ := hex.DecodeString(os.Args[1])
	outindex, _ := strconv.Atoi(os.Args[2])
	contract, _ := hex.DecodeString(os.Args[3])
	secret, _ := hex.DecodeString(os.Args[4])

	contractOutpoint := customtransactions.GenerateInputIndex(os.Args[1], uint32(outindex))

	fmt.Printf("%x\n", txid)

	//Fetch tx
	hash := contractOutpoint.Hash
	contractTx, error := client.GetRawTransaction(&hash)

	if error != nil {
		fmt.Println(error)
		return
	}

	output := contractTx.MsgTx().TxOut[outindex]

	fmt.Println(output.Value)
	stuff, _ := txscript.ExtractAtomicSwapDataPushes(0, contract)
	recipient, _ := ltcutil.NewAddressPubKeyHash(stuff.RecipientHash160[:], &chaincfg.TestNet4Params)

	//Form redeem transaction
	redeemTx := wire.NewMsgTx(2)
	redeemTx.AddTxIn(wire.NewTxIn(contractOutpoint, nil, nil))
	redeemTx.AddTxOut(wire.NewTxOut(output.Value-int64(customtransactions.DefaultFee), customtransactions.CreateP2PKHScript(recipient.ScriptAddress())))

	// Get redeem address private key
	redeemPriv, error := client.DumpPrivKey(recipient)

	sig, pub, error := customtransactions.CreateSignature(redeemTx, 0, contract, redeemPriv)

	redeemScript := createRedeemScript(sig, pub, secret, contract)
	redeemTx.TxIn[0].SignatureScript = redeemScript

	buf := new(bytes.Buffer)
	redeemTx.SerializeNoWitness(buf)
	fmt.Printf("\n%x\n", buf.Bytes())

	client.Disconnect()
}

func createRedeemScript(sig []byte, pubkey []byte, secret []byte, contract []byte) []byte {

	builder := txscript.NewScriptBuilder()

	builder.AddData(sig)
	builder.AddData(pubkey)
	builder.AddData(secret)
	builder.AddInt64(1)
	builder.AddData(contract)

	script, _ := builder.Script()
	return script
}
