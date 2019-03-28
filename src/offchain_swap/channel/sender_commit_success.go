package channel

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/lightningnetwork/lnd/input"
	"github.com/pixelguy95/master-thesis-cross-chain-settlement/src/onchain_swaps_contract/bitcoin/customtransactions"
)

// GenerateSenderCommitSuccessTx consumes a commit with secret reveal
func (c *Channel) GenerateSenderCommitSuccessTx(index uint32, sender *User, receiver *User) error {

	revocationPrivateKey := input.DeriveRevocationPrivKey(receiver.FundingPrivateKey, sender.RevokationSecrets[index].CommitSecret)
	revocationPub := revocationPrivateKey.PubKey()

	commitSuccess := wire.NewMsgTx(2)

	commitInput := &wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  sender.Commits[index].CommitTx.TxHash(),
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
		Value:    sender.Commits[index].CommitTx.TxOut[2].Value - int64(customtransactions.DefaultFee),
	}

	commitSuccess.AddTxOut(secondLevelOutput)

	htlcScript := sender.Commits[index].HTLCOutScript
	signSenderCommitSuccessTx(commitSuccess, htlcScript, sender.Commits[index].CommitTx.TxOut[2], sender, receiver)

	// REDEEM
	redeem, err := generateSenderCommitSuccessRedeemTx(commitSuccess, witnessScript, receiver)
	if err != nil {
		return err
	}

	// REVOKE
	revoke, err := generateSenderCommitSuccessRevokeTx(commitSuccess, witnessScript, sender, revocationPrivateKey)
	if err != nil {
		return err
	}

	if sender.HTLCOutputTxs[index] == nil {
		sender.HTLCOutputTxs[index] = &HTLCOutputTxs{
			SenderCommitSuccessScript:   witnessScript,
			SenderCommitSuccessTx:       commitSuccess,
			SenderCommitSuccessRedeemTx: redeem,
		}
	} else {
		sender.HTLCOutputTxs[index].SenderCommitSuccessScript = witnessScript
		sender.HTLCOutputTxs[index].SenderCommitSuccessTx = commitSuccess
		sender.HTLCOutputTxs[index].SenderCommitSuccessRedeemTx = redeem
	}

	if receiver.HTLCOutputTxs[index] == nil {
		receiver.HTLCOutputTxs[index] = &HTLCOutputTxs{
			SenderCommitSuccessRevokeTx: revoke,
		}
	} else {
		receiver.HTLCOutputTxs[index].SenderCommitSuccessRevokeTx = revoke
	}

	return nil
}

func signSenderCommitSuccessTx(senderCommitSuccessTx *wire.MsgTx, commitScript []byte, commitOut *wire.TxOut, sender *User, receiver *User) ([][]byte, error) {

	signDesc := input.SignDescriptor{
		WitnessScript: commitScript,
		Output:        commitOut,
		HashType:      txscript.SigHashAll,
		SigHashes:     txscript.NewTxSigHashes(senderCommitSuccessTx),
		InputIndex:    0,
	}

	s := &SimpleSigner{
		PrivateKey: receiver.HTLCPrivateKey,
	}
	receiverSignature, err := s.SignOutputRaw(senderCommitSuccessTx, &signDesc)
	if err != nil {
		return nil, err
	}

	witnessStack := wire.TxWitness(make([][]byte, 3))
	witnessStack[0] = append(receiverSignature, byte(signDesc.HashType))
	witnessStack[1] = sender.HTLCPreImage[:]
	witnessStack[2] = signDesc.WitnessScript

	senderCommitSuccessTx.TxIn[0].Witness = witnessStack

	return witnessStack, nil
}

func generateSenderCommitSuccessRedeemTx(commitSuccessTx *wire.MsgTx, commitSuccessScript []byte, receiver *User) (*wire.MsgTx, error) {

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

func generateSenderCommitSuccessRevokeTx(commitSuccessTx *wire.MsgTx, commitSuccessScript []byte, receiver *User, revokePrivateKey *btcec.PrivateKey) (*wire.MsgTx, error) {

	revoke := wire.NewMsgTx(2)

	revoke.AddTxIn(&wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  commitSuccessTx.TxHash(),
			Index: 0,
		},
	})

	outputScript, err := txscript.PayToAddrScript(receiver.PayOutAddress)
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
