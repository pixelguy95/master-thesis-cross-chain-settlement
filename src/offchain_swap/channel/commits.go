package channel

import (
	"errors"
	"log"
	"time"

	"github.com/pixelguy95/master-thesis-cross-chain-settlement/src/onchain_swaps_contract/bitcoin/customtransactions"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	"github.com/lightningnetwork/lnd/input"
)

// Settle creates new commits with whatever funds each party had on their side of the channel. The commits will not have any HTLC outputs
func (channel *Channel) Settle(client *rpcclient.Client) error {

	log.Println("Generating settle commits")

	if channel.Party1.UserBalance < 0 || channel.Party2.UserBalance < 0 {
		return errors.New("Channel balance is impossible, Negative values")
	}

	//PARTY 1 COMMIT
	commitParty1, err := channel.createSettleCommit(channel.Party1, channel.Party2, client)
	if err != nil {
		return err
	}

	channel.Party1.Commits = append(channel.Party1.Commits, commitParty1)

	//**************************************************************************************************
	//PARTY2 COMMIT
	commitParty2, err := channel.createSettleCommit(channel.Party2, channel.Party1, client)
	if err != nil {
		return err
	}

	channel.Party2.Commits = append(channel.Party2.Commits, commitParty2)

	channel.SignCommitsTx(uint(len(channel.Party1.Commits)) - 1)
	channel.GenerateCommitSpends(uint(len(channel.Party1.Commits) - 1))
	channel.GenerateRevocations(uint(len(channel.Party1.Commits) - 1))

	return nil
}

// createStaticCommit creates a single commit, no htlc
func (channel *Channel) createSettleCommit(encumbered *User, unencumbered *User, client *rpcclient.Client) (*CommitData, error) {

	//Create revoke key on party1 commit side
	commitPoint, commitSecret, revocationPub, _ := GenerateRevokePubKey(encumbered.RevokePreImage, unencumbered.FundingPublicKey)

	encumbered.RevokationSecrets = append(encumbered.RevokationSecrets, &CommitRevokationSecret{CommitPoint: commitPoint, CommitSecret: commitSecret})

	commitTx := wire.NewMsgTx(2)

	fundingTxIn := wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  channel.FundingTx.TxHash(),
			Index: 0,
		},
	}

	commitTx.AddTxIn(&fundingTxIn)

	//Delayed script to self, or via revocation
	var encumberedPayout *btcec.PublicKey
	if encumbered.IsLitecoinUser {
		encumberedPayout, _ = btcec.ParsePubKey(encumbered.LtcPayoutPubKey.SerializeCompressed(), btcec.S256())
	} else {
		encumberedPayout = encumbered.PayoutPubKey.PubKey()
	}
	ourRedeemScript, out, _ := EncumberedOutput(encumberedPayout, revocationPub, encumbered.UserBalance)
	commitTx.AddTxOut(out)

	//Unencumbered payout to other party
	var unencumberedPayout *btcec.PublicKey
	if encumbered.IsLitecoinUser {
		unencumberedPayout, _ = btcec.ParsePubKey(unencumbered.LtcPayoutPubKey.SerializeCompressed(), btcec.S256())
	} else {
		unencumberedPayout = unencumbered.PayoutPubKey.PubKey()
	}
	theirWitnessKeyHash, uout, _ := UnencumberedOutput(unencumberedPayout, unencumbered.UserBalance)
	commitTx.AddTxOut(uout)

	if encumbered.UserBalance < 2000 {
		commitTx.TxOut[0].Value = encumbered.UserBalance
		commitTx.TxOut[1].Value = unencumbered.UserBalance - 2000
	}

	return &CommitData{
		CommitTx:                   commitTx,
		TimeLockedRevocationScript: ourRedeemScript,
		UnencumberedScript:         theirWitnessKeyHash,
		RevocationPub:              revocationPub,
		HasHTLCOutput:              false}, nil
}

// EncumberedOutput produces whatever is needed for a encumbered timeout output in a commit transaction
func EncumberedOutput(encumberedPayoutKey, revocationPubKey *btcec.PublicKey, balance int64) (script []byte, output *wire.TxOut, err error) {

	//Delayed script to self, or via revocation
	ourRedeemScript, err := input.CommitScriptToSelf(DefaultRelativeLockTime, encumberedPayoutKey, revocationPubKey)
	if err != nil {
		return nil, nil, err
	}

	ourRedeemScriptWitnessHash, err := input.WitnessScriptHash(ourRedeemScript)
	if err != nil {
		return nil, nil, err
	}

	out := &wire.TxOut{
		PkScript: ourRedeemScriptWitnessHash,
		Value:    int64(balance - 2000),
	}

	return ourRedeemScript, out, nil
}

