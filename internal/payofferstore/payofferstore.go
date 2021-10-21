package payofferstore

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"github.com/ipfs/go-cid"
	"github.com/wcgcyx/fcr/internal/payoffer"
)

// PayOfferStore is used to store all the linked offers.
type PayOfferStore interface {
	// AddOffer is used to store a pay offer.
	// It takes a pay offer as the argument.
	// It returns error.
	AddOffer(offer *payoffer.PayOffer) error

	// GetOffer is used to get a pay offer.
	// It takes a offer id as the argument.
	// It returns the pay offer and error.
	GetOffer(id cid.Cid) (*payoffer.PayOffer, error)
}
