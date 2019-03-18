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

	_, out, error := input.GenFundingPkScript(party1PubKey.ScriptAddress(), party2PubKey.ScriptAddress(), funder.UserBalance)

	if error != nil {
		return nil, error
	}

	fundingTx.AddTxOut(out)

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

	buf := new(bytes.Buffer)
	commit1.Serialize(buf)
	fmt.Printf("\nCommit:\n%x\n\n", buf)

	return channel, nil
}