// UnencumberedOutput produces whatever is needed for a unencumbered output in a commit transaction
func UnencumberedOutput(unencumberedPauoutPubKey *btcec.PublicKey, amount int64) (witnessKeyHash []byte, out *wire.TxOut, err error) {

	theirWitnessKeyHash, err := input.CommitScriptUnencumbered(unencumberedPauoutPubKey)
	if err != nil {
		return nil, nil, err
	}

	out = &wire.TxOut{
		PkScript: theirWitnessKeyHash,
		Value:    amount,
	}

	return theirWitnessKeyHash, out, nil
}

// SendCommit creates all new commit tx and related data that represents sending money
func (channel *Channel) Pay(sd *SendDescriptor) error {

	log.Printf("Sending %d satoshis from %s to %s", sd.Balance, sd.Sender.Name, sd.Receiver.Name)

	if !channel.Party1.Equals(sd.Sender) && !channel.Party2.Equals(sd.Sender) {
		return errors.New("Sender is not in this channel")
	} else if !channel.Party1.Equals(sd.Receiver) && !channel.Party2.Equals(sd.Receiver) {
		return errors.New("Receiver is not in this channel")
	} else if sd.Receiver.Equals(sd.Sender) {
		return errors.New("Sender and receiver can't be the same user")
	}

	if sd.Sender.UserBalance < int64(sd.Balance) {
		return errors.New("Sender doesn't have enough money to send")
	}

	sd.Sender.UserBalance -= sd.Balance

	senderCommit, _ := channel.createSenderCommit(sd)
	sd.Sender.Commits = append(sd.Sender.Commits, senderCommit)

	receiverCommit, _ := channel.createReceiverCommit(sd)
	sd.Receiver.Commits = append(sd.Receiver.Commits, receiverCommit)

	index := uint32(len(sd.Receiver.Commits) - 1)

	channel.SignCommitsTx(uint(len(channel.Party1.Commits)) - 1)
	channel.GenerateCommitSpends(uint(len(channel.Party1.Commits) - 1))
	channel.GenerateRevocations(uint(len(channel.Party1.Commits) - 1))

	/*TIMEOUT*/
	cltvExpiry := uint32(time.Now().Unix() + (60 * 10))
	//cltvExpiry := uint32(1553763911)

	err := channel.GenerateSenderCommitTimeoutTx(index, cltvExpiry, sd.Sender, sd.Receiver)
	if err != nil {
		log.Fatal(err)
	}

	err = channel.GenerateSenderCommitSuccessTx(index, sd)
	if err != nil {
		log.Fatal(err)
	}

	err = channel.GenerateReceiverCommitTimeoutTx(index, cltvExpiry, sd.Sender, sd.Receiver)
	if err != nil {
		log.Fatal(err)
	}

	err = channel.GenerateReceiverCommitSuccessTx(index, sd)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func (channel *Channel) createSenderCommit(sd *SendDescriptor) (*CommitData, error) {

	commitPoint, commitSecret, revocationPub, _ := GenerateRevokePubKey(sd.Sender.RevokePreImage, sd.Receiver.FundingPublicKey)
	sd.Sender.RevokationSecrets = append(sd.Sender.RevokationSecrets, &CommitRevokationSecret{CommitPoint: commitPoint, CommitSecret: commitSecret})

	commitTx := wire.NewMsgTx(2)

	fundingTxIn := wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  channel.FundingTx.TxHash(),
			Index: 0,
		},
	}

	commitTx.AddTxIn(&fundingTxIn)

	//Delayed script to self, or via revocation
	var senderPayout *btcec.PublicKey
	if sd.Sender.IsLitecoinUser {
		senderPayout, _ = btcec.ParsePubKey(sd.Sender.LtcPayoutPubKey.SerializeCompressed(), btcec.S256())
	} else {
		senderPayout = sd.Sender.PayoutPubKey.PubKey()
	}

	ourRedeemScript, out, _ := EncumberedOutput(senderPayout, revocationPub, sd.Sender.UserBalance)
	commitTx.AddTxOut(out)

	//Unencumbered payout to other party
	var receiverPayout *btcec.PublicKey
	if sd.Receiver.IsLitecoinUser {
		receiverPayout, _ = btcec.ParsePubKey(sd.Receiver.LtcPayoutPubKey.SerializeCompressed(), btcec.S256())
	} else {
		receiverPayout = sd.Receiver.PayoutPubKey.PubKey()
	}

	theirWitnessKeyHash, uout, _ := UnencumberedOutput(receiverPayout, sd.Receiver.UserBalance)
	commitTx.AddTxOut(uout)

	if sd.Sender.UserBalance < int64(customtransactions.DefaultFee) {
		commitTx.TxOut[0].Value = sd.Sender.UserBalance
		commitTx.TxOut[1].Value = sd.Receiver.UserBalance - int64(customtransactions.DefaultFee)
	}

	//HTLC output
	htclOutPutScript, err := input.SenderHTLCScript(sd.Sender.HTLCPublicKey, sd.Receiver.HTLCPublicKey, revocationPub, sd.PaymentHash[:])
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	witnessScripthash, _ := input.WitnessScriptHash(htclOutPutScript)

	HTLCOut := &wire.TxOut{
		PkScript: witnessScripthash,
		Value:    int64(sd.Balance),
	}

	commitTx.AddTxOut(HTLCOut)

	return &CommitData{
		CommitTx:                   commitTx,
		TimeLockedRevocationScript: ourRedeemScript,
		UnencumberedScript:         theirWitnessKeyHash,
		RevocationPub:              revocationPub,
		HasHTLCOutput:              true,
		IsSender:                   true,
		HTLCOutScript:              htclOutPutScript}, nil
}

