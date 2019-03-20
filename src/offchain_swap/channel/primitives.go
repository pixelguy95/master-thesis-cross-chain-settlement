package channel

import (
	"fmt"

	rpcutils "../../extensions/bitcoin"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/rpcclient"

	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

// User represents a party in a payment channel.
type User struct {
	/* The keys related to the funding of the channel */
	FundingPublicKey  btcutil.Address
	FundingPrivateKey *btcutil.WIF

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
}

// CommitData stores commits and their related data
type CommitData struct {
	HasHTLCOutput bool
	Data          *CommitWithoutHTLC
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
	fmt.Printf("=== %s ===\n%s\n%s\n%t\n", user.WalletName, user.FundingPublicKey.String(), user.FundingPrivateKey.String(), user.Fundee)
}

// GenerateNewUserFromWallet generates a new channel user from a wallet
func GenerateNewUserFromWallet(walletName string, client *rpcclient.Client) (*User, error) {
	clientWraper := rpcutils.New(client)
	clientWraper.UnloadAllWallets()
	clientWraper.LoadWallet(walletName)

	address, _ := btcutil.DecodeAddress(clientWraper.GetNewP2PKHAddress(), config)
	privKey, _ := client.DumpPrivKey(address)

	signer := &SimpleSigner{
		PrivateKey: privKey.PrivKey,
	}

	user := &User{
		FundingPublicKey:  address,
		FundingPrivateKey: privKey,
		FundingSigner:     signer,
		UserBalance:       0,
		Fundee:            true,
		WalletName:        walletName,
		RevokePreImage:    []byte(walletName),
	}

	return user, nil
}
