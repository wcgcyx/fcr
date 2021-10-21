package messages

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPublishRoundTrip(t *testing.T) {
	testRoute1 := []string{"A", "B", "C"}
	testRoute2 := []string{"D", "E", "F", "G"}

	data := EncodePublish(1, [][]string{
		testRoute1,
		testRoute2,
	})
	currencyID, routes, err := DecodePublish(data)
	assert.Empty(t, err)
	assert.Equal(t, uint64(1), currencyID)
	assert.Equal(t, [][]string{testRoute1, testRoute2}, routes)
}
