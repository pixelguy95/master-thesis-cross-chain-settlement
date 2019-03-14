package litecoin

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ltcsuite/ltcd/rpcclient"
	"github.com/ltcsuite/ltcd/wire"
)

// CustomRPC a wrapper around the rpcclient
type CustomRPC struct {
	client *rpcclient.Client
}

// New constructor for the wrapper
func New(client *rpcclient.Client) *CustomRPC {
	return &CustomRPC{
		client: client,
	}
}

// LoadWalletReply a representation of the json reply
type LoadWalletReply = struct {
	Name    string `json:"name"`
	Warning string `json:"warning"`
}

// SignRawTransactionReply a representation of the json reply
type SignRawTransactionReply = struct {
	Hex      string                    `json:"hex"`
	Complete bool                      `json:"complete"`
	Errors   []SignRawTransactionError `json:"errors"`
}

// SignRawTransactionError holds any possible errors from above reply
type SignRawTransactionError = struct {
	Txid      string `json:"txid"`
	Vout      string `json:"vout"`
	ScriptSig string `json:"scriptSig"`
	Sequence  int    `json:"sequence"`
	Error     string `json:"error"`
}

// GetNewP2PKHAddress returns a brand new P2PKH address
func (c *CustomRPC) GetNewP2PKHAddress() string {

	fmt.Print("Generating new P2PKH address: ")

	param1, _ := json.Marshal("")
	param2, _ := json.Marshal("legacy")
	paramsRaw := []json.RawMessage{param1, param2}

	rawData, _ := c.client.RawRequest("getnewaddress", paramsRaw)

	address := new(string)
	json.Unmarshal(rawData, &address)

	fmt.Printf("%s\n", *address)

	return *address
}

// SignRawTransaction signs a transaction
func (c *CustomRPC) SignRawTransaction(tx *wire.MsgTx) (*wire.MsgTx, error) {

	fmt.Println("Signing raw transaction with wallet")

	// Serialize the transaction
	buf := new(bytes.Buffer)
	tx.SerializeNoWitness(buf)
	hexEncoding := hex.EncodeToString(buf.Bytes())
	params, _ := json.Marshal(hexEncoding)
	paramsRaw := []json.RawMessage{params}

	rawData, error := c.client.RawRequest("signrawtransaction", paramsRaw)

	if error != nil {
		fmt.Println(error)
		return nil, error
	}

	reply := new(SignRawTransactionReply)
	json.Unmarshal(rawData, &reply)

	ret := new(wire.MsgTx)
	txbytes, _ := hex.DecodeString(reply.Hex)
	ret.DeserializeNoWitness(bytes.NewReader(txbytes))

	return ret, nil
}
