package channel

import (
	"github.com/btcsuite/btcutil"
)

// User represents a party in a payment channel.
type User = struct {
	FundingPublicKey  *btcutil.Address
	FundingPrivateKey *btcutil.WIF
	UserBalance       uint
	Fundee            bool
	WalletName        string
}

// Channel is a data type representing a channel
type Channel = struct {
	Party1         User
	Party2         User
	ChannelBalance uint
}
