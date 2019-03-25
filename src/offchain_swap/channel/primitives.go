package channel

import (
	"fmt"

	rpcutils "../../extensions/bitcoin"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil"
	"github.com/fatih/color"

	"github.com/btcsuite/btcd/wire"

	"crypto/sha256"
)

var (
	// Yellow prints with yellow text
	Yellow = color.New(color.FgYellow)

	//Red prints with red text
	Red = color.New(color.FgRed)

	//Green prints with green text
	Green = color.New(color.FgGreen)

	//DefaultRelativeLockTime is the default number of blocks to wait before a commit is spendable
	DefaultRelativeLockTime = uint32(3)
)

// User represents a party in a payment channel.
type User struct {
	Name string

	/* The keys related to the funding of the channel */
	FundingPublicKey  *btcec.PublicKey
	FundingPrivateKey *btcec.PrivateKey

	/* HTLC keys */
	HTLCPublicKey  *btcec.PublicKey
	HTLCPrivateKey *btcec.PrivateKey

	/* Whenever a channel closes, all funds should go to this address */
	PayOutAddress    btcutil.Address
	PayoutPubKey     *btcutil.AddressPubKey
	PayoutPrivateKey *btcutil.WIF

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

	/* The pre image used for htlc outputs
	TODO: remove, this should be random for each commit*/
	HTLCPreImage    []byte
	HTLCPaymentHash [sha256.Size]byte

	/* Array of all revokation secrets and some related data */
	RevokationSecrets []*CommitRevokationSecret

	/* Array of all commits so far in this channel */
	Commits []*CommitData

	/* Array of Commit-Revokes. Index 0 is the revoke for commit 0 etc...
	NOTE: These are spending the commit tx from the other party.*/
	CommitRevokes []*CommitRevokeData

	/* Commit spend */
	CommitSpends []*CommitSpendData
}

// CommitData is a representation of a commit transaction with some companion data
type CommitData struct {

	// The base commit TX
	CommitTx *wire.MsgTx

	// The scripts in the timelocked return / revocation and the unencumbered outputs.
	TimeLockedRevocationScript []byte
	UnencumberedScript         []byte

	// The public key used in the timelocked return / revocation output.
	RevocationPub *btcec.PublicKey

	// This is set to true if the commit contians a HTLC output
	HasHTLCOutput bool

	/* BELOW THIS POINT IS ONLY RELEVANT IF ABOVE IS TRUE */

	// True if the holder of this commit is the sender
	IsSender bool

	// The script in the htlc output
	HTLCOutScript []byte
}

// CommitRevokeData is a structure holding data related to revokes
type CommitRevokeData struct {
	RevokeTx *wire.MsgTx
}

// CommitSpendData is a structure holding data related to spending the timelocked output in the base commit
type CommitSpendData struct {
	CommitSpend *wire.MsgTx
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

// SendDescriptor represents how a transaction should be constructed
type SendDescriptor struct {
	Sender   *User
	Receiver *User
	Balance  int64
}

// PrintUser prints all info on user
func (user *User) PrintUser() {
	fmt.Printf("=== %s ===\n%x\n%x\n%t\n", user.WalletName, user.FundingPublicKey.SerializeCompressed(), user.FundingPrivateKey.Serialize(), user.Fundee)
}

// Equals compares two users
func (user *User) Equals(user2 *User) bool {
	return user.WalletName == user2.WalletName && user.UserBalance == user2.UserBalance
}

// GenerateNewUserFromWallet generates a new channel user from a wallet
func GenerateNewUserFromWallet(name string, walletName string, fundee bool, client *rpcclient.Client) (*User, error) {
	clientWraper := rpcutils.New(client)
	clientWraper.UnloadAllWallets()
	clientWraper.LoadWallet(walletName)

	payoutAddress, _ := btcutil.DecodeAddress(clientWraper.GetNewP2PKHAddress(), config)
	payoutPrivKey, _ := client.DumpPrivKey(payoutAddress)

	//Payout address for unencumbered
	payoutPubKey, _ := clientWraper.GetPubKey(payoutAddress.EncodeAddress())

	fundingPrivateKey, _ := btcec.NewPrivateKey(btcec.S256())
	HTLCPrivateKey, _ := btcec.NewPrivateKey(btcec.S256())

	fundingSigner := &SimpleSigner{
		PrivateKey: fundingPrivateKey,
	}

	user := &User{
		Name:              name,
		FundingPublicKey:  fundingPrivateKey.PubKey(),
		FundingPrivateKey: fundingPrivateKey,
		HTLCPublicKey:     HTLCPrivateKey.PubKey(),
		HTLCPrivateKey:    HTLCPrivateKey,
		PayOutAddress:     payoutAddress,
		PayoutPubKey:      payoutPubKey,
		PayoutPrivateKey:  payoutPrivKey,
		FundingSigner:     fundingSigner,
		UserBalance:       0,
		Fundee:            fundee,
		WalletName:        walletName,
		RevokePreImage:    []byte(walletName),
		HTLCPreImage:      []byte(walletName),
		HTLCPaymentHash:   sha256.Sum256([]byte(walletName)),
	}

	return user, nil
}
