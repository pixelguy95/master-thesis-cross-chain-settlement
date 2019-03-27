package channel

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/lightningnetwork/lnd/input"
	"github.com/pixelguy95/master-thesis-cross-chain-settlement/src/onchain_swaps_contract/bitcoin/customtransactions"
)

func (c *Channel) GenerateSenderCommitTimeoutTx(index, cltvLocktime uint32, sender *User, receiver *User) error {

	_, _, revocationPub, _ := GenerateRevokePubKey(sender.RevokePreImage, receiver.FundingPublicKey)

	commitTimeout := wire.NewMsgTx(2)
	commitTimeout.LockTime = cltvLocktime

	commitInput := &wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  sender.Commits[index].CommitTx.TxHash(),
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
		Value:    sender.Commits[index].CommitTx.TxOut[2].Value - int64(customtransactions.DefaultFee),
	}

	commitTimeout.AddTxOut(secondLevelOutput)

	signSenderCommitTimeoutTx(commitTimeout, witnessScript, sender.Commits[index].CommitTx.TxOut[2], sender, receiver)

	/* REDEEM */
	redeem, err := generateSenderCommitTimeoutRedeemTx(commitTimeout, witnessScript, sender)
	if err != nil {
		return err
	}

	/* REVOKE */
	revocationPrivateKey := input.DeriveRevocationPrivKey(receiver.FundingPrivateKey, sender.RevokationSecrets[index].CommitSecret)
	revoke, err := generateSenderCommitTimeoutRevokeTx(commitTimeout, witnessScript, receiver, revocationPrivateKey)
	if err != nil {
		return err
	}

	if sender.HTLCOutputTxs[index] == nil {
		sender.HTLCOutputTxs[index] = &HTLCOutputTxs{
			SenderCommitTimeoutScript:   witnessScript,
			SenderCommitTimeoutTx:       commitTimeout,
			SenderCommitTimeoutRedeemTx: redeem,
		}
	} else {
		sender.HTLCOutputTxs[index].SenderCommitTimeoutScript = witnessScript
		sender.HTLCOutputTxs[index].SenderCommitTimeoutTx = commitTimeout
		sender.HTLCOutputTxs[index].SenderCommitTimeoutRedeemTx = redeem
	}

	if receiver.HTLCOutputTxs[index] == nil {
		receiver.HTLCOutputTxs[index] = &HTLCOutputTxs{
			SenderCommitTimeoutRevokeTx: revoke,
		}
	} else {
		receiver.HTLCOutputTxs[index].SenderCommitTimeoutRevokeTx = revoke
	}

	return nil
}

func signSenderCommitTimeoutTx(senderCommitTimeoutTx *wire.MsgTx, commitScript []byte, commitOut *wire.TxOut, sender *User, receiver *User) ([][]byte, error) {

	signDesc := input.SignDescriptor{
		WitnessScript: commitScript,
		Output:        commitOut,
		HashType:      txscript.SigHashAll,
		SigHashes:     txscript.NewTxSigHashes(senderCommitTimeoutTx),
		InputIndex:    0,
	}

	s := &SimpleSigner{
		PrivateKey: sender.HTLCPrivateKey,
	}
	senderSignature, err := s.SignOutputRaw(senderCommitTimeoutTx, &signDesc)
	if err != nil {
		return nil, err
	}

	s = &SimpleSigner{
		PrivateKey: receiver.HTLCPrivateKey,
	}
	receiverSignature, err := s.SignOutputRaw(senderCommitTimeoutTx, &signDesc)
	if err != nil {
		return nil, err
	}

	witnessStack := wire.TxWitness(make([][]byte, 5))
	witnessStack[0] = nil
	witnessStack[1] = append(receiverSignature, byte(signDesc.HashType))
	witnessStack[2] = append(senderSignature, byte(signDesc.HashType))
	witnessStack[3] = nil
	witnessStack[4] = signDesc.WitnessScript

	senderCommitTimeoutTx.TxIn[0].Witness = witnessStack

	return witnessStack, nil
}

func generateSenderCommitTimeoutRedeemTx(commitTimeoutTx *wire.MsgTx, commitTimeoutScript []byte, sender *User) (*wire.MsgTx, error) {

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

func generateSenderCommitTimeoutRevokeTx(commitTimeoutTx *wire.MsgTx, commitTimeoutScript []byte, receiver *User, revokePrivateKey *btcec.PrivateKey) (*wire.MsgTx, error) {

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
