package paychstate

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
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoundTrip(t *testing.T) {
	s1 := State{
		CurrencyID:         1,
		FromAddr:           "test-from-addr",
		ToAddr:             "test-to-addr",
		ChAddr:             "test-ch-addr",
		Balance:            big.NewInt(100),
		Redeemed:           big.NewInt(99),
		Nonce:              1,
		Voucher:            "voucher",
		NetworkLossVoucher: "test-loss",
		StateNonce:         10,
	}
	data, err := s1.Encode()
	assert.Nil(t, err)
	s2 := State{}
	err = s2.Decode(data)
	assert.Nil(t, err)
	assert.Equal(t, s1, s2)
}
