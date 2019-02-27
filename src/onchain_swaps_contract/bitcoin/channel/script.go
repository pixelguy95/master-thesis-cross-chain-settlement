package channel

import (
	"crypto/sha256"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
)

// GetFundingOutputScript returns a P2WSH script that represents funding a new channel.
func GetFundingOutputScript(pub1 btcutil.Address, pub2 btcutil.Address) (script []byte, err error) {

	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_2)
	builder.AddData(pub1.ScriptAddress())
	builder.AddData(pub2.ScriptAddress())
	builder.AddOp(txscript.OP_2)
	builder.AddOp(txscript.OP_CHECKMULTISIG)

	witnessScript, _ := builder.Script()

	builder.Reset()
	builder.AddOp(txscript.OP_0)
	scriptHash := sha256.Sum256(witnessScript)
	builder.AddData(scriptHash[:])

	return nil, nil
}
