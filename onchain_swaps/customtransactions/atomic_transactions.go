package customtransactions

import (
	"crypto/rand"
	"crypto/sha256"

	"../rpcutils"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

// AtomicSwapWithSecret is a basic defenition of a atomic swap
// contains the secret so be careful with who you share with
type AtomicSwapWithSecret = struct {
	Tx         *wire.MsgTx
	Secret     []byte // Should make this [32]byte too
	SecretHash [32]byte
	Sender     *btcutil.AddressPubKey
	Reciver    *btcutil.AddressPubKey
}

// GenerateAtomicSwapTransaction creates a new atomic swap transaction
func GenerateAtomicSwapTransaction(receiver *btcutil.AddressPubKey, amount uint64, client *rpcclient.Client) (*AtomicSwapWithSecret, error) {

	// Generate a new secret, 32 bytes
	secret := make([]byte, 32)
	rand.Read(secret)

	// Hash of secret
	secretHash := sha256.Sum256(secret)

	tx := wire.NewMsgTx(2)

	// Adding inputs and summing them
	sum, _ := gatherFunds(tx, client)

	// Change output
	change := sum - amount - DefaultFee
	clientWraper := rpcutils.New(client)
	changeTo := clientWraper.GetNewP2PKHAddress()
	changeToAddress, _ := btcutil.DecodeAddress(changeTo, &chaincfg.TestNet3Params)
	tx.AddTxOut(wire.NewTxOut(int64(change), createP2PKHScript(changeToAddress.ScriptAddress())))

	outputscript := createAtomicSwapScript(receiver.PubKey().SerializeCompressed(), nil, secretHash)
	tx.AddTxOut(wire.NewTxOut(int64(amount), outputscript))

	swap := &AtomicSwapWithSecret{
		Tx:         tx,
		Secret:     secret,
		SecretHash: secretHash,
		Reciver:    receiver,
	}

	return swap, nil
}

// Creates the script that performs the actual swap
// NOTE: the aKey and bKey should be the actual pubkeys and not the hash160 equivalent
func createAtomicSwapScript(aKey []byte, bKey []byte, secretHash [32]byte) []byte {

	builder := txscript.NewScriptBuilder()

	builder.AddOp(txscript.OP_IF)
	builder.AddOp(txscript.OP_2)
	builder.AddData(aKey)
	builder.AddData(bKey)
	builder.AddOp(txscript.OP_CHECKMULTISIG) //Should leave a 1 on the stack if correct

	builder.AddOp(txscript.OP_ELSE)
	builder.AddData(bKey)
	builder.AddOp(txscript.OP_CHECKSIGVERIFY)
	builder.AddOp(txscript.OP_SHA256)
	builder.AddData(secretHash[:])
	builder.AddOp(txscript.OP_EQUAL) //Should leave a 1 on the stack if correct
	builder.AddOp(txscript.OP_ENDIF)

	script, _ := builder.Script()
	return script
}
