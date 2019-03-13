package channel

import (
	"errors"

	rpcutils "github.com/pixelguy95/btcd-rpcclient-extension/bitcoin"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
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

	// Loads the wallet used for funding
	clientWraper := rpcutils.New(client)
	clientWraper.UnloadAllWallets()
	clientWraper.LoadWallet(funder.WalletName)

	// Create funding transaction
	fundingTx := wire.NewMsgTx(2)
	_, out, error := input.GenFundingPkScript(party1.FundingPublicKey.PubKey().SerializeCompressed(), party1.FundingPublicKey.PubKey().SerializeCompressed(), funder.UserBalance)

	if error != nil {
		return nil, error
	}

	fundingTx.AddTxOut(out)

	channel = &Channel{
		Party1:         party1,
		Party2:         party2,
		ChannelBalance: funder.UserBalance,
		FundingTx:      fundingTx,
	}

	return channel, nil
}

// GenerateNewUserFromWallet generates a new channel user from a wallet
func GenerateNewUserFromWallet(walletName string, client *rpcclient.Client) (*User, error) {
	clientWraper := rpcutils.New(client)
	clientWraper.UnloadAllWallets()
	clientWraper.LoadWallet(walletName)

	address, _ := btcutil.DecodeAddress(clientWraper.GetNewP2PKHAddress(), config)
	privKey, _ := client.DumpPrivKey(address)

	user := &User{
		FundingPublicKey:  address,
		FundingPrivateKey: privKey,
		UserBalance:       0,
		Fundee:            true,
		WalletName:        walletName,
	}

	return user, nil
}
