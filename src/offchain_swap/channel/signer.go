package channel

import (
	"errors"
	"fmt"
	"reflect"

	rpcutils "../../extensions/bitcoin"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/lightningnetwork/lnd/input"
)

// SimpleSigner the struct for a basic signer
type SimpleSigner struct {
	PrivateKey *btcec.PrivateKey
}

// SignOutputRaw signs a raw input with given private key
func (b *SimpleSigner) SignOutputRaw(tx *wire.MsgTx,
	signDesc *input.SignDescriptor) ([]byte, error) {

	witnessScript := signDesc.WitnessScript
	privKey := b.PrivateKey

	// If a tweak (single or double) is specified, then we'll need to use
	// this tweak to derive the final private key to be used for signing
	// this output.
	privKey, err := maybeTweakPrivKey(signDesc, privKey)
	if err != nil {
		return nil, err
	}

	// TODO(roasbeef): generate sighash midstate if not present?

	amt := signDesc.Output.Value
	sig, err := txscript.RawTxInWitnessSignature(
		tx, signDesc.SigHashes, signDesc.InputIndex, amt,
		witnessScript, signDesc.HashType, privKey,
	)
	if err != nil {
		return nil, err
	}

	// Chop off the sighash flag at the end of the signature.
	return sig[:len(sig)-1], nil
}

// maybeTweakPrivKey examines the single and double tweak parameters on the
// passed sign descriptor and may perform a mapping on the passed private key
// in order to utilize the tweaks, if populated.
func maybeTweakPrivKey(signDesc *input.SignDescriptor,
	privKey *btcec.PrivateKey) (*btcec.PrivateKey, error) {

	var retPriv *btcec.PrivateKey
	switch {

	case signDesc.SingleTweak != nil:
		retPriv = input.TweakPrivKey(privKey,
			signDesc.SingleTweak)

	case signDesc.DoubleTweak != nil:
		retPriv = input.DeriveRevocationPrivKey(privKey,
			signDesc.DoubleTweak)

	default:
		retPriv = privKey
	}

	return retPriv, nil
}

// SignCommitTx Signs a commit
func (c *Channel) SignCommitTx(reverse bool, commitIndex uint, client *rpcclient.Client) error {

	var holder *User
	var other *User
	if !reverse {
		holder = c.Party1
		other = c.Party2
	} else {
		holder = c.Party2
		other = c.Party1
	}

	clientWraper := rpcutils.New(client)

	clientWraper.UnloadAllWallets()
	clientWraper.LoadWallet(holder.WalletName)
	holderPubKey, _ := clientWraper.GetPubKey(holder.FundingPublicKey.EncodeAddress())

	clientWraper.UnloadAllWallets()
	clientWraper.LoadWallet(other.WalletName)
	//otherPubKey, _ := clientWraper.GetPubKey(other.FundingPublicKey.EncodeAddress())

	// Generate a signature for their version of the initial commitment
	// transaction.
	signDesc := input.SignDescriptor{
		WitnessScript: c.FundingWitnessScript,
		Output:        c.FundingMultiSigOut,
		HashType:      txscript.SigHashAll,
		SigHashes:     txscript.NewTxSigHashes(holder.Commits[commitIndex].Data.CommitTx),
		InputIndex:    0,
	}

	s := &SimpleSigner{
		PrivateKey: holder.FundingPrivateKey.PrivKey,
	}

	// Signature from holder of commit
	signature1, err := s.SignOutputRaw(holder.Commits[commitIndex].Data.CommitTx, &signDesc)
	if err != nil {
		fmt.Println(err)
		return err
	}

	s = &SimpleSigner{
		PrivateKey: other.FundingPrivateKey.PrivKey,
	}

	//Signature from the counter party
	signature2, err := s.SignOutputRaw(holder.Commits[commitIndex].Data.CommitTx, &signDesc)
	if err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Printf("\nSignature1\n%x\n", signature1)
	fmt.Printf("\nSignature2\n%x\n", signature2)

	signature1 = append(signature1, byte(txscript.SigHashAll))
	signature2 = append(signature2, byte(txscript.SigHashAll))

	witnessStack := make([][]byte, 4)
	witnessStack[0] = nil
	witnessStack[3] = c.FundingWitnessScript

	pushes, _ := txscript.PushedData(c.FundingWitnessScript)
	for index, data := range pushes {
		if reflect.DeepEqual(data, holderPubKey.PubKey().SerializeCompressed()) {

			if index == 0 {
				witnessStack[1] = signature1
				witnessStack[2] = signature2
			} else {
				witnessStack[1] = signature1
				witnessStack[2] = signature2
			}
		}
	}

	//Below caused problems
	//witness := input.SpendMultiSig(c.FundingWitnessScript, holderPubKey.ScriptAddress(), signature1, otherPubKey.ScriptAddress(), signature2)
	holder.Commits[commitIndex].Data.CommitTx.TxIn[0].Witness = witnessStack

	if !reflect.DeepEqual(c.Party1.Commits[commitIndex].Data.CommitTx.TxIn[0].Witness, holder.Commits[commitIndex].Data.CommitTx.TxIn[0].Witness) {
		return errors.New("Witness didn't match somehow")
	}

	return nil
}
