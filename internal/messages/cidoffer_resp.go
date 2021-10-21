package messages

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"encoding/hex"
	"encoding/json"

	"github.com/wcgcyx/fcr/internal/cidoffer"
)

// cidOfferRespJson is used for serialization of the message.
type cidOfferRespJson struct {
	CIDOffer string `json:"cidoffer"`
}

// EncodeCIDOfferResp encodes a cid offer resp message.
// It takes a cid offer the as argument.
// It returns the bytes serialized from this message.
func EncodeCIDOfferResp(
	offer cidoffer.CIDOffer,
) []byte {
	data, _ := json.Marshal(cidOfferRespJson{
		CIDOffer: hex.EncodeToString(offer.ToBytes()),
	})
	return data
}

// DecodeCIDOfferResp decodes bytes into a cid offer resp message.
// It takes a byte array as the argument.
// It returns the a cid offer and error.
func DecodeCIDOfferResp(data []byte) (
	*cidoffer.CIDOffer,
	error,
) {
	msg := cidOfferRespJson{}
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	offerData, err := hex.DecodeString(msg.CIDOffer)
	if err != nil {
		return nil, err
	}
	offer, err := cidoffer.FromBytes(offerData)
	if err != nil {
		return nil, err
	}
	return offer, nil
}
