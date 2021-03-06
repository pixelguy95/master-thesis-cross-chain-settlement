package bitcoin

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"

	"errors"

	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
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

// SignRawTransactionWithWalletReply a representation of the json reply
type SignRawTransactionWithWalletReply = struct {
	Hex      string                              `json:"hex"`
	Complete bool                                `json:"complete"`
	Errors   []SignRawTransactionWithWalletError `json:"errors"`
}

// SignRawTransactionWithWalletError holds any possible errors from above reply
type SignRawTransactionWithWalletError = struct {
	Txid      string `json:"txid"`
	Vout      string `json:"vout"`
	ScriptSig string `json:"scriptSig"`
	Sequence  int    `json:"sequence"`
	Error     string `json:"error"`
}

// PubKeyReply used to fetch the raw pubkey from an address
type PubKeyReply = struct {
	PubKey string `json:"pubkey"`
}

// FundPosition a
type FundPosition = struct {
	ChangePosition int `json:"changePosition"`
}

// FundRawTransactionReply a representation of the fund json reply
type FundRawTransactionReply = struct {
	Hex       string  `json:"hex"`
	Fee       float64 `json:"fee"`
	ChangePos int     `json:"changepos"`
}

// LoadWallet loads a new wallet
func (c *CustomRPC) LoadWallet(walletName string) (*LoadWalletReply, error) {

	log.Printf("Loading wallet: %s\n", walletName)

	params, _ := json.Marshal(walletName)
	paramsRaw := []json.RawMessage{params}

	rawData, error := c.client.RawRequest("loadwallet", paramsRaw)

	if error != nil {
		log.Fatalf("Error occured while loading wallet: %s\n", error)
		return nil, errors.New("Wallet already loaded or doesn't exist most likely")
	}

	reply := new(LoadWalletReply)
	json.Unmarshal(rawData, &reply)

	return reply, nil
}

// CreateWallet creates a new wallet
// Returns a loadwallet struct because the messages look the same
func (c *CustomRPC) CreateWallet(walletName string) (*LoadWalletReply, error) {

	log.Println("Creating a new wallet")

	params, _ := json.Marshal(walletName)
	paramsRaw := []json.RawMessage{params}

	rawData, error := c.client.RawRequest("createwallet", paramsRaw)

	if error != nil {
		fmt.Printf("Error occured while creating wallet: %s\n", error)
		return nil, error
	}

	reply := new(LoadWalletReply)
	json.Unmarshal(rawData, &reply)

	return reply, nil
}

// ListWallets returns a list of all the wallets that has been loaded on the node
func (c *CustomRPC) ListWallets() ([]string, error) {

	rawData, error := c.client.RawRequest("listwallets", nil)

	if error != nil {
		fmt.Printf("Error occured while listing wallets: %s\n", error)
		return nil, errors.New("I dont't know what went wrong")
	}

	reply := new([]string)
	json.Unmarshal(rawData, &reply)

	return *reply, nil
}

// UnloadAllWallets loops through all loaded wallets and unloads them
func (c *CustomRPC) UnloadAllWallets() error {

	log.Println("Unloading all wallets")

	wallets, _ := c.ListWallets()

	for _, walletName := range wallets {

		params, _ := json.Marshal(walletName)
		paramsRaw := []json.RawMessage{params}
		_, error := c.client.RawRequest("unloadwallet", paramsRaw)

		if error != nil {
			log.Fatalf("Error occured while loading wallet: %s\n", error)
			return errors.New("Wallet already loaded or doesn't exist most likely")
		}
	}

	return nil
}

// GetNewP2PKHAddress returns a brand new P2PKH address
func (c *CustomRPC) GetNewP2PKHAddress() string {

	log.Print("Generating new P2PKH address: ")

	param1, _ := json.Marshal("")
	param2, _ := json.Marshal("legacy")
	paramsRaw := []json.RawMessage{param1, param2}

	rawData, _ := c.client.RawRequest("getnewaddress", paramsRaw)

	address := new(string)
	json.Unmarshal(rawData, &address)

	log.Printf("%s\n", *address)

	return *address
}

