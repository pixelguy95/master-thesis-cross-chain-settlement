package channel

import (
	rpcutils "../../extensions/bitcoin"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	"github.com/lightningnetwork/lnd/input"
)

// CreateStaticCommits creates both commits with no HTLC output
func (channel *Channel) CreateStaticCommits(client *rpcclient.Client) (*wire.MsgTx, *wire.MsgTx, error) {

	fundingTxIn := wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  channel.FundingTx.TxHash(),
			Index: 0,
		},
	}

	commitParty1, err := createStaticCommit(channel.Party1, channel.Party2, &fundingTxIn, client)
	if err != nil {
		return nil, nil, err
	}

	commitParty2, err := createStaticCommit(channel.Party2, channel.Party1, &fundingTxIn, client)
	if err != nil {
		return nil, nil, err
	}

	return commitParty1, commitParty2, nil
}

// createStaticCommit creates a single commit, no htlc
func createStaticCommit(encumbered *User, unencumbered *User, fundingTxIn *wire.TxIn, client *rpcclient.Client) (*wire.MsgTx, error) {
	clientWraper := rpcutils.New(client)

	//Create revoke key on party1 commit side
	commitSecret, commitPoint := btcec.PrivKeyFromBytes(btcec.S256(), encumbered.RevokePreImage)
	basePriv := unencumbered.FundingPrivateKey.PrivKey
	basePub := (*btcec.PublicKey)(&basePriv.PublicKey)

	revocationPub := input.DeriveRevocationPubkey(basePub, commitPoint)

	encumbered.RevokationSecrets = append(encumbered.RevokationSecrets, &CommitRevokationSecret{CommitPoint: commitPoint, CommitSecret: commitSecret})

	//New address for encumbered
	clientWraper.UnloadAllWallets()
	clientWraper.LoadWallet(encumbered.WalletName)
	party1PubKey, _ := clientWraper.GetPubKey(encumbered.FundingPublicKey.EncodeAddress())

	//New address for unencumbered
	clientWraper.UnloadAllWallets()
	clientWraper.LoadWallet(unencumbered.WalletName)
	party2PubKey, _ := clientWraper.GetPubKey(unencumbered.FundingPublicKey.EncodeAddress())

	//Delayed script to self, or via revocation
	ourRedeemScript, err := input.CommitScriptToSelf(100, party1PubKey.PubKey(),
		revocationPub)
	if err != nil {
		return nil, err
	}

	payToUsScriptHash, err := input.WitnessScriptHash(ourRedeemScript)
	if err != nil {
		return nil, err
	}

	//Unencumbered payout to other party
	theirWitnessKeyHash, err := input.CommitScriptUnencumbered(party2PubKey.PubKey())
	if err != nil {
		return nil, err
	}

	commitTx := wire.NewMsgTx(2)
	commitTx.AddTxIn(fundingTxIn)

	commitTx.AddTxOut(&wire.TxOut{
		PkScript: payToUsScriptHash,
		Value:    int64(encumbered.UserBalance),
	})

	commitTx.AddTxOut(&wire.TxOut{
		PkScript: theirWitnessKeyHash,
		Value:    int64(unencumbered.UserBalance),
	})

	return commitTx, nil
}
