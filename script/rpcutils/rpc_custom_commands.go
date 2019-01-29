package rpcutils

import (
	"encoding/json"
	"fmt"

	"errors"

	"github.com/btcsuite/btcd/rpcclient"
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

// LoadWallet loads a new wallet
func (c *CustomRPC) LoadWallet(walletName string) (*LoadWalletReply, error) {
	params, _ := json.Marshal(walletName)
	paramsRaw := []json.RawMessage{params}

	rawData, error := c.client.RawRequest("loadwallet", paramsRaw)

	if error != nil {
		fmt.Printf("Error occured while loading wallet: %s\n", error)
		return nil, errors.New("Wallet already loaded or doesn't exist most likely")
	}

	reply := new(LoadWalletReply)
	json.Unmarshal(rawData, &reply)

	return reply, nil
}

// CreateWallet creates a new wallet
// Returns a loadwallet struct because the messages look the same
func (c *CustomRPC) CreateWallet(walletName string) (*LoadWalletReply, error) {
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

	wallets, _ := c.ListWallets()

	for _, walletName := range wallets {

		params, _ := json.Marshal(walletName)
		paramsRaw := []json.RawMessage{params}
		_, error := c.client.RawRequest("unloadwallet", paramsRaw)

		if error != nil {
			fmt.Printf("Error occured while loading wallet: %s\n", error)
			return errors.New("Wallet already loaded or doesn't exist most likely")
		}
	}

	return nil
}
