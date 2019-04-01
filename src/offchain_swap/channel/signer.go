package channel

import (
	"log"

	"github.com/btcsuite/btcd/btcec"
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

// SignCommitsTx Signs a commit
func (c *Channel) SignCommitsTx(commitIndex uint) error {

	err := c.signCommit(false, commitIndex)
	if err != nil {
		return err
	}

	err = c.signCommit(true, commitIndex)
	if err != nil {
		return err
	}

	return nil
}

func (c *Channel) signCommit(reverse bool, commitIndex uint) error {
	log.Printf("Signing commit transaction\t\t ")

	var holder *User
	var other *User
	if !reverse {
		holder = c.Party1
		other = c.Party2
	} else {
		holder = c.Party2
		other = c.Party1
	}

	// Generate a signature for their version of the initial commitment
	// transaction.
	signDesc := input.SignDescriptor{
		WitnessScript: c.FundingWitnessScript,
		Output:        c.FundingMultiSigOut,
		HashType:      txscript.SigHashAll,
		SigHashes:     txscript.NewTxSigHashes(holder.Commits[commitIndex].CommitTx),
		InputIndex:    0,
	}

	s := &SimpleSigner{
		PrivateKey: holder.FundingPrivateKey,
	}

	// Signature from holder of commit
	signature1, err := s.SignOutputRaw(holder.Commits[commitIndex].CommitTx, &signDesc)
	if err != nil {
		log.Fatal(err)
		return err
	}

	s = &SimpleSigner{
		PrivateKey: other.FundingPrivateKey,
	}

	//Signature from the counter party
	signature2, err := s.SignOutputRaw(holder.Commits[commitIndex].CommitTx, &signDesc)
	if err != nil {
		log.Fatal(err)
		return err
	}

	signature1 = append(signature1, byte(txscript.SigHashAll))
	signature2 = append(signature2, byte(txscript.SigHashAll))

	//Below caused problems
	witness := input.SpendMultiSig(c.FundingWitnessScript, holder.FundingPublicKey.SerializeCompressed(), signature1, other.FundingPublicKey.SerializeCompressed(), signature2)
	holder.Commits[commitIndex].CommitTx.TxIn[0].Witness = witness

	return nil
}
