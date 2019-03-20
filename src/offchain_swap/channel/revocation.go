package channel

import (
	"errors"
	"fmt"

	rpcutils "../../extensions/bitcoin"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/lightningnetwork/lnd/input"
	"github.com/pixelguy95/master-thesis-cross-chain-settlement/src/onchain_swaps_contract/bitcoin/customtransactions"
)

// GenerateRevocation generates a commit revocation transaction between two parties
func GenerateRevocation(encumbered, unencumbered *User, commitIndex uint, client *rpcclient.Client) (revocationTx *wire.MsgTx, err error) {

	clientWraper := rpcutils.New(client)
	revocation := input.DeriveRevocationPrivKey(unencumbered.FundingPrivateKey.PrivKey, encumbered.RevokationSecrets[0].CommitSecret)

	if !revocation.PubKey().IsEqual(encumbered.Commits[commitIndex].Data.RevocationPub) {
		fmt.Println("Incorrect revocation key")
		return nil, errors.New("Incorrect revocation key")
	}

	s := &SimpleSigner{
		PrivateKey: revocation,
	}

	//Revoke payout address
	clientWraper.UnloadAllWallets()
	clientWraper.LoadWallet(unencumbered.WalletName)
	changeTo := clientWraper.GetNewP2PKHAddress()
	changeAddress, _ := btcutil.DecodeAddress(changeTo, config)

	fmt.Println(changeAddress.EncodeAddress())

	revoke := wire.NewMsgTx(2)

	commitInputPoint := wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  encumbered.Commits[commitIndex].Data.CommitTx.TxHash(),
			Index: 0,
		},
	}
	revoke.AddTxIn(&commitInputPoint)

	//TODO: Fix output amount to reflect channel balance
	revocationOutputValue := encumbered.Commits[commitIndex].Data.CommitTx.TxOut[0].Value

	if revocationOutputValue < int64(customtransactions.DefaultFee) {
		fmt.Println("Revocation output value too small, no revoation tx needed!")
		return nil, nil
	}
	revoke.AddTxOut(wire.NewTxOut(revocationOutputValue-int64(customtransactions.DefaultFee), customtransactions.CreateP2PKHScript(changeAddress.ScriptAddress())))

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
		return nil, err
	}

	witness := make([][]byte, 3)
	witness[0] = append(revokeSig, byte(signDesc.HashType))
	witness[1] = []byte{1}
	witness[2] = encumbered.Commits[commitIndex].Data.TimeLockedRevocationScript

	revoke.TxIn[0].Witness = witness

	return revoke, nil
}
