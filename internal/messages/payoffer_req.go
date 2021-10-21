package messages

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"encoding/json"
	"fmt"
	"math/big"
)

// payOfferReqJson is used for serialization of the message.
type payOfferReqJson struct {
	CurrencyID uint64   `json:"currency_id"`
	Route      []string `json:"route"`
	Amt        string   `json:"amt"`
}

// EncodePayOfferReq encodes a pay offer req message.
// It takes a currency id, a route, amt as arguments.
// It returns the bytes serialized from this message.
func EncodePayOfferReq(
	currencyID uint64,
	route []string,
	amt *big.Int,
) []byte {
	data, _ := json.Marshal(payOfferReqJson{
		CurrencyID: currencyID,
		Route:      route,
		Amt:        amt.String(),
	})
	return data
}

// DecodePayOfferReq decodes bytes into a pay offer req message.
// It takes a byte array as the argument.
// It returns the currency id, a route, amt and error.
func DecodePayOfferReq(data []byte) (
	uint64,
	[]string,
	*big.Int,
	error,
) {
	msg := payOfferReqJson{}
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return 0, nil, nil, err
	}
	amt, ok := big.NewInt(0).SetString(msg.Amt, 10)
	if !ok {
		return 0, nil, nil, fmt.Errorf("fail to set amt")
	}
	return msg.CurrencyID, msg.Route, amt, nil
}