// GetNewPubKey returns a new pure pubkey
func (c *CustomRPC) GetNewPubKey() (*btcutil.AddressPubKey, error) {

	param1, _ := json.Marshal("")
	param2, _ := json.Marshal("legacy")
	paramsRaw := []json.RawMessage{param1, param2}

	rawData, _ := c.client.RawRequest("getnewaddress", paramsRaw)

	address := new(string)
	json.Unmarshal(rawData, &address)

	fmt.Printf("Generating new pub-key from %s: ", *address)

	param1, _ = json.Marshal(address)
	paramsRaw = []json.RawMessage{param1}

	rawData, _ = c.client.RawRequest("getaddressinfo", paramsRaw)

	pubkey := new(PubKeyReply)
	json.Unmarshal(rawData, &pubkey)
	fmt.Printf("%s\n", pubkey.PubKey)

	asBytes, _ := hex.DecodeString(pubkey.PubKey)
	key, error := btcutil.NewAddressPubKey(asBytes, &chaincfg.TestNet3Params)

	if error != nil {
		fmt.Println(error)
		return nil, error
	}

	return key, nil
}

// GetPubKey takes in a address string and returns the publoc key
func (c *CustomRPC) GetPubKey(address string) (*btcutil.AddressPubKey, error) {

	param1, _ := json.Marshal(address)
	paramsRaw := []json.RawMessage{param1}

	rawData, _ := c.client.RawRequest("getaddressinfo", paramsRaw)

	pubkey := new(PubKeyReply)
	json.Unmarshal(rawData, &pubkey)

	asBytes, _ := hex.DecodeString(pubkey.PubKey)
	key, error := btcutil.NewAddressPubKey(asBytes, &chaincfg.TestNet3Params)

	if error != nil {
		log.Fatal(error)
		return nil, error
	}

	return key, nil
}

// SignRawTransactionWithWallet signs a transaction with whatever wallet is loaded
func (c *CustomRPC) SignRawTransactionWithWallet(tx *wire.MsgTx) (*wire.MsgTx, error) {

	log.Println("Signing raw transaction with wallet")

	// Serialize the transaction
	buf := new(bytes.Buffer)
	tx.Serialize(buf)
	hexEncoding := hex.EncodeToString(buf.Bytes())
	params, _ := json.Marshal(hexEncoding)
	paramsRaw := []json.RawMessage{params}

	rawData, error := c.client.RawRequest("signrawtransactionwithwallet", paramsRaw)

	if error != nil {
		fmt.Println(error)
		return nil, error
	}

	reply := new(SignRawTransactionWithWalletReply)
	json.Unmarshal(rawData, &reply)

	ret := new(wire.MsgTx)
	txbytes, _ := hex.DecodeString(reply.Hex)
	ret.Deserialize(bytes.NewReader(txbytes))

	return ret, nil
}

// FundRawTransaction fund a transaction with whatever wallet is loaded
func (c *CustomRPC) FundRawTransaction(tx *wire.MsgTx) (*wire.MsgTx, error) {

	log.Println("Funding raw transaction")

	// Serialize the transaction
	buf := new(bytes.Buffer)
	tx.Serialize(buf)
	hexEncoding := hex.EncodeToString(buf.Bytes())
	param1, error := json.Marshal(hexEncoding)
	param2, error := json.Marshal(&FundPosition{ChangePosition: 1})

	paramsRaw := []json.RawMessage{param1, param2}

	rawData, error := c.client.RawRequest("fundrawtransaction", paramsRaw)

	if error != nil {
		log.Fatal(error)
		return nil, error
	}

	reply := new(FundRawTransactionReply)
	json.Unmarshal(rawData, &reply)

	ret := new(wire.MsgTx)
	txbytes, _ := hex.DecodeString(reply.Hex)
	ret.Deserialize(bytes.NewReader(txbytes))

	return ret, nil
}
