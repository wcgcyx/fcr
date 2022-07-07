package sreader

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
	"io"

	"github.com/filecoin-project/lotus/storage/sealer/fr32"
)

// SectorReader is a relatively memory efficient reader to
// read blocks from a filecoin lotus generated unsealed sector copy.
type SectorReader struct {
	ptr         uint64
	bufPtr      int
	buf         []byte
	unpadReader io.Reader
}

// NewSectorReader creates a new SectorReader instance.
//
// @input - io reader.
//
// @output - sector reader, error.
func NewSectorReader(f io.Reader) (*SectorReader, error) {
	r, err := fr32.NewUnpadReader(f, 64<<30)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 127)
	m, err := r.Read(buf)
	if err != nil {
		return nil, err
	}
	if m != 127 {
		return nil, io.EOF
	}
	return &SectorReader{
		ptr:         0,
		bufPtr:      0,
		buf:         buf,
		unpadReader: r,
	}, nil
}

// Read reads p bytes.
//
// @input - input byte slice.
//
// @output - bytes read, error.
func (sr *SectorReader) Read(p []byte) (n int, err error) {
	n = 0
	for n < len(p) {
		if sr.bufPtr == 127 {
			m, err := sr.unpadReader.Read(sr.buf)
			if err != nil {
				return 0, err
			}
			if m != 127 {
				return 0, io.EOF
			}
			sr.bufPtr = 0
			sr.ptr += 128
		}
		p[n] = sr.buf[sr.bufPtr]
		n++
		sr.bufPtr++
	}
	return n, nil
}

// Advance advances the current pointer to the start of next block.
//
// @output - error.
func (sr *SectorReader) Advance() error {
	for {
		if sr.bufPtr == 127 {
			m, err := sr.unpadReader.Read(sr.buf)
			if err != nil {
				return err
			}
			if m != 127 {
				return io.EOF
			}
			sr.bufPtr = 0
			sr.ptr += 128
		}
		if sr.buf[sr.bufPtr] == 0 {
			sr.bufPtr++
		} else {
			break
		}
	}
	return nil
}

// GetPos gets the current position of the main pointer.
//
// @output - position of the main pointer.
func (sr *SectorReader) GetPos() uint64 {
	return sr.ptr
}

// GetBufPos gets the current position of the sub pointer.
//
// @output - position of the sub pointer.
func (sr *SectorReader) GetBufPos() uint64 {
	return uint64(sr.bufPtr)
}
