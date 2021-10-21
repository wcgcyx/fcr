package messages

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"encoding/hex"
	"encoding/json"

	"github.com/wcgcyx/fcr/internal/paychoffer"
)

// paychRegisterJson is used for serialization of the message.
type paychRegisterJson struct {
	PaychAddr  string `json:"paych_addr"`
	PaychOffer string `json:"paych_offer"`
}

// EncodePaychRegister encodes a paych register message.
// It takes a payment channel address, a paych offer as arguments.
// It returns the bytes serialized from this message.
func EncodePaychRegister(
	paychAddr string,
	offer *paychoffer.PaychOffer,
) []byte {
	offerStr := hex.EncodeToString(offer.ToBytes())
	data, _ := json.Marshal(paychRegisterJson{
		PaychAddr:  paychAddr,
		PaychOffer: offerStr,
	})
	return data
}

// DecodePaychRegister decodes bytes into a paych register message.
// It takes a byte array as the argument.
// It returns the payment channel address, a paych offer and error.
func DecodePaychRegister(data []byte) (
	string,
	*paychoffer.PaychOffer,
	error,
) {
	msg := paychRegisterJson{}
	err := json.Unmarshal(data, &msg)
	offerData, err := hex.DecodeString(msg.PaychOffer)
	if err != nil {
		return "", nil, err
	}
	offer, err := paychoffer.FromBytes(offerData)
	if err != nil {
		return "", nil, err
	}
	return msg.PaychAddr, offer, nil
}
