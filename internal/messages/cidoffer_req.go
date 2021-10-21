package messages

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"encoding/json"

	"github.com/ipfs/go-cid"
)

// cidOfferReqJson is used for serialization of the message.
type cidOfferReqJson struct {
	CurrencyID uint64 `json:"currency_id"`
	CID        string `json:"cid"`
}

// EncodeCIDOfferReq encodes a cid offer req message.
// It takes a currency id, a cid as arguments.
// It returns the bytes serialized from this message.
func EncodeCIDOfferReq(
	currencyID uint64,
	id cid.Cid,
) []byte {
	data, _ := json.Marshal(cidOfferReqJson{
		CurrencyID: currencyID,
		CID:        id.String(),
	})
	return data
}

// DecodeCIDOfferReq decodes bytes into a cid offer req message.
// It takes a byte array as the argument.
// It returns the currency id, a cid and error.
func DecodeCIDOfferReq(data []byte) (
	uint64,
	cid.Cid,
	error,
) {
	msg := cidOfferReqJson{}
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return 0, cid.Undef, err
	}
	id, err := cid.Parse(msg.CID)
	if err != nil {
		return 0, cid.Undef, err
	}
	return msg.CurrencyID, id, nil
}
