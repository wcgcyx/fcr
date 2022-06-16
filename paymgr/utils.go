package paymgr

/*
 * Dual-licensed under Apache-2.0 and MIT.
 *
 * You can get a copy of the Apache License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * You can also get a copy of the MIT License at
 *
 * http://opensource.org/licenses/MIT
 *
 * @wcgcyx - https://github.com/wcgcyx
 */

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/wcgcyx/fcr/fcroffer"
)

// calculatePettyAmountToReceive is used to calculate the petty amount about to receive from gross amount received.
//
// @input - price-per-period, period, old petty amount received, old surcharge received, gross amount received.
//
// @output - new petty amount received, new surcharge received, petty amount received this time.
func calculatePettyAmountToReceive(ppp *big.Int, period *big.Int, pettyAmtReceived *big.Int, surchargeReceived *big.Int, grossAmtReceived *big.Int) (*big.Int, *big.Int, *big.Int) {
	// Make copies
	ppp = big.NewInt(0).Set(ppp)
	period = big.NewInt(0).Set(period)
	pettyAmtReceived = big.NewInt(0).Set(pettyAmtReceived)
	surchargeReceived = big.NewInt(0).Set(surchargeReceived)
	grossAmtReceived = big.NewInt(0).Set(grossAmtReceived)

	// Start calculating.
	if ppp.Cmp(big.NewInt(0)) > 0 {
		// There is an incoming surcharge.
		pettyAmtToReceive := big.NewInt(0)
		// First fill up surcharge of current period if any.
		var remainder *big.Int
		if surchargeReceived.Cmp(big.NewInt(0)) == 0 {
			remainder = ppp
		} else {
			temp := big.NewInt(0).Sub(surchargeReceived, big.NewInt(1))
			temp = big.NewInt(0).Add(big.NewInt(0).Div(temp, ppp), big.NewInt(1))
			temp = big.NewInt(0).Mul(temp, ppp)
			remainder = big.NewInt(0).Sub(temp, surchargeReceived)
		}
		if remainder.Cmp(grossAmtReceived) >= 0 {
			surchargeReceived.Add(surchargeReceived, grossAmtReceived)
			grossAmtReceived = big.NewInt(0)
		} else {
			surchargeReceived.Add(surchargeReceived, remainder)
			grossAmtReceived.Sub(grossAmtReceived, remainder)
		}
		if grossAmtReceived.Cmp(big.NewInt(0)) > 0 {
			// Second fill up payment of this period.
			// received - (surcharge / ppp) * period
			temp := big.NewInt(0).Div(surchargeReceived, ppp)
			temp = big.NewInt(0).Mul(temp, period)
			remainder = big.NewInt(0).Sub(temp, pettyAmtReceived)
			if remainder.Cmp(grossAmtReceived) >= 0 {
				pettyAmtReceived.Add(pettyAmtReceived, grossAmtReceived)
				pettyAmtToReceive.Add(pettyAmtToReceive, grossAmtReceived)
				grossAmtReceived = big.NewInt(0)
			} else {
				pettyAmtReceived.Add(pettyAmtReceived, remainder)
				pettyAmtToReceive.Add(pettyAmtToReceive, remainder)
				grossAmtReceived.Sub(grossAmtReceived, remainder)
			}
			if grossAmtReceived.Cmp(big.NewInt(0)) > 0 {
				// Third, fill up big period (period + ppp)
				temp := big.NewInt(0).Add(period, ppp)
				bigPeriod := big.NewInt(0).Div(grossAmtReceived, temp)
				temp = big.NewInt(0).Mul(bigPeriod, ppp)
				surchargeReceived.Add(surchargeReceived, temp)
				temp = big.NewInt(0).Mul(bigPeriod, period)
				pettyAmtReceived.Add(pettyAmtReceived, temp)
				temp = big.NewInt(0).Mul(bigPeriod, period)
				pettyAmtToReceive.Add(pettyAmtToReceive, temp)
				temp = big.NewInt(0).Add(period, ppp)
				temp = big.NewInt(0).Mul(bigPeriod, temp)
				grossAmtReceived.Sub(grossAmtReceived, temp)
				if grossAmtReceived.Cmp(big.NewInt(0)) > 0 {
					// Lastly, fill up the last period.
					if grossAmtReceived.Cmp(ppp) >= 0 {
						surchargeReceived.Add(surchargeReceived, ppp)
						temp := big.NewInt(0).Sub(grossAmtReceived, ppp)
						pettyAmtReceived.Add(pettyAmtReceived, temp)
						temp = big.NewInt(0).Sub(grossAmtReceived, ppp)
						pettyAmtToReceive.Add(pettyAmtToReceive, temp)
					} else {
						surchargeReceived.Add(surchargeReceived, grossAmtReceived)
					}
				}
			}
		}
		return pettyAmtReceived, surchargeReceived, pettyAmtToReceive
	} else {
		// There is no incoming surcharge. petty amount receiving will just be the gross amount received.
		pettyAmtReceived.Add(pettyAmtReceived, grossAmtReceived)
		return pettyAmtReceived, surchargeReceived, grossAmtReceived
	}
}

