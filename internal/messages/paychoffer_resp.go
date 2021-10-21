package messages

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"encoding/hex"
	"encoding/json"

	"github.com/wcgcyx/fcr/internal/paychoffer"
)

// paychOfferRespJson is used for serialization of the message.
type paychOfferRespJson struct {
	PaychOffer string `json:"paych_offer"`
}

// EncodePaychOfferResp encodes a paych offer resp message.
// It takes a paych offer as the argument.
// It returns the bytes serialized from this message.
func EncodePaychOfferResp(
	offer paychoffer.PaychOffer,
) []byte {
	data, _ := json.Marshal(paychOfferRespJson{
		PaychOffer: hex.EncodeToString(offer.ToBytes()),
	})
	return data
}

// DecodePaychOfferResp decodes bytes into a paych offer resp message.
// It takes a byte array as the argument.
// It returns the a paych offer and error.
func DecodePaychOfferResp(data []byte) (
	*paychoffer.PaychOffer,
	error,
) {
	msg := paychOfferRespJson{}
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	offerData, err := hex.DecodeString(msg.PaychOffer)
	if err != nil {
		return nil, err
	}
	offer, err := paychoffer.FromBytes(offerData)
	if err != nil {
		return nil, err
	}
	return offer, nil
}
