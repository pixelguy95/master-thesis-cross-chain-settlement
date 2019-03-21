package channel

import (
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/lightningnetwork/lnd/input"
	"github.com/pixelguy95/master-thesis-cross-chain-settlement/src/onchain_swaps_contract/bitcoin/customtransactions"
)

// GenerateRevocation generates a commit revocation transaction between two parties
func (c *Channel) GenerateRevocation(reverse bool, commitIndex uint, client *rpcclient.Client) error {

	var encumbered *User
	var unencumbered *User
	if !reverse {
		encumbered = c.Party1
		unencumbered = c.Party2
	} else {
		encumbered = c.Party2
		unencumbered = c.Party1
	}

	//TODO: Fix output amount to reflect channel balance
	revocationOutputValue := encumbered.Commits[commitIndex].Data.CommitTx.TxOut[0].Value

	if revocationOutputValue < int64(customtransactions.DefaultFee) {
		fmt.Println("Revocation output value too small, no revoation tx needed!")
		unencumbered.CommitRevokes = append(unencumbered.CommitRevokes, nil)
		return nil
	}

	revocationPrivateKey := input.DeriveRevocationPrivKey(unencumbered.FundingPrivateKey, encumbered.RevokationSecrets[0].CommitSecret)

	if !revocationPrivateKey.PubKey().IsEqual(encumbered.Commits[commitIndex].Data.RevocationPub) {
		fmt.Println("Incorrect revocation key")
		return errors.New("Incorrect revocation key")
	}

	s := &SimpleSigner{
		PrivateKey: revocationPrivateKey,
	}

	revoke := wire.NewMsgTx(2)

	commitInputPoint := wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  encumbered.Commits[commitIndex].Data.CommitTx.TxHash(),
			Index: 0,
		},
	}
	revoke.AddTxIn(&commitInputPoint)

	revoke.AddTxOut(wire.NewTxOut(revocationOutputValue-int64(customtransactions.DefaultFee), customtransactions.CreateP2PKHScript(unencumbered.PayOutAddress.ScriptAddress())))

	signDesc := input.SignDescriptor{
		WitnessScript: encumbered.Commits[commitIndex].Data.TimeLockedRevocationScript,
		Output:        encumbered.Commits[commitIndex].Data.CommitTx.TxOut[0],
		HashType:      txscript.SigHashAll,
		SigHashes:     txscript.NewTxSigHashes(revoke),
		InputIndex:    0,
	}

	revokeSig, err := s.SignOutputRaw(revoke, &signDesc)
	if err != nil {
		fmt.Println(err)
		return err
	}

	witness := make([][]byte, 3)
	witness[0] = append(revokeSig, byte(signDesc.HashType))
	witness[1] = []byte{1}
	witness[2] = encumbered.Commits[commitIndex].Data.TimeLockedRevocationScript

	revoke.TxIn[0].Witness = witness

	unencumbered.CommitRevokes = append(unencumbered.CommitRevokes, &CommitRevokeData{RevokeTx: revoke})

	return nil
}
