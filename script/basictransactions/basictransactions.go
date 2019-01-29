package basictransactions

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"golang.org/x/crypto/ripemd160"
)

// DefaultFee is the default fee that will be payed in a transaction
const DefaultFee uint64 = 300

// GenerateInputIndex creates a new input pointer, points to the transaction you want to fund a new transaction with
func GenerateInputIndex(txid string, outputIndex uint32) *wire.OutPoint {
	prevHash, _ := hex.DecodeString(txid)
	prevHash = reverse(prevHash)
	prevTxHash, _ := chainhash.NewHash(prevHash)
	prevTxHash.SetBytes(prevHash)

	return wire.NewOutPoint(prevTxHash, outputIndex)
}

// GenerateP2PKHTransaction creates a new P2PKH transaction with some parameters
func GenerateP2PKHTransaction(payToKey string, changeKey string, fundFrom *wire.OutPoint, amount uint64, myAmount uint64) string {
	tx := wire.NewMsgTx(2)

	payToKeyBytes, _ := hex.DecodeString(payToKey)
	changeKeyBytes, _ := hex.DecodeString(changeKey)

	tx.AddTxIn(wire.NewTxIn(fundFrom, nil, nil))
	tx.AddTxOut(wire.NewTxOut(int64(amount), createP2PKHScript(payToKeyBytes)))

	change := myAmount - amount - DefaultFee
	tx.AddTxOut(wire.NewTxOut(int64(change), createP2PKHScript(changeKeyBytes)))

	buf := new(bytes.Buffer)
	tx.SerializeNoWitness(buf)
	return hex.EncodeToString(buf.Bytes())
}

// createP2PKH creates a basic P2PKH transaction
// Input should be the public key you want to send too
// You can get the the raw publickey from an address with:
// bitcoind getaddressinfo [address]
func createP2PKHScript(address []byte) []byte {

	// Creates the hash160 of the given public key
	shaHash := sha256.Sum256(address)
	h := ripemd160.New()
	h.Write(shaHash[:])
	hash160 := h.Sum(nil)

	builder := txscript.NewScriptBuilder()

	builder.AddOp(txscript.OP_DUP)
	builder.AddOp(txscript.OP_HASH160)
	builder.AddData(hash160)
	builder.AddOp(txscript.OP_EQUALVERIFY)
	builder.AddOp(txscript.OP_CHECKSIG)

	script, _ := builder.Script()
	return script
}

func reverse(numbers []byte) []byte {
	newNumbers := make([]byte, len(numbers))
	for i, j := 0, len(numbers)-1; i < j; i, j = i+1, j-1 {
		newNumbers[i], newNumbers[j] = numbers[j], numbers[i]
	}
	return newNumbers
}