// calculateGrossAmountToPay is used to calculate the gross amount required to pay from petty amount required to pay.
//
// @input - price-per-period, period, old petty amount paid, old surcharge paid, petty amount required to pay.
//
// @output - new petty amount paid, new surcharge paid, gross amount required to pay this time.
func calculateGrossAmountToPay(ppp *big.Int, period *big.Int, pettyAmtPaid *big.Int, surchargePaid *big.Int, pettyAmtToPay *big.Int) (*big.Int, *big.Int, *big.Int) {
	// Make copies
	ppp = big.NewInt(0).Set(ppp)
	period = big.NewInt(0).Set(period)
	pettyAmtPaid = big.NewInt(0).Set(pettyAmtPaid)
	surchargePaid = big.NewInt(0).Set(surchargePaid)
	pettyAmtToPay = big.NewInt(0).Set(pettyAmtToPay)

	// Start calculating
	if ppp.Cmp(big.NewInt(0)) > 0 {
		// There is an out going surcharge.
		expectedAmt := big.NewInt(0).Add(pettyAmtPaid, pettyAmtToPay)
		temp := big.NewInt(0).Sub(expectedAmt, big.NewInt(1))
		temp = big.NewInt(0).Div(temp, period)
		temp = big.NewInt(0).Add(temp, big.NewInt(1))
		expectedSurcharge := big.NewInt(0).Mul(temp, ppp)
		// Amt = (new amt + new surcharge) - (old amt + old surcharge)
		grossAmt := big.NewInt(0).Sub(big.NewInt(0).Add(expectedAmt, expectedSurcharge), big.NewInt(0).Add(pettyAmtPaid, surchargePaid))
		return expectedAmt, expectedSurcharge, grossAmt
	} else {
		// There is no out going surcharge. gross amount to pay will just be the petty amount to pay.
		pettyAmtPaid.Add(pettyAmtPaid, pettyAmtToPay)
		return pettyAmtPaid, surchargePaid, pettyAmtToPay
	}
}

// getExpiredAmount is used to get expired total reserved amount of an outbound channel.
// Can be called only when RLock has been obtained for given channel.
//
// @input - cached state.
//
// @output - total expired reserved amount.
func getExpiredAmount(cs *cachedPaychState) *big.Int {
	expired := big.NewInt(0)
	for _, reservation := range cs.reservations {
		if (!reservation.expiration.IsZero() && time.Now().After(reservation.expiration)) || (reservation.expiration.IsZero() && time.Now().After(reservation.lastAccessed.Add(reservation.inactivity))) {
			// Expired.
			expired.Add(expired, reservation.remain)
		}
	}
	return expired
}

// releaseExpiredAmount is used to clean reservations of outbound channel.
// Should be called every time a Lock has been obtained for given channel.
//
// @input - cached state.
func releaseExpiredAmount(cs *cachedPaychState) {
	expired := big.NewInt(0)
	removedIDs := make([]uint64, 0)
	for resID, reservation := range cs.reservations {
		if (!reservation.expiration.IsZero() && time.Now().After(reservation.expiration)) || (reservation.expiration.IsZero() && time.Now().After(reservation.lastAccessed.Add(reservation.inactivity))) {
			// Expired.
			removedIDs = append(removedIDs, resID)
			expired.Add(expired, reservation.remain)
		}
	}
	// Now perform cleanup.
	for _, removedID := range removedIDs {
		delete(cs.reservations, removedID)
	}
	cs.reservedAmt.Sub(cs.reservedAmt, expired)
}

