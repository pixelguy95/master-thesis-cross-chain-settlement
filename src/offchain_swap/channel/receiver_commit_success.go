package channel

import (
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
	redeem, err := GenerateSecondlevelHTLCSpendTx(commitSuccess, witnessScript, receiver.PayOutAddress, receiver.HTLCPrivateKey, 0, DefaultRelativeLockTime)
	if err != nil {
		return err
	}

	// REVOKE
	revoke, err := GenerateSecondlevelHTLCSpendTx(commitSuccess, witnessScript, sender.PayOutAddress, revocationPrivateKey, 1, 0)
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
