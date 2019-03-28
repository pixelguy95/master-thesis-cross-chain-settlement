package channel

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/lightningnetwork/lnd/input"
	"github.com/pixelguy95/master-thesis-cross-chain-settlement/src/onchain_swaps_contract/bitcoin/customtransactions"
)

func (c *Channel) GenerateReceiverCommitTimeoutTx(index, cltvLocktime uint32, sender *User, receiver *User) error {

	revocationPrivateKey := input.DeriveRevocationPrivKey(receiver.FundingPrivateKey, sender.RevokationSecrets[index].CommitSecret)
	revocationPub := revocationPrivateKey.PubKey()

	commitTimeout := wire.NewMsgTx(2)
	commitTimeout.LockTime = cltvLocktime

	commitInput := &wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  receiver.Commits[index].CommitTx.TxHash(),
			Index: 2,
		},
	}

	commitTimeout.AddTxIn(commitInput)

	witnessScript, err := input.SecondLevelHtlcScript(revocationPub, sender.HTLCPublicKey, DefaultRelativeLockTime)
	if err != nil {
		return err
	}

	witnesScriptHash, _ := input.WitnessScriptHash(witnessScript)

	secondLevelOutput := &wire.TxOut{
		PkScript: witnesScriptHash,
		Value:    receiver.Commits[index].CommitTx.TxOut[2].Value - int64(customtransactions.DefaultFee),
	}

	commitTimeout.AddTxOut(secondLevelOutput)

	htlcScript := receiver.Commits[index].HTLCOutScript
	signReceiverCommitTimeoutTx(commitTimeout, htlcScript, receiver.Commits[index].CommitTx.TxOut[2], sender, receiver)

	/* REDEEM */
	redeem, err := generateReceiverCommitTimeoutRedeemTx(commitTimeout, witnessScript, sender)
	if err != nil {
		return err
	}

	/* REVOKE */
	revoke, err := generateSenderCommitTimeoutRevokeTx(commitTimeout, witnessScript, receiver, revocationPrivateKey)
	if err != nil {
		return err
	}

	if sender.HTLCOutputTxs[index] == nil {
		sender.HTLCOutputTxs[index] = &HTLCOutputTxs{
			ReceiverCommitTimeoutScript:   witnessScript,
			ReceiverCommitTimeoutTx:       commitTimeout,
			ReceiverCommitTimeoutRedeemTx: redeem,
		}
	} else {
		sender.HTLCOutputTxs[index].ReceiverCommitTimeoutScript = witnessScript
		sender.HTLCOutputTxs[index].ReceiverCommitTimeoutTx = commitTimeout
		sender.HTLCOutputTxs[index].ReceiverCommitTimeoutRedeemTx = redeem
	}

	if receiver.HTLCOutputTxs[index] == nil {
		receiver.HTLCOutputTxs[index] = &HTLCOutputTxs{
			ReceiverCommitTimeoutRevokeTx: revoke,
		}
	} else {
		receiver.HTLCOutputTxs[index].ReceiverCommitTimeoutRevokeTx = revoke
	}

	return nil
}

func signReceiverCommitTimeoutTx(receiverCommitTimeoutTx *wire.MsgTx, commitScript []byte, commitOut *wire.TxOut, sender *User, receiver *User) ([][]byte, error) {

	signDesc := input.SignDescriptor{
		WitnessScript: commitScript,
		Output:        commitOut,
		HashType:      txscript.SigHashAll,
		SigHashes:     txscript.NewTxSigHashes(receiverCommitTimeoutTx),
		InputIndex:    0,
	}

	s := &SimpleSigner{
		PrivateKey: sender.HTLCPrivateKey,
	}
	senderSignature, err := s.SignOutputRaw(receiverCommitTimeoutTx, &signDesc)
	if err != nil {
		return nil, err
	}

	witnessStack := wire.TxWitness(make([][]byte, 3))
	witnessStack[0] = append(senderSignature, byte(signDesc.HashType))
	witnessStack[1] = nil
	witnessStack[2] = signDesc.WitnessScript

	receiverCommitTimeoutTx.TxIn[0].Witness = witnessStack

	return witnessStack, nil
}

func generateReceiverCommitTimeoutRedeemTx(commitTimeoutTx *wire.MsgTx, commitTimeoutScript []byte, sender *User) (*wire.MsgTx, error) {

	redeem := wire.NewMsgTx(2)

	redeem.AddTxIn(&wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  commitTimeoutTx.TxHash(),
			Index: 0,
		},
		Sequence: DefaultRelativeLockTime,
	})

	outputScript, err := txscript.PayToAddrScript(sender.PayOutAddress)
	if err != nil {
		return nil, err
	}

	redeem.AddTxOut(&wire.TxOut{
		PkScript: outputScript,
		Value:    commitTimeoutTx.TxOut[0].Value - int64(customtransactions.DefaultFee),
	})

	signDesc := input.SignDescriptor{
		WitnessScript: commitTimeoutScript,
		Output:        commitTimeoutTx.TxOut[0],
		HashType:      txscript.SigHashAll,
		SigHashes:     txscript.NewTxSigHashes(redeem),
		InputIndex:    0,
	}

	s := &SimpleSigner{
		PrivateKey: sender.HTLCPrivateKey,
	}

	signature, err := s.SignOutputRaw(redeem, &signDesc)
	if err != nil {
		return nil, err
	}

	witnessStack := wire.TxWitness(make([][]byte, 3))
	witnessStack[0] = append(signature, byte(signDesc.HashType))
	witnessStack[1] = nil
	witnessStack[2] = commitTimeoutScript

	redeem.TxIn[0].Witness = witnessStack

	return redeem, nil
}

func generateReceiverCommitTimeoutRevokeTx(commitTimeoutTx *wire.MsgTx, commitTimeoutScript []byte, receiver *User, revokePrivateKey *btcec.PrivateKey) (*wire.MsgTx, error) {

	revoke := wire.NewMsgTx(2)

	revoke.AddTxIn(&wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  commitTimeoutTx.TxHash(),
			Index: 0,
		},
	})

	outputScript, err := txscript.PayToAddrScript(receiver.PayOutAddress)
	if err != nil {
		return nil, err
	}

	revoke.AddTxOut(&wire.TxOut{
		PkScript: outputScript,
		Value:    commitTimeoutTx.TxOut[0].Value - int64(customtransactions.DefaultFee),
	})

	signDesc := input.SignDescriptor{
		WitnessScript: commitTimeoutScript,
		Output:        commitTimeoutTx.TxOut[0],
		HashType:      txscript.SigHashAll,
		SigHashes:     txscript.NewTxSigHashes(revoke),
		InputIndex:    0,
	}

	s := &SimpleSigner{
		PrivateKey: revokePrivateKey,
	}

	signature, err := s.SignOutputRaw(revoke, &signDesc)
	if err != nil {
		return nil, err
	}

	witnessStack := wire.TxWitness(make([][]byte, 3))
	witnessStack[0] = append(signature, byte(signDesc.HashType))
	witnessStack[1] = []byte{1}
	witnessStack[2] = commitTimeoutScript

	revoke.TxIn[0].Witness = witnessStack

	return revoke, nil
}
