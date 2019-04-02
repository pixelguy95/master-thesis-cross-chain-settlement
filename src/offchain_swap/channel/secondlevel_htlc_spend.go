package channel

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/lightningnetwork/lnd/input"
	"github.com/pixelguy95/master-thesis-cross-chain-settlement/src/onchain_swaps_contract/bitcoin/customtransactions"
)

func GenerateSecondlevelHTLCSpendTx(tx *wire.MsgTx, script []byte, payoutScriptAddress []byte, signKey *btcec.PrivateKey, path byte, csv uint32) (*wire.MsgTx, error) {

	spend := wire.NewMsgTx(2)

	spend.AddTxIn(&wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Hash:  tx.TxHash(),
			Index: 0,
		},
		Sequence: csv,
	})

	outputScript := customtransactions.CreateP2PKHScript(payoutScriptAddress)

	spend.AddTxOut(&wire.TxOut{
		PkScript: outputScript,
		Value:    tx.TxOut[0].Value - int64(customtransactions.DefaultFee),
	})

	signDesc := input.SignDescriptor{
		WitnessScript: script,
		Output:        tx.TxOut[0],
		HashType:      txscript.SigHashAll,
		SigHashes:     txscript.NewTxSigHashes(spend),
		InputIndex:    0,
	}

	s := &SimpleSigner{
		PrivateKey: signKey,
	}

	signature, err := s.SignOutputRaw(spend, &signDesc)
	if err != nil {
		return nil, err
	}

	witnessStack := wire.TxWitness(make([][]byte, 3))
	witnessStack[0] = append(signature, byte(signDesc.HashType))

	if path == 0 {
		witnessStack[1] = nil
	} else {
		witnessStack[1] = []byte{path}
	}

	witnessStack[2] = script

	spend.TxIn[0].Witness = witnessStack

	return spend, nil
}