// encDSVal encodes all active outbound channels excluding channel state into a bytes array.
//
// @input - active outbound channel state.
//
// @output - ds value, error.
func encDSVal(activeOut *cachedPaychState) ([]byte, error) {
	type subValJson struct {
		InboundPPP        *big.Int      `json:"inbound_ppp"`
		InboundPeriod     *big.Int      `json:"inbound_period"`
		PettyAmtReceived  *big.Int      `json:"petty_amt_received"`
		SurchargeReceived *big.Int      `json:"surcharge_received"`
		OutboundPPP       *big.Int      `json:"outbound_ppp"`
		OutboundPeriod    *big.Int      `json:"outbound_period"`
		PettyAmtPaid      *big.Int      `json:"petty_amt_paid"`
		SurchargePaid     *big.Int      `json:"surcharge_paid"`
		Remain            *big.Int      `json:"remain"`
		LastAccessed      time.Time     `json:"last_accessed"`
		Expiration        time.Time     `json:"expiration"`
		Inactivity        time.Duration `json:"inactivity"`
		OfferData         []byte        `json:"offer_data"`
	}
	type valJson struct {
		ReservedAmt   *big.Int              `json:"reserved_amt"`
		ReservedNonce uint64                `json:"reserved_nonce"`
		Reservations  map[uint64]subValJson `json:"reservations"`
	}
	val := valJson{
		ReservedAmt:   activeOut.reservedAmt,
		ReservedNonce: activeOut.reservedNonce,
		Reservations:  make(map[uint64]subValJson),
	}
	for resID, res := range activeOut.reservations {
		var offerData []byte
		if res.offer != nil {
			var err error
			offerData, err = res.offer.Encode()
			if err != nil {
				return nil, err
			}
		}
		val.Reservations[resID] = subValJson{
			InboundPPP:        res.inboundPPP,
			InboundPeriod:     res.inboundPeriod,
			PettyAmtReceived:  res.pettyAmtReceived,
			SurchargeReceived: res.surchargeReceived,
			OutboundPPP:       res.outboundPPP,
			OutboundPeriod:    res.outboundPeriod,
			PettyAmtPaid:      res.pettyAmtPaid,
			SurchargePaid:     res.surchargePaid,
			Remain:            res.remain,
			LastAccessed:      res.lastAccessed,
			Expiration:        res.expiration,
			Inactivity:        res.inactivity,
			OfferData:         offerData,
		}
	}
	return json.Marshal(val)
}

// decDSVal decodes the ds value into all active outbound channels excluding channel state.
//
// @input - ds value.
//
// @output - active outbound channel states.
func decDSVal(val []byte) (*cachedPaychState, error) {
	type subValJson struct {
		InboundPPP        *big.Int      `json:"inbound_ppp"`
		InboundPeriod     *big.Int      `json:"inbound_period"`
		PettyAmtReceived  *big.Int      `json:"petty_amt_received"`
		SurchargeReceived *big.Int      `json:"surcharge_received"`
		OutboundPPP       *big.Int      `json:"outbound_ppp"`
		OutboundPeriod    *big.Int      `json:"outbound_period"`
		PettyAmtPaid      *big.Int      `json:"petty_amt_paid"`
		SurchargePaid     *big.Int      `json:"surcharge_paid"`
		Remain            *big.Int      `json:"remain"`
		LastAccessed      time.Time     `json:"last_accessed"`
		Expiration        time.Time     `json:"expiration"`
		Inactivity        time.Duration `json:"inactivity"`
		OfferData         []byte        `json:"offer_data"`
	}
	type valJson struct {
		ReservedAmt   *big.Int              `json:"reserved_amt"`
		ReservedNonce uint64                `json:"reserved_nonce"`
		Reservations  map[uint64]subValJson `json:"reservations"`
	}
	valDec := valJson{}
	err := json.Unmarshal(val, &valDec)
	if err != nil {
		return nil, err
	}
	cs := &cachedPaychState{
		reservedAmt:   valDec.ReservedAmt,
		reservedNonce: valDec.ReservedNonce,
		reservations:  make(map[uint64]*reservation),
	}
	for resID, res := range valDec.Reservations {
		var offer *fcroffer.PayOffer
		if res.OfferData != nil {
			offer = &fcroffer.PayOffer{}
			err := offer.Decode(res.OfferData)
			if err != nil {
				return nil, err
			}
		}
		cs.reservations[resID] = &reservation{
			inboundPPP:        res.InboundPPP,
			inboundPeriod:     res.InboundPeriod,
			pettyAmtReceived:  res.PettyAmtReceived,
			surchargeReceived: res.SurchargeReceived,
			outboundPPP:       res.OutboundPPP,
			outboundPeriod:    res.OutboundPeriod,
			pettyAmtPaid:      res.PettyAmtPaid,
			surchargePaid:     res.SurchargePaid,
			remain:            res.Remain,
			lastAccessed:      res.LastAccessed,
			expiration:        res.Expiration,
			inactivity:        res.Inactivity,
			offer:             offer,
		}
	}
	return cs, nil
}
