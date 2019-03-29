package channel

import (
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/lightningnetwork/lnd/input"
	"github.com/pixelguy95/master-thesis-cross-chain-settlement/src/onchain_swaps_contract/bitcoin/customtransactions"
)

// GenerateSenderCommitTimeoutTx generates a timeout transaction that spends the htlc output from the senders commit
func (c *Channel) GenerateSenderCommitTimeoutTx(index, cltvLocktime uint32, sender *User, receiver *User) error {

	//_, _, revocationPub, _ := GenerateRevokePubKey(sender.RevokePreImage, receiver.FundingPublicKey)

	revocationPrivateKey := input.DeriveRevocationPrivKey(receiver.FundingPrivateKey, sender.RevokationSecrets[index].CommitSecret)
	revocationPub := revocationPrivateKey.PubKey()

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

	htlcScript := sender.Commits[index].HTLCOutScript
	signSenderCommitTimeoutTx(commitTimeout, htlcScript, sender.Commits[index].CommitTx.TxOut[2], sender, receiver)

	// REDEEM
	redeem, err := GenerateSecondlevelHTLCSpendTx(commitTimeout, witnessScript, sender.PayOutAddress, sender.HTLCPrivateKey, 0, DefaultRelativeLockTime)
	if err != nil {
		return err
	}

	// REVOKE
	revoke, err := GenerateSecondlevelHTLCSpendTx(commitTimeout, witnessScript, receiver.PayOutAddress, revocationPrivateKey, 1, 0)
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
