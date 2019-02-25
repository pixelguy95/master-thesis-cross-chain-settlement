package customtransactions

import (
	"encoding/hex"
	"fmt"

	rpcutils "github.com/pixelguy95/btcd-rpcclient-extension/litecoin"
	
	"github.com/ltcsuite/ltcd/chaincfg"
	"github.com/ltcsuite/ltcd/chaincfg/chainhash"
	"github.com/ltcsuite/ltcd/rpcclient"
	"github.com/ltcsuite/ltcd/txscript"
	"github.com/ltcsuite/ltcd/wire"
	"github.com/ltcsuite/ltcutil"
)

// DefaultFee is the default fee that will be payed in a transaction
const DefaultFee uint64 = 2000

// GenerateP2PKHTransaction creates a new P2PKH transaction with some parameters
func GenerateP2PKHTransaction(payTo string, amount uint64, client *rpcclient.Client) (*wire.MsgTx, error) {

	clientWraper := rpcutils.New(client)
	changeTo := clientWraper.GetNewP2PKHAddress()

	tx := wire.NewMsgTx(2)

	payToAddress, _ := ltcutil.DecodeAddress(payTo, &chaincfg.TestNet4Params)
	changeToAddress, _ := ltcutil.DecodeAddress(changeTo, &chaincfg.TestNet4Params)

	sum, _ := gatherFunds(tx, client)

	// Output to reciver
	tx.AddTxOut(wire.NewTxOut(int64(amount), CreateP2PKHScript(payToAddress.ScriptAddress())))

	// Change output
	change := sum - amount - DefaultFee
	tx.AddTxOut(wire.NewTxOut(int64(change), CreateP2PKHScript(changeToAddress.ScriptAddress())))

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

// CreateP2PKHScript creates a basic P2PKH transaction
func CreateP2PKHScript(scriptAddress []byte) []byte {

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

// GenerateInputIndexWithByteHash generates an input ndex from txid and vout index
func GenerateInputIndexWithByteHash(txid []byte, outputIndex uint32) *wire.OutPoint {
	prevTxHash, _ := chainhash.NewHash(txid)
	return wire.NewOutPoint(prevTxHash, outputIndex)
}

func reverse(numbers []byte) []byte {
	newNumbers := make([]byte, len(numbers))
	for i, j := 0, len(numbers)-1; i < j; i, j = i+1, j-1 {
		newNumbers[i], newNumbers[j] = numbers[j], numbers[i]
	}
	return newNumbers
}
