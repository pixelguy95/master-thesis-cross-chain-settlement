package channel

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/pixelguy95/master-thesis-cross-chain-settlement/src/onchain_swaps_contract/bitcoin/customtransactions"

	rpcutils "../../extensions/bitcoin"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/lightningnetwork/lnd/input"
)

var config = &chaincfg.TestNet3Params

// OpenNewChannel opens a new channel between two parties, first user with fundee set will be considred to be the funder of the channel
// Only supports single funder
func OpenNewChannel(party1 *User, party2 *User, client *rpcclient.Client) (channel *Channel, err error) {

	/** CHANNEL FUNDING **/
	// Find out who will be considered for funding the channel
	var funder *User
	if party1.Fundee {
		funder = party1
	} else if party2.Fundee {
		funder = party2
	} else {
		return nil, errors.New("No funder for the channel")
	}

	clientWraper := rpcutils.New(client)

	// Create funding transaction
	fundingTx := wire.NewMsgTx(2)

	clientWraper.UnloadAllWallets()
	clientWraper.LoadWallet(party1.WalletName)
	party1PubKey, _ := clientWraper.GetPubKey(party1.FundingPublicKey.EncodeAddress())

	clientWraper.UnloadAllWallets()
	clientWraper.LoadWallet(party2.WalletName)
	party2PubKey, _ := clientWraper.GetPubKey(party2.FundingPublicKey.EncodeAddress())

	witnessScript, multiSigOut, error := input.GenFundingPkScript(party1PubKey.ScriptAddress(), party2PubKey.ScriptAddress(), funder.UserBalance)

	if error != nil {
		return nil, error
	}

	fundingTx.AddTxOut(multiSigOut)

	clientWraper.UnloadAllWallets()
	clientWraper.LoadWallet(funder.WalletName)
	fundingTx, _ = clientWraper.FundRawTransaction(fundingTx)
	fundingTx, _ = clientWraper.SignRawTransactionWithWallet(fundingTx)

	channel = &Channel{
		Party1:         party1,
		Party2:         party2,
		ChannelBalance: funder.UserBalance,
		FundingTx:      fundingTx,
	}

	commit1, _, err := channel.CreateStaticCommits(client)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// Generate a signature for their version of the initial commitment
	// transaction.
	signDesc := input.SignDescriptor{
		WitnessScript: witnessScript,
		Output:        multiSigOut,
		HashType:      txscript.SigHashAll,
		SigHashes:     txscript.NewTxSigHashes(commit1.CommitTx),
		InputIndex:    0,
	}

	s := &SimpleSigner{
		PrivateKey: party1.FundingPrivateKey.PrivKey,
	}

	signature1, err := s.SignOutputRaw(commit1.CommitTx, &signDesc)
	if err != nil {
		fmt.Println(err)
		return nil, error
	}

	s = &SimpleSigner{
		PrivateKey: party2.FundingPrivateKey.PrivKey,
	}

	signature2, err := s.SignOutputRaw(commit1.CommitTx, &signDesc)
	if err != nil {
		fmt.Println(err)
		return nil, error
	}

	fmt.Printf("\nSignature1\n%x\n", signature1)
	fmt.Printf("\nSignature2\n%x\n", signature2)

	signature1 = append(signature1, byte(txscript.SigHashAll))
	signature2 = append(signature2, byte(txscript.SigHashAll))

	witness := input.SpendMultiSig(witnessScript, party1PubKey.ScriptAddress(), signature1, party2PubKey.ScriptAddress(), signature2)
	commit1.CommitTx.TxIn[0].Witness = witness

	buf := new(bytes.Buffer)
	commit1.CommitTx.Serialize(buf)
	fmt.Printf("\nCommit:\n%x\n\n", buf)

	revocation := input.DeriveRevocationPrivKey(party2.FundingPrivateKey.PrivKey, party1.RevokationSecrets[0].CommitSecret)

	if revocation.PubKey().IsEqual(commit1.RevocationPub) {
		fmt.Println("Correct revocation key")
		fmt.Printf("%x\n", revocation.PubKey().SerializeCompressed())
	} else {
		fmt.Println("Incorrect revocation key")
	}

	s = &SimpleSigner{
		PrivateKey: revocation,
	}

	//Revoke payout address
	changeTo := clientWraper.GetNewP2PKHAddress()
	changeAddress, _ := btcutil.DecodeAddress(changeTo, config)

	revoke := wire.NewMsgTx(2)

	commitInputPoint := wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  commit1.CommitTx.TxHash(),
			Index: 0,
		},
	}
	revoke.AddTxIn(&commitInputPoint)
	revoke.AddTxOut(wire.NewTxOut(party1.UserBalance-4000, customtransactions.CreateP2PKHScript(changeAddress.ScriptAddress())))

	signDesc = input.SignDescriptor{
		WitnessScript: commit1.TimeLockedRevocationScript,
		Output:        commit1.CommitTx.TxOut[0],
		HashType:      txscript.SigHashAll,
		SigHashes:     txscript.NewTxSigHashes(revoke),
		InputIndex:    0,
	}

	revokeSig, err := s.SignOutputRaw(revoke, &signDesc)
	if err != nil {
		fmt.Println(err)
		return nil, error
	}

	witness = make([][]byte, 3)
	witness[0] = append(revokeSig, byte(signDesc.HashType))
	witness[1] = []byte{1}
	witness[2] = commit1.TimeLockedRevocationScript

	revoke.TxIn[0].Witness = witness

	buf = new(bytes.Buffer)
	revoke.Serialize(buf)
	fmt.Printf("\nRevoke:\n%x\n\n", buf)

	return channel, nil
}
