package channel

import (
	"errors"
	"log"

	"github.com/btcsuite/btcutil"

	"github.com/btcsuite/btcd/btcec"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/lightningnetwork/lnd/input"
	"github.com/pixelguy95/master-thesis-cross-chain-settlement/src/onchain_swaps_contract/bitcoin/customtransactions"
)

// GenerateRevocations generates commit revocation transactions between two parties in a channel
func (c *Channel) GenerateRevocations(commitIndex uint) error {

	err := buildRevocation(commitIndex, c.Party1, c.Party2)
	if err != nil {
		log.Fatal(err)
	}

	err = buildRevocation(commitIndex, c.Party2, c.Party1)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func buildRevocation(commitIndex uint, encumbered *User, unencumbered *User) error {
	log.Printf("Building revocation for %s", unencumbered.Name)

	//TODO: Fix output amount to reflect channel balance
	revocationOutputValue := encumbered.Commits[commitIndex].CommitTx.TxOut[0].Value

	if revocationOutputValue < int64(customtransactions.DefaultFee) {
		log.Println("Revocation output value too small, no revoation tx needed!")
		unencumbered.CommitRevokes = append(unencumbered.CommitRevokes, nil)
		return nil
	}

	revocationPrivateKey := input.DeriveRevocationPrivKey(unencumbered.FundingPrivateKey, encumbered.RevokationSecrets[commitIndex].CommitSecret)

	if !revocationPrivateKey.PubKey().IsEqual(encumbered.Commits[commitIndex].RevocationPub) {
		log.Println("Incorrect revocation key")
		return errors.New("Incorrect revocation key")
	}

	s := &SimpleSigner{
		PrivateKey: revocationPrivateKey,
	}

	revoke := wire.NewMsgTx(2)

	commitInputPoint := wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  encumbered.Commits[commitIndex].CommitTx.TxHash(),
			Index: 0,
		},
	}
	revoke.AddTxIn(&commitInputPoint)

	var unencumberedPayoutScriptAddress []byte
	if unencumbered.IsLitecoinUser {
		unencumberedPayoutScriptAddress = btcutil.Hash160(unencumbered.LtcPayoutPubKey.SerializeCompressed())
	} else {
		unencumberedPayoutScriptAddress = unencumbered.PayOutAddress.ScriptAddress()
	}

	revoke.AddTxOut(wire.NewTxOut(revocationOutputValue-int64(customtransactions.DefaultFee), customtransactions.CreateP2PKHScript(unencumberedPayoutScriptAddress)))

	signDesc := input.SignDescriptor{
		WitnessScript: encumbered.Commits[commitIndex].TimeLockedRevocationScript,
		Output:        encumbered.Commits[commitIndex].CommitTx.TxOut[0],
		HashType:      txscript.SigHashAll,
		SigHashes:     txscript.NewTxSigHashes(revoke),
		InputIndex:    0,
	}

	revokeSig, err := s.SignOutputRaw(revoke, &signDesc)
	if err != nil {
		log.Fatal(err)
		return err
	}

	witness := make([][]byte, 3)
	witness[0] = append(revokeSig, byte(signDesc.HashType))
	witness[1] = []byte{1}
	witness[2] = encumbered.Commits[commitIndex].TimeLockedRevocationScript

	revoke.TxIn[0].Witness = witness

	unencumbered.CommitRevokes = append(unencumbered.CommitRevokes, &CommitRevokeData{RevokeTx: revoke})

	return nil
}

// GenerateRevokePubKey generates a revokation key
func GenerateRevokePubKey(preimage []byte, basePub *btcec.PublicKey) (commitPoint *btcec.PublicKey, commitSecret *btcec.PrivateKey, revocationPub *btcec.PublicKey, err error) {

	commitSecret, commitPoint = btcec.PrivKeyFromBytes(btcec.S256(), preimage)
	revocationPub = input.DeriveRevocationPubkey(basePub, commitPoint)

	return commitPoint, commitSecret, revocationPub, nil
}
