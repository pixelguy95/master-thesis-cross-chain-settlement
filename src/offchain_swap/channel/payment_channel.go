package channel

import (
	"bytes"
	"errors"
	"fmt"

	rpcutils "../../extensions/bitcoin"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
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
		Party1:               party1,
		Party2:               party2,
		ChannelBalance:       funder.UserBalance,
		FundingTx:            fundingTx,
		FundingWitnessScript: witnessScript,
		FundingMultiSigOut:   multiSigOut,
	}

	_, _, err = channel.CreateStaticCommits(client)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	channel.SignCommitTx(false, 0, client)
	channel.SignCommitTx(true, 0, client)

	/*
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
	*/

	buf := new(bytes.Buffer)
	channel.Party1.Commits[0].Data.CommitTx.Serialize(buf)
	fmt.Printf("\nCommit1:\n%x\n\n", buf)

	buf = new(bytes.Buffer)
	channel.Party2.Commits[0].Data.CommitTx.Serialize(buf)
	fmt.Printf("\nCommit2:\n%x\n\n", buf)

	revoke, err := GenerateRevocation(party1, party2, 0, client)
	if err != nil {
		fmt.Println(err)
		return nil, error
	}

	if revoke != nil {
		buf = new(bytes.Buffer)
		revoke.Serialize(buf)
		fmt.Printf("\nRevoke1:\n%x\n\n", buf)
	}

	revoke, err = GenerateRevocation(party2, party1, 0, client)
	if err != nil {
		fmt.Println(err)
		return nil, error
	}

	if revoke != nil {
		buf = new(bytes.Buffer)
		revoke.Serialize(buf)
		fmt.Printf("\nRevoke2:\n%x\n\n", buf)
	}

	return channel, nil
}
