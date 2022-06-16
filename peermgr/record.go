package peermgr

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
	"time"
)

// Record is a record of activity of a peer.
type Record struct {
	// Description describes the record.
	Description string

	// CreatedAt is the time when the record is created.
	CreatedAt time.Time
}

// Encode encodes a record.
//
// @output - data, error.
func (r Record) Encode() ([]byte, error) {
	type valJson struct {
		Description string    `json:"description"`
		CreatedAt   time.Time `json:"created_at"`
	}
	return json.Marshal(valJson{
		Description: r.Description,
		CreatedAt:   r.CreatedAt,
	})
}

// DecodeRecord decodes a record data bytes.
//
// @input - data.
//
// @output - record, error.
func DecodeRecord(data []byte) (Record, error) {
	type valJson struct {
		Description string    `json:"description"`
		CreatedAt   time.Time `json:"created_at"`
	}
	valDec := valJson{}
	err := json.Unmarshal(data, &valDec)
	if err != nil {
		return Record{}, err
	}
	return Record{Description: valDec.Description, CreatedAt: valDec.CreatedAt}, nil
}
