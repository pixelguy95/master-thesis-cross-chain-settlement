package channel

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/lightningnetwork/lnd/input"
	"github.com/pixelguy95/master-thesis-cross-chain-settlement/src/onchain_swaps_contract/bitcoin/customtransactions"
)

func (c *Channel) GenerateReceiverCommitSuccessTx(index uint32, sender *User, receiver *User) error {

	revocationPrivateKey := input.DeriveRevocationPrivKey(sender.FundingPrivateKey, receiver.RevokationSecrets[index].CommitSecret)
	revocationPub := revocationPrivateKey.PubKey()

	commitSuccess := wire.NewMsgTx(2)

	commitInput := &wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  receiver.Commits[index].CommitTx.TxHash(),
			Index: 2,
		},
	}

	commitSuccess.AddTxIn(commitInput)

	witnessScript, err := input.SecondLevelHtlcScript(revocationPub, receiver.HTLCPublicKey, DefaultRelativeLockTime)
	if err != nil {
		return err
	}

	witnesScriptHash, _ := input.WitnessScriptHash(witnessScript)

	secondLevelOutput := &wire.TxOut{
		PkScript: witnesScriptHash,
		Value:    receiver.Commits[index].CommitTx.TxOut[2].Value - int64(customtransactions.DefaultFee),
	}

	commitSuccess.AddTxOut(secondLevelOutput)

	htlcScript := receiver.Commits[index].HTLCOutScript
	signReceiverCommitSuccessTx(commitSuccess, htlcScript, receiver.Commits[index].CommitTx.TxOut[2], sender, receiver)

	// REDEEM
	redeem, err := generateReceiverCommitSuccessRedeemTx(commitSuccess, witnessScript, receiver)
	if err != nil {
		return err
	}

	// REVOKE
	revoke, err := generateReceiverCommitSuccessRevokeTx(commitSuccess, witnessScript, sender, revocationPrivateKey)
	if err != nil {
		return err
	}

	if receiver.HTLCOutputTxs[index] == nil {
		receiver.HTLCOutputTxs[index] = &HTLCOutputTxs{
			ReceiverCommitSuccessScript:   witnessScript,
			ReceiverCommitSuccessTx:       commitSuccess,
			ReceiverCommitSuccessRedeemTx: redeem,
		}
	} else {
		receiver.HTLCOutputTxs[index].ReceiverCommitSuccessScript = witnessScript
		receiver.HTLCOutputTxs[index].ReceiverCommitSuccessTx = commitSuccess
		receiver.HTLCOutputTxs[index].ReceiverCommitSuccessRedeemTx = redeem
	}

	if sender.HTLCOutputTxs[index] == nil {
		sender.HTLCOutputTxs[index] = &HTLCOutputTxs{
			ReceiverCommitSuccessRevokeTx: revoke,
		}
	} else {
		sender.HTLCOutputTxs[index].ReceiverCommitSuccessRevokeTx = revoke
	}

	return nil
}

func signReceiverCommitSuccessTx(receiverCommitSuccessTx *wire.MsgTx, commitScript []byte, commitOut *wire.TxOut, sender *User, receiver *User) ([][]byte, error) {

	signDesc := input.SignDescriptor{
		WitnessScript: commitScript,
		Output:        commitOut,
		HashType:      txscript.SigHashAll,
		SigHashes:     txscript.NewTxSigHashes(receiverCommitSuccessTx),
		InputIndex:    0,
	}

	s := &SimpleSigner{
		PrivateKey: receiver.HTLCPrivateKey,
	}
	receiverSignature, err := s.SignOutputRaw(receiverCommitSuccessTx, &signDesc)
	if err != nil {
		return nil, err
	}

	s = &SimpleSigner{
		PrivateKey: sender.HTLCPrivateKey,
	}
	senderSignature, err := s.SignOutputRaw(receiverCommitSuccessTx, &signDesc)
	if err != nil {
		return nil, err
	}

	witnessStack := wire.TxWitness(make([][]byte, 5))
	witnessStack[0] = nil
	witnessStack[1] = append(senderSignature, byte(signDesc.HashType))
	witnessStack[2] = append(receiverSignature, byte(signDesc.HashType))
	witnessStack[3] = sender.HTLCPreImage[:]
	witnessStack[4] = signDesc.WitnessScript

	receiverCommitSuccessTx.TxIn[0].Witness = witnessStack

	return witnessStack, nil
}

func generateReceiverCommitSuccessRedeemTx(commitSuccessTx *wire.MsgTx, commitSuccessScript []byte, receiver *User) (*wire.MsgTx, error) {

	redeem := wire.NewMsgTx(2)

	redeem.AddTxIn(&wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  commitSuccessTx.TxHash(),
			Index: 0,
		},
		Sequence: DefaultRelativeLockTime,
	})

	outputScript, err := txscript.PayToAddrScript(receiver.PayOutAddress)
	if err != nil {
		return nil, err
	}

	redeem.AddTxOut(&wire.TxOut{
		PkScript: outputScript,
		Value:    commitSuccessTx.TxOut[0].Value - int64(customtransactions.DefaultFee),
	})

	signDesc := input.SignDescriptor{
		WitnessScript: commitSuccessScript,
		Output:        commitSuccessTx.TxOut[0],
		HashType:      txscript.SigHashAll,
		SigHashes:     txscript.NewTxSigHashes(redeem),
		InputIndex:    0,
	}

	s := &SimpleSigner{
		PrivateKey: receiver.HTLCPrivateKey,
	}

	signature, err := s.SignOutputRaw(redeem, &signDesc)
	if err != nil {
		return nil, err
	}

	witnessStack := wire.TxWitness(make([][]byte, 3))
	witnessStack[0] = append(signature, byte(signDesc.HashType))
	witnessStack[1] = nil
	witnessStack[2] = commitSuccessScript

	redeem.TxIn[0].Witness = witnessStack

	return redeem, nil
}

func generateReceiverCommitSuccessRevokeTx(commitSuccessTx *wire.MsgTx, commitSuccessScript []byte, sender *User, revokePrivateKey *btcec.PrivateKey) (*wire.MsgTx, error) {

	revoke := wire.NewMsgTx(2)

	revoke.AddTxIn(&wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  commitSuccessTx.TxHash(),
			Index: 0,
		},
	})

	outputScript, err := txscript.PayToAddrScript(sender.PayOutAddress)
	if err != nil {
		return nil, err
	}

	revoke.AddTxOut(&wire.TxOut{
		PkScript: outputScript,
		Value:    commitSuccessTx.TxOut[0].Value - int64(customtransactions.DefaultFee),
	})

	signDesc := input.SignDescriptor{
		WitnessScript: commitSuccessScript,
		Output:        commitSuccessTx.TxOut[0],
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
	witnessStack[2] = commitSuccessScript

	revoke.TxIn[0].Witness = witnessStack

	return revoke, nil
}
