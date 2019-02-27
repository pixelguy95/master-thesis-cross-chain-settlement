package channel

import (
	"errors"

	rpcutils "github.com/pixelguy95/btcd-rpcclient-extension/bitcoin"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
)

// OpenNewChannel opens a new channel between two parties, first user with fundee set will be considred to be the funder of the channel
// Only supports single funder
func OpenNewChannel(party1 *User, party2 *User, amount uint64, client *rpcclient.Client) (channel *Channel, err error) {

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

	fundingTx := wire.NewMsgTx(2)

	return nil, nil
}
