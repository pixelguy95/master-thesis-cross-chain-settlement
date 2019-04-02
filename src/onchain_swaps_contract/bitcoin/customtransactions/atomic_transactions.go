package customtransactions

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"time"

	rpcutils "github.com/pixelguy95/master-thesis-cross-chain-settlement/src/extensions/bitcoin"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const secretLenght int64 = 32

var config = &chaincfg.TestNet3Params

// AtomicSwapContractDetails is a basic defenition of a atomic swap contract
// contains the secret so be careful with who you share with
type AtomicSwapContractDetails = struct {
	Contract        []byte
	ContractTx      *wire.MsgTx
	Refund          *wire.MsgTx
	Secret          []byte // Should make this [32]byte too
	SecretHash      [32]byte
	RefundAddress   *btcutil.Address
	ReceiverAddress *btcutil.Address
	Amount          uint64
}

// GenerateAtomicSwapContract generates a new atomic swap contract with a refund tx
func GenerateAtomicSwapContract(reciver btcutil.Address, amount uint64, client *rpcclient.Client) (*AtomicSwapContractDetails, error) {

	clientWraper := rpcutils.New(client)

	//The address that will be used for refunds
	refundTo := clientWraper.GetNewP2PKHAddress()
	refundAddress, _ := btcutil.DecodeAddress(refundTo, config)

	secret := make([]byte, 32)
	rand.Read(secret)
	secretHash := sha256.Sum256(secret)

	//*** CONTRACT ***//
	contract, locktime := createNewContract(refundAddress.ScriptAddress(), reciver.ScriptAddress(), secretHash)

	//*** CONTRACT TRANSACTION ***//
	contractTx := createContractTx(contract, amount, client)

	//*** REFUND ***//
	refundTx, _ := createRefundTx(contractTx, contract, amount, locktime, refundAddress, client)

	swapDetails := &AtomicSwapContractDetails{
		Contract:        contract,
		ContractTx:      contractTx,
		Refund:          refundTx,
		Secret:          secret,
		SecretHash:      secretHash,
		RefundAddress:   &refundAddress,
		ReceiverAddress: &reciver,
		Amount:          amount,
	}

	return swapDetails, nil
}

func createContractTx(contract []byte, amount uint64, client *rpcclient.Client) *wire.MsgTx {

	clientWraper := rpcutils.New(client)

	// Create change address
	changeTo := clientWraper.GetNewP2PKHAddress()
	changeAddress, _ := btcutil.DecodeAddress(changeTo, config)

	P2SHAddressContract, _ := btcutil.NewAddressScriptHash(contract, config)
	contractTxPkScript, _ := txscript.PayToAddrScript(P2SHAddressContract)

	//Create contract tx and fund it
	contractTx := wire.NewMsgTx(2)
	sum, _ := gatherFunds(contractTx, client)

	//Add change output
	change := int64(sum - DefaultFee - amount)
	changeOut := wire.NewTxOut(change, CreateP2PKHScript(changeAddress.ScriptAddress()))
	contractTx.AddTxOut(changeOut)

	contractOut := wire.NewTxOut(int64(amount), contractTxPkScript)
	contractTx.AddTxOut(contractOut)

	//Sign contract tx with wallet
	contractTx, _ = clientWraper.SignRawTransactionWithWallet(contractTx)

	return contractTx
}

func createRefundTx(contractTx *wire.MsgTx, contract []byte, amount uint64, locktime int64, refundAddress btcutil.Address, client *rpcclient.Client) (*wire.MsgTx, error) {

	// Create refund address and dump private key for signing later
	refundPrivKey, error := client.DumpPrivKey(refundAddress)

	//Create refund transaction
	refundTx := wire.NewMsgTx(2)
	refundAmount := int64(amount - DefaultFee)

	//Set lockVaraiable in tx
	refundTx.LockTime = uint32(locktime + 1)

	// Output from refund
	refundOut := wire.NewTxOut(refundAmount, CreateP2PKHScript(refundAddress.ScriptAddress()))
	refundTx.AddTxOut(refundOut)

	// Add contract as input
	contractTxHash := contractTx.TxHash()
	inputIndex := wire.NewOutPoint(&contractTxHash, 1)
	txIn := wire.NewTxIn(inputIndex, nil, nil)
	txIn.Sequence = 0
	refundTx.AddTxIn(txIn)

	//Create signature
	signature, pubKey, error := CreateSignature(refundTx, 0, contract, refundPrivKey)
	if error != nil {
		fmt.Println(error)
		return nil, error
	}

	//Create new refund script with signature and pubkey
	refundInputScript := createAtomicSwapRefundScript(signature, pubKey, contract)
	refundTx.TxIn[0].SignatureScript = refundInputScript

	return refundTx, nil
}

// CreateSignature signs a new transaction (preferably a P2SH tx) and returns a signature and public key
func CreateSignature(tx *wire.MsgTx, inIndex int, contractScript []byte, signingKey *btcutil.WIF) (signature []byte, pubkey []byte, e error) {

	signature, error := txscript.RawTxInSignature(tx, inIndex, contractScript, txscript.SigHashAll, signingKey.PrivKey)
	if error != nil {
		fmt.Println(error)
		return nil, nil, error
	}

	return signature, signingKey.PrivKey.PubKey().SerializeCompressed(), nil
}

func createNewContract(refund []byte, receiver []byte, secretHash [32]byte) ([]byte, int64) {
	locktime := time.Now().Add(48 * time.Hour).Unix() // Refund locktime
	return createAtomicSwapContractScript(refund, receiver, secretHash, locktime), locktime
}

// Creates the script that performs the actual swap
func createAtomicSwapContractScript(meHashKey []byte, receiverHashKey []byte, secretHash [32]byte, locktime int64) []byte {

	builder := txscript.NewScriptBuilder()

	builder.AddOp(txscript.OP_IF)
	{

		// Audit the size of the secret, not super important but good if the secret is larger than the maximum data push
		builder.AddOp(txscript.OP_SIZE)
		builder.AddInt64(secretLenght)
		builder.AddOp(txscript.OP_EQUALVERIFY)

		// Verify that the given secret hashed is equal to the hashed secret
		builder.AddOp(txscript.OP_SHA256)
		builder.AddData(secretHash[:])
		builder.AddOp(txscript.OP_EQUALVERIFY)

		// Prepare for equal verify on pub key
		// Using the receiver address
		builder.AddOp(txscript.OP_DUP)
		builder.AddOp(txscript.OP_HASH160)
		builder.AddData(receiverHashKey)
	}
	builder.AddOp(txscript.OP_ELSE)
	{
		// If a refund is in order check that the locktime has passed
		builder.AddInt64(locktime)
		builder.AddOp(txscript.OP_CHECKLOCKTIMEVERIFY)
		builder.AddOp(txscript.OP_DROP)

		// Prepare for equal verify on pub key
		// Using the refund address
		builder.AddOp(txscript.OP_DUP)
		builder.AddOp(txscript.OP_HASH160)
		builder.AddData(meHashKey)
	}
	builder.AddOp(txscript.OP_ENDIF)

	builder.AddOp(txscript.OP_EQUALVERIFY)
	builder.AddOp(txscript.OP_CHECKSIG)

	script, _ := builder.Script()
	return script
}

// Creates the script that performs the actual swap
func createAtomicSwapRefundScript(signature []byte, pubKey []byte, contract []byte) []byte {

	builder := txscript.NewScriptBuilder()

	builder.AddData(signature)
	builder.AddData(pubKey)
	builder.AddInt64(0)
	builder.AddData(contract)

	script, _ := builder.Script()
	return script
}
