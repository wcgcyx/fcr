package messages

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"encoding/hex"
	"encoding/json"

	"github.com/wcgcyx/fcr/internal/payoffer"
)

// payOfferRespJson is used for serialization of the message.
type payOfferRespJson struct {
	PayOffer string `json:"payoffer"`
}

// EncodePayOfferResp encodes a pay offer resp message.
// It takes a pay offer the as argument.
// It returns the bytes serialized from this message.
func EncodePayOfferResp(
	offer payoffer.PayOffer,
) []byte {
	data, _ := json.Marshal(payOfferRespJson{
		PayOffer: hex.EncodeToString(offer.ToBytes()),
	})
	return data
}

// DecodePayOfferResp decodes bytes into a pay offer resp message.
// It takes a byte array as the argument.
// It returns the a pay offer and error.
func DecodePayOfferResp(data []byte) (
	*payoffer.PayOffer,
	error,
) {
	msg := payOfferRespJson{}
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	offerData, err := hex.DecodeString(msg.PayOffer)
	if err != nil {
		return nil, err
	}
	offer, err := payoffer.FromBytes(offerData)
	if err != nil {
		return nil, err
	}
	return offer, nil
}
