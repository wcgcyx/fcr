package messages

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// payRespJson is used for serialization of the message.
type payRespJson struct {
	Receipt string `json:"receipt"`
}

// EncodePayResp encodes a pay resp message.
// It takes the receipt as the argument.
// It returns the bytes serialised from this message.
func EncodePayResp(
	receipt []byte,
) []byte {
	data, _ := json.Marshal(payRespJson{
		Receipt: hex.EncodeToString(receipt),
	})
	return data
}

// DecodePayResp decodes bytes into a pay resp message.
// It takes a byte array as the argument.
// It returns the receipt and error.
func DecodePayResp(data []byte) (
	[65]byte,
	error,
) {
	msg := payRespJson{}
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return [65]byte{}, err
	}
	receipt, err := hex.DecodeString(msg.Receipt)
	if err != nil {
		return [65]byte{}, err
	}
	if len(receipt) != 65 {
		return [65]byte{}, fmt.Errorf("receipt expect 65 got %v", len(receipt))
	}
	var receiptData [65]byte
	copy(receiptData[:], receipt)
	return receiptData, nil

}
