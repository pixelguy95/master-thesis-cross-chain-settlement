package customtransactions

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

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
	Amount     uint64
}

// GenerateAtomicSwapTransaction creates a new atomic swap transaction
func GenerateAtomicSwapTransaction(receiver *btcutil.AddressPubKey, amount uint64, client *rpcclient.Client) (*AtomicSwapWithSecret, error) {

	clientWraper := rpcutils.New(client)
	sender, _ := clientWraper.GetNewPubKey()

	// Generate a new secret, 32 bytes
	//secret := make([]byte, 32)
	//rand.Read(secret)

	//TODO uncomment real random secret bytes
	secret := []byte("secret")
	fmt.Println("secret hex: ", hex.EncodeToString(secret))

	// Hash of secret
	secretHash := sha256.Sum256(secret)

	tx := wire.NewMsgTx(2)

	// Adding inputs and summing them
	sum, _ := gatherFunds(tx, client)

	// Change output
	change := sum - amount - DefaultFee

	changeTo := clientWraper.GetNewP2PKHAddress()
	changeToAddress, _ := btcutil.DecodeAddress(changeTo, &chaincfg.TestNet3Params)
	tx.AddTxOut(wire.NewTxOut(int64(change), createP2PKHScript(changeToAddress.ScriptAddress())))

	outputscript := createAtomicSwapScript(receiver.PubKey().SerializeCompressed(), sender.PubKey().SerializeCompressed(), secretHash)
	tx.AddTxOut(wire.NewTxOut(int64(amount), outputscript))

	swap := &AtomicSwapWithSecret{
		Tx:         tx,
		Secret:     secret,
		SecretHash: secretHash,
		Sender:     sender,
		Reciver:    receiver,
		Amount:     amount,
	}

	return swap, nil
}

// GenerateAtomicClaimWithSecret creates a new atomic swap transaction
func GenerateAtomicClaimWithSecret(outIndex *wire.OutPoint, amount uint64, client *rpcclient.Client) (*wire.MsgTx, error) {

	clientWraper := rpcutils.New(client)

	// Generate a new secret, 32 bytes
	//secret := make([]byte, 32)
	//rand.Read(secret)

	//TODO uncomment real random secret bytes
	secret := []byte("secret")

	// Hash of secret
	//secretHash := sha256.Sum256(secret)

	tx := wire.NewMsgTx(2)

	// Output address, total amount
	claimAddress := clientWraper.GetNewP2PKHAddress()
	claimToAddress, _ := btcutil.DecodeAddress(claimAddress, &chaincfg.TestNet3Params)
	tx.AddTxOut(wire.NewTxOut(int64(amount-DefaultFee), createP2PKHScript(claimToAddress.ScriptAddress())))

	// TxIn claim script etc...
	tx.AddTxIn(wire.NewTxIn(outIndex, createAtomicSwapSecretClaimScript(nil, secret[:]), nil))

	return tx, nil
}

// Creates the script that performs the actual swap
func createAtomicSwapScript(meHashKey []byte, receiverHashKey []byte, secretHash [32]byte) []byte {

	builder := txscript.NewScriptBuilder()

	builder.AddOp(txscript.OP_IF)
	{
		// Verify that the given secret hashed is equal to the hashed secret
		builder.AddOp(txscript.OP_SHA256)
		builder.AddData(secretHash[:])
		builder.AddOp(txscript.OP_EQUALVERIFY)

		// Check that the hashed pubkey is the recivers hashed key
		builder.AddOp(txscript.OP_DUP)
		builder.AddOp(txscript.OP_HASH160)
		builder.AddData(receiverHashKey)
	}
	builder.AddOp(txscript.OP_ELSE)
	{

	}
	builder.AddOp(txscript.OP_ENDIF)

	builder.AddOp(txscript.OP_EQUALVERIFY)
	builder.AddOp(txscript.OP_CHECKSIG)

	script, _ := builder.Script()
	return script
}

// Creates the script that performs the actual swap
// NOTE: the aKey and bKey should be the actual pubkeys and not the hash160 equivalent
// Old
func createAtomicSwapScriptOld(aKey []byte, bKey []byte, secretHash [32]byte) []byte {

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

// Creates the script that performs the actual swap
// NOTE: the aKey and bKey should be the actual pubkeys and not the hash160 equivalent
func createAtomicSwapSecretClaimScript(bSig []byte, secret []byte) []byte {

	builder := txscript.NewScriptBuilder()

	builder.AddData(bSig)
	builder.AddData(secret[:])
	builder.AddOp(txscript.OP_0)

	script, _ := builder.Script()
	return script
}
