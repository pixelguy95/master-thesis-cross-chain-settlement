package customtransactions

import (
	"encoding/hex"
	"fmt"

	"../rpcutils"
	"github.com/btcsuite/btcd/rpcclient"

	"github.com/btcsuite/btcutil"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// DefaultFee is the default fee that will be payed in a transaction
const DefaultFee uint64 = 500

// GenerateP2PKHTransaction creates a new P2PKH transaction with some parameters
func GenerateP2PKHTransaction(payTo string, amount uint64, client *rpcclient.Client) (*wire.MsgTx, error) {

	clientWraper := rpcutils.New(client)
	changeTo := clientWraper.GetNewP2PKHAddress()

	tx := wire.NewMsgTx(2)

	payToAddress, _ := btcutil.DecodeAddress(payTo, &chaincfg.TestNet3Params)
	changeToAddress, _ := btcutil.DecodeAddress(changeTo, &chaincfg.TestNet3Params)

	sum, _ := gatherFunds(tx, client)

	// Output to reciver
	tx.AddTxOut(wire.NewTxOut(int64(amount), createP2PKHScript(payToAddress.ScriptAddress())))

	// Change output
	change := sum - amount - DefaultFee
	tx.AddTxOut(wire.NewTxOut(int64(change), createP2PKHScript(changeToAddress.ScriptAddress())))

	return tx, nil
}

func gatherFunds(tx *wire.MsgTx, client *rpcclient.Client) (uint64, error) {
	unspent, error := client.ListUnspent()

	if error != nil {
		fmt.Println(error)
		return uint64(0), error
	}

	// Summing and adding every available input
	sum := uint64(0)
	for _, out := range unspent {
		tx.AddTxIn(wire.NewTxIn(GenerateInputIndex(out.TxID, out.Vout), nil, nil))
		sum += uint64(out.Amount * 100000000)
	}

	return sum, nil
}

// createP2PKH creates a basic P2PKH transaction
// Input should be the public key you want to send too
// You can get the the raw publickey from an address with:
// bitcoind getaddressinfo [address]
func createP2PKHScript(scriptAddress []byte) []byte {

	builder := txscript.NewScriptBuilder()

	builder.AddOp(txscript.OP_DUP)
	builder.AddOp(txscript.OP_HASH160)
	builder.AddData(scriptAddress)
	builder.AddOp(txscript.OP_EQUALVERIFY)
	builder.AddOp(txscript.OP_CHECKSIG)

	script, _ := builder.Script()
	return script
}

// GenerateInputIndex generates an input ndex from txid and vout index
func GenerateInputIndex(txid string, outputIndex uint32) *wire.OutPoint {
	prevHash, _ := hex.DecodeString(txid)
	prevHash = reverse(prevHash)
	prevTxHash, _ := chainhash.NewHash(prevHash)
	prevTxHash.SetBytes(prevHash)

	return wire.NewOutPoint(prevTxHash, outputIndex)
}

func reverse(numbers []byte) []byte {
	newNumbers := make([]byte, len(numbers))
	for i, j := 0, len(numbers)-1; i < j; i, j = i+1, j-1 {
		newNumbers[i], newNumbers[j] = numbers[j], numbers[i]
	}
	return newNumbers
}
