package channel

import (
	"fmt"

	rpcutils "../../extensions/bitcoin"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil"
	"github.com/fatih/color"

	"github.com/btcsuite/btcd/wire"
)

var (
	// Yellow prints with yellow text
	Yellow = color.New(color.FgYellow)

	//Red prints with red text
	Red = color.New(color.FgRed)

	//Green prints with green text
	Green = color.New(color.FgGreen)
)

// User represents a party in a payment channel.
type User struct {
	/* The keys related to the funding of the channel */
	FundingPublicKey  *btcec.PublicKey
	FundingPrivateKey *btcec.PrivateKey

	/* Whenever a channel closes, all funds should go to this address */
	PayOutAddress btcutil.Address
	PayoutPubKey  *btcutil.AddressPubKey

	/* A construct that can be used to sign commits,
	derived from the funding transaction */
	FundingSigner *SimpleSigner

	/* The balance on this side of the channel */
	UserBalance int64

	/* True if this party wis the one who should fund the new channel */
	Fundee bool

	/* The name of the wallet used, only relevant when opening and closing a channel */
	WalletName string

	/* The pre image that should be used when generating commit secrets
	TODO: remove, this should be random for each commit */
	RevokePreImage []byte

	/* Array of all revokation secrets and some related data */
	RevokationSecrets []*CommitRevokationSecret

	/* Array of all commits so far in this channel */
	Commits []*CommitData

	/* Array of Commit-Revokes. Index 0 is the revoke for commit 0 etc...*/
	CommitRevokes []*CommitRevokeData
}

// CommitData stores commits and their related data
type CommitData struct {
	HasHTLCOutput bool
	Data          *CommitWithoutHTLC
}

// CommitRevokeData is a structure holding data related to revokes
type CommitRevokeData struct {
	RevokeTx *wire.MsgTx
}

// CommitRevokationSecret is part of what is needed to revoke a commit
type CommitRevokationSecret struct {
	CommitPoint  *btcec.PublicKey
	CommitSecret *btcec.PrivateKey
}

// Channel is a data type representing a channel
type Channel struct {
	Party1               *User
	Party2               *User
	ChannelBalance       int64
	FundingTx            *wire.MsgTx
	FundingWitnessScript []byte
	FundingMultiSigOut   *wire.TxOut
}

// PrintUser prints all info on user
func (user *User) PrintUser() {
	fmt.Printf("=== %s ===\n%x\n%x\n%t\n", user.WalletName, user.FundingPublicKey.SerializeCompressed(), user.FundingPrivateKey.Serialize(), user.Fundee)
}

// GenerateNewUserFromWallet generates a new channel user from a wallet
func GenerateNewUserFromWallet(walletName string, fundee bool, client *rpcclient.Client) (*User, error) {
	clientWraper := rpcutils.New(client)
	clientWraper.UnloadAllWallets()
	clientWraper.LoadWallet(walletName)

	payoutAddress, _ := btcutil.DecodeAddress(clientWraper.GetNewP2PKHAddress(), config)

	//Payout address for unencumbered
	payoutPubKey, _ := clientWraper.GetPubKey(payoutAddress.EncodeAddress())

	privKey, _ := btcec.NewPrivateKey(btcec.S256())

	signer := &SimpleSigner{
		PrivateKey: privKey,
	}

	user := &User{
		FundingPublicKey:  privKey.PubKey(),
		FundingPrivateKey: privKey,
		PayOutAddress:     payoutAddress,
		PayoutPubKey:      payoutPubKey,
		FundingSigner:     signer,
		UserBalance:       0,
		Fundee:            fundee,
		WalletName:        walletName,
		RevokePreImage:    []byte(walletName),
	}

	return user, nil
}