func (channel *Channel) createReceiverCommit(sd *SendDescriptor) (*CommitData, error) {

	commitPoint, commitSecret, revocationPub, _ := GenerateRevokePubKey(sd.Receiver.RevokePreImage, sd.Sender.FundingPublicKey)
	sd.Receiver.RevokationSecrets = append(sd.Receiver.RevokationSecrets, &CommitRevokationSecret{CommitPoint: commitPoint, CommitSecret: commitSecret})

	commitTx := wire.NewMsgTx(2)

	fundingTxIn := wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  channel.FundingTx.TxHash(),
			Index: 0,
		},
	}
	commitTx.AddTxIn(&fundingTxIn)

	//Delayed script to self, or via revocation
	var receiverPayout *btcec.PublicKey
	if sd.Receiver.IsLitecoinUser {
		receiverPayout, _ = btcec.ParsePubKey(sd.Receiver.LtcPayoutPubKey.SerializeCompressed(), btcec.S256())
	} else {
		receiverPayout = sd.Receiver.PayoutPubKey.PubKey()
	}
	ourRedeemScript, out, _ := EncumberedOutput(receiverPayout, revocationPub, sd.Receiver.UserBalance)
	commitTx.AddTxOut(out)

	//Unencumbered payout to other party
	var senderPayout *btcec.PublicKey
	if sd.Sender.IsLitecoinUser {
		senderPayout, _ = btcec.ParsePubKey(sd.Sender.LtcPayoutPubKey.SerializeCompressed(), btcec.S256())
	} else {
		senderPayout = sd.Sender.PayoutPubKey.PubKey()
	}
	theirWitnessKeyHash, uout, _ := UnencumberedOutput(senderPayout, sd.Sender.UserBalance)
	commitTx.AddTxOut(uout)

	if sd.Receiver.UserBalance < int64(customtransactions.DefaultFee) {
		commitTx.TxOut[0].Value = sd.Receiver.UserBalance
		commitTx.TxOut[1].Value = sd.Sender.UserBalance - int64(customtransactions.DefaultFee)
	}

	cltvExpiry := time.Now().Unix() + (60 * 10)
	//cltvExpiry := uint32(1553763911)

	//HTLC output
	htclOutPutScript, err := input.ReceiverHTLCScript(uint32(cltvExpiry), sd.Sender.HTLCPublicKey, sd.Receiver.HTLCPublicKey, revocationPub, sd.PaymentHash[:])
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	witnessScripthash, _ := input.WitnessScriptHash(htclOutPutScript)

	HTLCOut := &wire.TxOut{
		PkScript: witnessScripthash,
		Value:    int64(sd.Balance),
	}

	commitTx.AddTxOut(HTLCOut)

	return &CommitData{
		CommitTx:                   commitTx,
		TimeLockedRevocationScript: ourRedeemScript,
		UnencumberedScript:         theirWitnessKeyHash,
		RevocationPub:              revocationPub,
		HasHTLCOutput:              true,
		IsSender:                   false,
		HTLCOutScript:              htclOutPutScript}, nil
}
