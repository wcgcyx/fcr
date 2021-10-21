package messages

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"encoding/hex"
	"encoding/json"

	"github.com/libp2p/go-libp2p-core/peer"
)

// subscribeJson is used for serialization of the message.
type subscribeJson struct {
	CurrencyID uint64 `json:"currency_id"`
	PeerInfo   string `json:"peer_info"`
}

// EncodeSubscribe encodes a subscribe message.
// It takes a currency id, the peer info of the subscriber as arguments.
// It returns the bytes serialized from this message.
func EncodeSubscribe(
	currencyID uint64,
	pi peer.AddrInfo,
) []byte {
	piData, _ := pi.MarshalJSON()
	data, _ := json.Marshal(subscribeJson{
		CurrencyID: currencyID,
		PeerInfo:   hex.EncodeToString(piData),
	})
	return data
}

// DecodeSubscribe decodes bytes into a subscribe message.
// It takes a byte array as the argument.
// It returns the currency id, the peer info of the subscriber and error.
func DecodeSubscribe(data []byte) (
	uint64,
	peer.AddrInfo,
	error,
) {
	msg := subscribeJson{}
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return 0, peer.AddrInfo{}, err
	}
	piData, err := hex.DecodeString(msg.PeerInfo)
	if err != nil {
		return 0, peer.AddrInfo{}, err
	}
	pi := &peer.AddrInfo{}
	err = pi.UnmarshalJSON(piData)
	if err != nil {
		return 0, peer.AddrInfo{}, err
	}
	return msg.CurrencyID, *pi, nil
}
