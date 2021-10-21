package paymgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/wcgcyx/fcr/internal/payoffer"
)

// outChState represents a state of the out channel.
type outChState struct {
	// channel state
	toAddr   string
	chAddr   string
	balance  *big.Int
	redeemed *big.Int
	// reserved
	reserved     *big.Int
	reservations map[string]*reservation
	// lane 0 state
	// only lane 0 is supported to minimize gas usage
	nonce   uint64
	voucher string
	// settling at time
	settlingAt int64
	// minimum settlement time
	settlement int64
}

// outChStateJson is used for serialization.
type outChStateJson struct {
	ToAddr       string            `json:"toaddr"`
	ChAddr       string            `json:"chaddr"`
	Balance      string            `json:"balance"`
	Redeemed     string            `json:"redeemed"`
	Reserved     string            `json:"reserved"`
	Reservations map[string][]byte `json:"reservations"`
	Nonce        uint64            `json:"nonce"`
	Voucher      string            `json:"voucher"`
	SettlingAt   int64             `json:"settling_at"`
	Settlement   int64             `json:"settlement"`
}

// reservation represents the state of a researvation in a out state.
type reservation struct {
	// remaining amount in this reservation
	remain *big.Int
	// amount actually paid
	amtPaid *big.Int
	// amount paid for surcharge
	surchargePaid *big.Int
	// last access time, unix second
	last int64
	// offer linked to this reservation
	offer *payoffer.PayOffer
}

// reservationJson is used for serialization.
type reservationJson struct {
	Remain        string `json:"remain"`
	AmtPaid       string `json:"amt_paid"`
	SurchargePaid string `json:"surcharge_paid"`
	Last          int64  `json:"last"`
	Offer         []byte `json:"offer"`
}

// CleanReservations is used to clean a reservation.
// It returns boolean indicating whether updated.
func (cs *outChState) CleanReservations() bool {
	toRemove := make([]string, 0)
	for resID, res := range cs.reservations {
		if res.offer == nil {
			if time.Now().Unix()-res.last > int64(maxInactivityForSelf.Seconds()) {
				toRemove = append(toRemove, resID)
			}
		} else {
			if time.Now().Unix()-res.last > int64(res.offer.MaxInactivity().Seconds()) {
				toRemove = append(toRemove, resID)
			}
		}
	}
	for _, resID := range toRemove {
		res := cs.reservations[resID]
		cs.reserved.Sub(cs.reserved, res.remain)
		delete(cs.reservations, resID)
	}
	return len(toRemove) > 0
}

// encodeOutChState is used encode out channel state to bytes.
func encodeOutChState(cs *outChState) []byte {
	reservations := make(map[string][]byte)
	for resID, reservation := range cs.reservations {
		var offerData []byte
		if reservation.offer == nil {
			offerData = make([]byte, 0)
		} else {
			offerData = reservation.offer.ToBytes()
		}
		data, _ := json.Marshal(reservationJson{
			Remain:        reservation.remain.String(),
			AmtPaid:       reservation.amtPaid.String(),
			SurchargePaid: reservation.surchargePaid.String(),
			Last:          reservation.last,
			Offer:         offerData,
		})
		reservations[resID] = data
	}
	data, _ := json.Marshal(outChStateJson{
		ToAddr:       cs.toAddr,
		ChAddr:       cs.chAddr,
		Balance:      cs.balance.String(),
		Redeemed:     cs.redeemed.String(),
		Reserved:     cs.reserved.String(),
		Reservations: reservations,
		Nonce:        cs.nonce,
		Voucher:      cs.voucher,
		SettlingAt:   cs.settlingAt,
		Settlement:   cs.settlement,
	})
	return data
}

// decodeOutChState is used to decode bytes into out channel state.
func decodeOutChState(data []byte) *outChState {
	cs := &outChState{}
	csJson := outChStateJson{}
	json.Unmarshal(data, &csJson)
	cs.toAddr = csJson.ToAddr
	cs.chAddr = csJson.ChAddr
	cs.balance, _ = big.NewInt(0).SetString(csJson.Balance, 10)
	cs.redeemed, _ = big.NewInt(0).SetString(csJson.Redeemed, 10)
	cs.reserved, _ = big.NewInt(0).SetString(csJson.Reserved, 10)
	cs.reservations = make(map[string]*reservation)
	cs.nonce = csJson.Nonce
	cs.voucher = csJson.Voucher
	cs.settlingAt = csJson.SettlingAt
	cs.settlement = csJson.Settlement
	for resID, resData := range csJson.Reservations {
		res := &reservation{}
		resJson := reservationJson{}
		json.Unmarshal(resData, &resJson)
		res.remain, _ = big.NewInt(0).SetString(resJson.Remain, 10)
		res.amtPaid, _ = big.NewInt(0).SetString(resJson.AmtPaid, 10)
		res.surchargePaid, _ = big.NewInt(0).SetString(resJson.SurchargePaid, 10)
		res.last = resJson.Last
		if len(resJson.Offer) != 0 {
			res.offer, _ = payoffer.FromBytes(resJson.Offer)
		}
		cs.reservations[resID] = res
	}
	return cs
}

