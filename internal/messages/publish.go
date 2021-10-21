package messages

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import "encoding/json"

// publishJson is used for serialization of the message.
type publishJson struct {
	CurrencyID uint64     `json:"currency_id"`
	Routes     [][]string `json:"routes"`
}

// EncodePublish encodes a publish message.
// It takes a currency id, the routes in this publish as arguments.
// It returns the bytes serialized from this message.
func EncodePublish(
	currencyID uint64,
	routes [][]string,
) []byte {
	data, _ := json.Marshal(publishJson{
		CurrencyID: currencyID,
		Routes:     routes,
	})
	return data
}

// DecodePublish decodes bytes into a publish message.
// It takes a byte array as the argument.
// It returns the currency id, the routes in this publish and error.
func DecodePublish(data []byte) (
	uint64,
	[][]string,
	error,
) {
	msg := publishJson{}
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return 0, nil, err
	}
	return msg.CurrencyID, msg.Routes, nil
}
