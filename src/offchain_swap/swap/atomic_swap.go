package swap

import (
	"crypto/rand"
	"crypto/sha256"
	"log"

	"github.com/pixelguy95/master-thesis-cross-chain-settlement/src/offchain_swap/channel"
)

//AtomicSwapDescriptor contains the data needed to do a swap offchain
type AtomicSwapDescriptor struct {
	SenderBitcoin   *channel.User
	ReceiverBitcoin *channel.User

	SenderLitecoin   *channel.User
	ReceiverLitecoin *channel.User

	Amount int64

	HTLCPreImage [32]byte
	PaymentHash  [32]byte

	Rate float32
}

// ExtractSendDescriptorBitcoin extracts the senddescriptor from the swap details
func (swap *AtomicSwapDescriptor) ExtractSendDescriptorBitcoin() *channel.SendDescriptor {

	return &channel.SendDescriptor{
		Sender:       swap.SenderBitcoin,
		Receiver:     swap.ReceiverBitcoin,
		Balance:      swap.Amount,
		HTLCPreImage: swap.HTLCPreImage,
		PaymentHash:  swap.PaymentHash,
	}
}

// ExtractSendDescriptorLitecoin extracts the senddescriptor from the swap details
func (swap *AtomicSwapDescriptor) ExtractSendDescriptorLitecoin() *channel.SendDescriptor {

	return &channel.SendDescriptor{
		Sender:       swap.SenderLitecoin,
		Receiver:     swap.ReceiverLitecoin,
		Balance:      int64(float32(swap.Amount) * swap.Rate),
		HTLCPreImage: swap.HTLCPreImage,
		PaymentHash:  swap.PaymentHash,
	}
}

// GenerateAtomicSwap generates an atomic swap between two payment channels
func GenerateAtomicSwap(bitcoinChannel *channel.Channel, litecoinChannel *channel.Channel, amount int64) error {

	log.Println("Generating an offchain atomic swap")
	log.Println("Creating regular T channel transaction on bitcoin side")

	// Preimage and hash used in the atomic swap
	var htlcPreImage [32]byte
	rand.Read(htlcPreImage[:])

	swap := &AtomicSwapDescriptor{
		SenderBitcoin:    bitcoinChannel.Party1,
		ReceiverBitcoin:  bitcoinChannel.Party2,
		SenderLitecoin:   litecoinChannel.Party1,
		ReceiverLitecoin: litecoinChannel.Party2,
		Amount:           amount,
		Rate:             1.1,
		HTLCPreImage:     htlcPreImage,
		PaymentHash:      sha256.Sum256(htlcPreImage[:]),
	}

	bitcoinPayment := swap.ExtractSendDescriptorBitcoin()
	bitcoinChannel.Pay(bitcoinPayment)

	return nil
}