// inChState represents a state of the in channel.
type inChState struct {
	// channel state
	fromAddr string
	chAddr   string
	balance  *big.Int
	redeemed *big.Int
	// offer states
	offerStates map[string]*offerState
	// lane 0 state
	// only lane 0 is supported to minimize gas usage
	nonce   uint64
	voucher string
	// settling at time
	settlingAt int64
	// minimum settlement time
	settlement int64
}

// inChStateJson is used for serialization.
type inChStateJson struct {
	FromAddr    string            `json:"fromaddr"`
	ChAddr      string            `json:"chaddr"`
	Balance     string            `json:"balance"`
	Redeemed    string            `json:"redeemed"`
	OfferStates map[string][]byte `json:"offer_states"`
	Nonce       uint64            `json:"nonce"`
	Voucher     string            `json:"voucher"`
	SettlingAt  int64             `json:"settling_at"`
	Settlement  int64             `json:"settlement"`
}

// offerState represents a state of an offer in active in channel.
type offerState struct {
	// amount actually received
	amtReceived *big.Int
	// surcharge actually received
	surchargeReceived *big.Int
	// last access time, unix second
	last int64
	// offer
	offer *payoffer.PayOffer
}

// CleanReservations is used to clean offer states.
// It returns boolean indicating whether updated.
func (cs *inChState) CleanReservations() bool {
	toRemove := make([]string, 0)
	for offerID, os := range cs.offerStates {
		if time.Now().Unix()-os.last > int64(os.offer.MaxInactivity().Seconds()) {
			toRemove = append(toRemove, offerID)
		}
	}
	for _, offerID := range toRemove {
		delete(cs.offerStates, offerID)
	}
	return len(toRemove) > 0
}

// offerStateJson is used for serialization.
type offerStateJson struct {
	AmtReceived       string `json:"amt_received"`
	SurchargeReceived string `json:"surcharge_received"`
	Last              int64  `json:"last"`
	Offer             []byte `json:"offer"`
}

// encodeInChState is used encode in channel state to bytes.
func encodeInChState(cs *inChState) []byte {
	offerStates := make(map[string][]byte)
	for id, os := range cs.offerStates {
		data, _ := json.Marshal(offerStateJson{
			AmtReceived:       os.amtReceived.String(),
			SurchargeReceived: os.surchargeReceived.String(),
			Last:              os.last,
			Offer:             os.offer.ToBytes(),
		})
		offerStates[id] = data
	}
	data, _ := json.Marshal(inChStateJson{
		FromAddr:    cs.fromAddr,
		ChAddr:      cs.chAddr,
		Balance:     cs.balance.String(),
		Redeemed:    cs.redeemed.String(),
		OfferStates: offerStates,
		Nonce:       cs.nonce,
		Voucher:     cs.voucher,
		SettlingAt:  cs.settlingAt,
		Settlement:  cs.settlement,
	})
	return data
}

// decodeInChState is used to decode bytes into in channel state.
func decodeInChState(data []byte) *inChState {
	cs := &inChState{}
	csJson := inChStateJson{}
	json.Unmarshal(data, &csJson)
	cs.fromAddr = csJson.FromAddr
	cs.chAddr = csJson.ChAddr
	cs.balance, _ = big.NewInt(0).SetString(csJson.Balance, 10)
	cs.redeemed, _ = big.NewInt(0).SetString(csJson.Redeemed, 10)
	cs.offerStates = map[string]*offerState{}
	cs.nonce = csJson.Nonce
	cs.voucher = csJson.Voucher
	cs.settlingAt = csJson.SettlingAt
	cs.settlement = csJson.Settlement
	for id, osData := range csJson.OfferStates {
		os := &offerState{}
		osJson := offerStateJson{}
		json.Unmarshal(osData, &osJson)
		os.amtReceived, _ = big.NewInt(0).SetString(osJson.AmtReceived, 10)
		os.surchargeReceived, _ = big.NewInt(0).SetString(osJson.SurchargeReceived, 10)
		os.last = osJson.Last
		os.offer, _ = payoffer.FromBytes(osJson.Offer)
		cs.offerStates[id] = os
	}
	return cs
}
