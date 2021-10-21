package messages

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import "encoding/json"

// paychOfferReqJson is used for serialization of the message.
type paychOfferReqJson struct {
	CurrencyID uint64 `json:"currency_id"`
	Addr       string `json:"addr"`
}

// EncodePaychOfferReq encodes a paych offer req message.
// It takes a currency id, the address of the proposed recipient as arguments.
// It returns the bytes serialized from this message.
func EncodePaychOfferReq(
	currencyID uint64,
	addr string,
) []byte {
	data, _ := json.Marshal(paychOfferReqJson{
		CurrencyID: currencyID,
		Addr:       addr,
	})
	return data
}

// DecodePaychOfferReq decodes bytes into a paych offer req message.
// It takes a byte array as the argument.
// It returns the currency id, the address of the proposed recipient and error.
func DecodePaychOfferReq(data []byte) (
	uint64,
	string,
	error,
) {
	msg := paychOfferReqJson{}
	err := json.Unmarshal(data, &msg)
	return msg.CurrencyID, msg.Addr, err
}
