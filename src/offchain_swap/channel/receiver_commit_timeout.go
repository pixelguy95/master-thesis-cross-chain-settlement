package channel

import (
	"log"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/lightningnetwork/lnd/input"
	"github.com/pixelguy95/master-thesis-cross-chain-settlement/src/onchain_swaps_contract/bitcoin/customtransactions"
)

func (c *Channel) GenerateReceiverCommitTimeoutTx(index, cltvLocktime uint32, sender *User, receiver *User) error {

	log.Println("Generating receiver commit timeout for", sender.Name)

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

	// REDEEM
	var senderPayout []byte
	if sender.IsLitecoinUser {
		senderPayout = btcutil.Hash160(sender.LtcPayoutPubKey.SerializeCompressed())
	} else {
		senderPayout = sender.PayoutPubKey.ScriptAddress()
	}
	redeem, err := GenerateSecondlevelHTLCSpendTx(commitTimeout, witnessScript, senderPayout, sender.HTLCPrivateKey, 0, DefaultRelativeLockTime)
	if err != nil {
		return err
	}

	// REVOKE
	var receiverPayout []byte
	if receiver.IsLitecoinUser {
		receiverPayout = btcutil.Hash160(receiver.LtcPayoutPubKey.SerializeCompressed())
	} else {
		receiverPayout = receiver.PayoutPubKey.ScriptAddress()
	}
	revoke, err := GenerateSecondlevelHTLCSpendTx(commitTimeout, witnessScript, receiverPayout, revocationPrivateKey, 1, 0)
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
