package messages

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/wcgcyx/fcr/internal/payoffer"
)

// payReqJson is used for serialization of the message.
type payReqJson struct {
	CurrencyID   uint64 `json:"currency_id"`
	OriginAddr   string `json:"origin_addr"`
	OriginSecret string `json:"origin_secret"`
	OriginAmt    string `json:"origin_amt"`
	FromAddr     string `json:"from_addr"`
	Voucher      string `json:"voucher"`
	PayOffer     string `json:"pay_offer"`
}

// EncodePayReq encodes a pay req message.
// It takes the currency id, origin address, origin secret, origin amount, from address, voucher, a pay offer (can be nil) as arguments.
// It returns the bytes serialized from this message.
func EncodePayReq(
	currencyID uint64,
	originAddr string,
	originSecret [32]byte,
	originAmt *big.Int,
	fromAddr string,
	voucher string,
	offer *payoffer.PayOffer,
) []byte {
	offerStr := ""
	if offer != nil {
		offerStr = hex.EncodeToString(offer.ToBytes())
	}
	data, _ := json.Marshal(payReqJson{
		CurrencyID:   currencyID,
		OriginAddr:   originAddr,
		OriginSecret: hex.EncodeToString(originSecret[:]),
		OriginAmt:    originAmt.String(),
		FromAddr:     fromAddr,
		Voucher:      voucher,
		PayOffer:     offerStr,
	})
	return data
}

// DecodePayReq decodees bytes into a pay req message.
// It takes a byte array as the argument.
// It returns the currency id, origin address, origin secret, origin amount, from address, voucher, a pay offer (will be nil if not exist) and error.
func DecodePayReq(data []byte) (
	uint64,
	string,
	[32]byte,
	*big.Int,
	string,
	string,
	*payoffer.PayOffer,
	error,
) {
	msg := payReqJson{}
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return 0, "", [32]byte{}, nil, "", "", nil, err
	}
	secret, err := hex.DecodeString(msg.OriginSecret)
	if err != nil {
		return 0, "", [32]byte{}, nil, "", "", nil, err
	}
	if len(secret) != 32 {
		return 0, "", [32]byte{}, nil, "", "", nil, fmt.Errorf("payer secret expect 32 got %v", len(secret))
	}
	originAmt, ok := big.NewInt(0).SetString(msg.OriginAmt, 10)
	if !ok {
		return 0, "", [32]byte{}, nil, "", "", nil, fmt.Errorf("fail to set origin amt")
	}
	var secretData [32]byte
	copy(secretData[:], secret)
	var offer *payoffer.PayOffer
	if msg.PayOffer != "" {
		offerData, err := hex.DecodeString(msg.PayOffer)
		if err != nil {
			return 0, "", [32]byte{}, nil, "", "", nil, err
		}
		offer, err = payoffer.FromBytes(offerData)
		if err != nil {
			return 0, "", [32]byte{}, nil, "", "", nil, err
		}
	}
	return msg.CurrencyID, msg.OriginAddr, secretData, originAmt, msg.FromAddr, msg.Voucher, offer, nil
}
