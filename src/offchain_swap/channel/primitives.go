package channel

import (
	"fmt"

	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

// User represents a party in a payment channel.
type User struct {
	FundingPublicKey  btcutil.Address
	FundingPrivateKey *btcutil.WIF
	UserBalance       int64
	Fundee            bool
	WalletName        string
}

// Channel is a data type representing a channel
type Channel struct {
	Party1         *User
	Party2         *User
	ChannelBalance int64
	FundingTx      *wire.MsgTx
}

// PrintUser prints all info on user
func (user *User) PrintUser() {
	fmt.Printf("=== %s ===\n%s\n%s\n%t\n", user.WalletName, user.FundingPublicKey.String(), user.FundingPrivateKey.String(), user.Fundee)
}
