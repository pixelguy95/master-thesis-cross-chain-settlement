package channel

import (
	"log"

	"github.com/btcsuite/btcutil"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/lightningnetwork/lnd/input"
	"github.com/pixelguy95/master-thesis-cross-chain-settlement/src/onchain_swaps_contract/bitcoin/customtransactions"
)

// GenerateSenderCommitSuccessTx consumes a commit with secret reveal
func (c *Channel) GenerateSenderCommitSuccessTx(index uint32, sender *User, receiver *User) error {

	log.Println("Generating sender commit success for", receiver.Name)

	revocationPrivateKey := input.DeriveRevocationPrivKey(sender.FundingPrivateKey, receiver.RevokationSecrets[index].CommitSecret)
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
	var receiverPayout []byte
	if receiver.IsLitecoinUser {
		receiverPayout = btcutil.Hash160(receiver.LtcPayoutPubKey.SerializeCompressed())
	} else {
		receiverPayout = receiver.PayoutPubKey.ScriptAddress()
	}
	redeem, err := GenerateSecondlevelHTLCSpendTx(commitSuccess, witnessScript, receiverPayout, receiver.HTLCPrivateKey, 0, DefaultRelativeLockTime)
	if err != nil {
		return err
	}

	// REVOKE
	var senderPayout []byte
	if sender.IsLitecoinUser {
		senderPayout = btcutil.Hash160(sender.LtcPayoutPubKey.SerializeCompressed())
	} else {
		senderPayout = sender.PayoutPubKey.ScriptAddress()
	}
	revoke, err := GenerateSecondlevelHTLCSpendTx(commitSuccess, witnessScript, senderPayout, revocationPrivateKey, 1, 0)
	if err != nil {
		return err
	}

	if receiver.HTLCOutputTxs[index] == nil {
		receiver.HTLCOutputTxs[index] = &HTLCOutputTxs{
			SenderCommitSuccessScript:   witnessScript,
			SenderCommitSuccessTx:       commitSuccess,
			SenderCommitSuccessRedeemTx: redeem,
		}
	} else {
		receiver.HTLCOutputTxs[index].SenderCommitSuccessScript = witnessScript
		receiver.HTLCOutputTxs[index].SenderCommitSuccessTx = commitSuccess
		receiver.HTLCOutputTxs[index].SenderCommitSuccessRedeemTx = redeem
	}

	if sender.HTLCOutputTxs[index] == nil {
		sender.HTLCOutputTxs[index] = &HTLCOutputTxs{
			SenderCommitSuccessRevokeTx: revoke,
		}
	} else {
		sender.HTLCOutputTxs[index].SenderCommitSuccessRevokeTx = revoke
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
