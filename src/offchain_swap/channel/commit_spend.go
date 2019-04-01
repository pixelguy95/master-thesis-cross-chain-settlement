package channel

import (
	"log"

	"github.com/btcsuite/btcd/btcec"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/lightningnetwork/lnd/input"
	"github.com/pixelguy95/master-thesis-cross-chain-settlement/src/onchain_swaps_contract/bitcoin/customtransactions"
)

// GenerateCommitSpends generates a spend fot eh timelocked output in the base commit
func (c *Channel) GenerateCommitSpends(index uint) error {

	err := createSpend(index, c.Party1)
	if err != nil {
		log.Fatal(err)
	}

	err = createSpend(index, c.Party2)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func createSpend(index uint, peer *User) error {

	log.Printf("Building commit spend transaction for %s", peer.Name)
	spend := wire.NewMsgTx(2)

	if peer.Commits[index].CommitTx.TxOut[0].Value-int64(customtransactions.DefaultFee) < 0 {
		log.Println("No commit spend is needed, as output is 0 or less than dust limit")

		peer.CommitSpends = append(peer.CommitSpends, nil)
		return nil
	}

	commitTxIn := &wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  peer.Commits[index].CommitTx.TxHash(),
			Index: 0,
		},
		Sequence: DefaultRelativeLockTime,
	}

	spend.AddTxIn(commitTxIn)

	var unencumberedPayoutScriptAddress []byte
	if peer.IsLitecoinUser {
		unencumberedPayoutScriptAddress = btcutil.Hash160(peer.LtcPayoutPubKey.SerializeCompressed())
	} else {
		unencumberedPayoutScriptAddress = peer.PayOutAddress.ScriptAddress()
	}

	outputscript := customtransactions.CreateP2PKHScript(unencumberedPayoutScriptAddress)

	output := &wire.TxOut{
		PkScript: outputscript,
		Value:    peer.Commits[index].CommitTx.TxOut[0].Value - int64(customtransactions.DefaultFee),
	}

	spend.AddTxOut(output)

	var s *SimpleSigner
	if peer.IsLitecoinUser {
		btcTypePriv, _ := btcec.PrivKeyFromBytes(btcec.S256(), peer.LtcPayoutPrivateKey.PrivKey.Serialize())
		s = &SimpleSigner{
			PrivateKey: btcTypePriv,
		}
	} else {
		s = &SimpleSigner{
			PrivateKey: peer.PayoutPrivateKey.PrivKey,
		}
	}

	signDesc := input.SignDescriptor{
		WitnessScript: peer.Commits[index].TimeLockedRevocationScript,
		Output:        peer.Commits[index].CommitTx.TxOut[0],
		HashType:      txscript.SigHashAll,
		SigHashes:     txscript.NewTxSigHashes(spend),
		InputIndex:    0,
	}

	signature, err := s.SignOutputRaw(spend, &signDesc)
	if err != nil {
		return err
	}

	signature = append(signature, byte(txscript.SigHashAll))

	witnessStack := make([][]byte, 3)
	witnessStack[0] = signature                                      // Signature
	witnessStack[1] = nil                                            // Choose the timeout path
	witnessStack[2] = peer.Commits[index].TimeLockedRevocationScript // Script that is being payed to

	spend.TxIn[0].Witness = witnessStack

	peer.CommitSpends = append(peer.CommitSpends, &CommitSpendData{CommitSpend: spend})

	return nil
}
