package channel

import (
	"bytes"
	"errors"
	"fmt"
	"log"

	rpcutils "../../extensions/bitcoin"
	ltcutils "../../extensions/litecoin"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	"github.com/lightningnetwork/lnd/input"
	ltcrpc "github.com/ltcsuite/ltcd/rpcclient"
)

var config = &chaincfg.TestNet3Params

// OpenNewChannel opens a new channel between two parties, first user with fundee set will be considred to be the funder of the channel
// Only supports single funder
func OpenNewChannel(party1 *User, party2 *User, isLtc bool, client *rpcclient.Client, LtcClient *ltcrpc.Client) (channel *Channel, err error) {

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
	ltcWraper := ltcutils.New(LtcClient)

	// Create funding transaction
	fundingTx := wire.NewMsgTx(2)

	witnessScript, multiSigOut, err := input.GenFundingPkScript(party1.FundingPublicKey.SerializeCompressed(), party2.FundingPublicKey.SerializeCompressed(), funder.UserBalance)
	fmt.Println(funder.UserBalance)
	if err != nil {
		return nil, err
	}

	fundingTx.AddTxOut(multiSigOut)

	if !isLtc {
		clientWraper.UnloadAllWallets()
		clientWraper.LoadWallet(funder.WalletName)
		fundingTx, _ = clientWraper.FundRawTransaction(fundingTx)
		fundingTx, _ = clientWraper.SignRawTransactionWithWallet(fundingTx)
	} else {

		buf := new(bytes.Buffer)
		fundingTx.Serialize(buf)
		fmt.Printf("FUNDING TX:\n%x\n\n", buf)

		fundingTx, _ = ltcWraper.FundRawTransaction(fundingTx)

		buf = new(bytes.Buffer)
		fundingTx.Serialize(buf)
		fmt.Printf("Funded FUNDING TX:\n%x\n\n", buf)

		fundingTx, _ = ltcWraper.SignRawTransaction(fundingTx)

		buf = new(bytes.Buffer)
		fundingTx.Serialize(buf)
		fmt.Printf("signed FUNDING TX:\n%x\n\n", buf)
	}

	channel = &Channel{
		Party1:               party1,
		Party2:               party2,
		ChannelBalance:       funder.UserBalance,
		FundingTx:            fundingTx,
		FundingWitnessScript: witnessScript,
		FundingMultiSigOut:   multiSigOut,
	}

	err = channel.Settle(client)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return channel, nil
}
