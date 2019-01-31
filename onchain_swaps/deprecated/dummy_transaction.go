package main

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// midXZ3QgeJcStWw6pCJAbqJSacPzJtSVg8 -> 2226a2c7388704c65ee6edb0148dd272f9c5e2d3

func claimAdditionTransaction() {
	tx := wire.NewMsgTx(2)

	prevHash, _ := hex.DecodeString("05345b91bc58a8274986ecaa18c5c73c5355fa4d91858e27e831af97593c6b7e")
	prevHash = reverse(prevHash)
	prevTxHash, _ := chainhash.NewHash(prevHash)
	prevTxHash.SetBytes(prevHash)

	outPoint := wire.NewOutPoint(prevTxHash, 0)

	tx.AddTxIn(wire.NewTxIn(outPoint, createDumbClaimScript(), nil))

	pubKeyHash, _ := hex.DecodeString("2226a2c7388704c65ee6edb0148dd272f9c5e2d3")
	tx.AddTxOut(wire.NewTxOut(int64(12848000), createP2PKH(pubKeyHash)))

	buf := new(bytes.Buffer)
	tx.SerializeNoWitness(buf)
	fmt.Println(hex.EncodeToString(buf.Bytes()))
}

func generateAdditionTransaction() {
	tx := wire.NewMsgTx(2)

	prevHash, _ := hex.DecodeString("b8326e9f74fc444e87572623b144d7b8c4acbf9a710ce67038b48e3ce62c92dc")
	prevHash = reverse(prevHash)
	prevTxHash, _ := chainhash.NewHash(prevHash)
	prevTxHash.SetBytes(prevHash)

	outPoint := wire.NewOutPoint(prevTxHash, 0)

	tx.AddTxIn(wire.NewTxIn(outPoint, nil, nil))
	tx.AddTxOut(wire.NewTxOut(int64(12849000), createDumbScript()))

	buf := new(bytes.Buffer)
	tx.SerializeNoWitness(buf)
	fmt.Println(hex.EncodeToString(buf.Bytes()))
}

func reverse(numbers []byte) []byte {
	newNumbers := make([]byte, len(numbers))
	for i, j := 0, len(numbers)-1; i < j; i, j = i+1, j-1 {
		newNumbers[i], newNumbers[j] = numbers[j], numbers[i]
	}
	return newNumbers
}

func createDumbScript() []byte {
	builder := txscript.NewScriptBuilder()

	builder.AddOp(txscript.OP_1)
	builder.AddOp(txscript.OP_ADD)
	builder.AddOp(txscript.OP_2)
	builder.AddOp(txscript.OP_NUMEQUALVERIFY)

	script, _ := builder.Script()
	return script
}

func createDumbClaimScript() []byte {
	builder := txscript.NewScriptBuilder()

	builder.AddOp(txscript.OP_1)
	builder.AddOp(txscript.OP_1)

	script, _ := builder.Script()
	return script
}

func createP2PKH(address []byte) []byte {
	builder := txscript.NewScriptBuilder()

	builder.AddOp(txscript.OP_DUP)
	builder.AddOp(txscript.OP_HASH160)
	builder.AddData(address)
	builder.AddOp(txscript.OP_EQUALVERIFY)
	builder.AddOp(txscript.OP_CHECKSIG)

	script, _ := builder.Script()
	return script
}
