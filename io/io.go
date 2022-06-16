package io

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
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/libp2p/go-libp2p-core/network"
)

const (
	MaxDataLength = 40960
)

// Read reads bytes from given stream.
//
// @input - context, network stream.
//
// @output - data, error.
func Read(ctx context.Context, conn network.Stream) ([]byte, error) {
	out := make(chan []byte)
	errChan := make(chan error)
	go func() {
		data, err := func() ([]byte, error) {
			// New reader
			r := bufio.NewReader(conn)
			// Read the length
			length := make([]byte, 2)
			_, err := io.ReadFull(r, length)
			if err != nil {
				return nil, err
			}
			size := binary.BigEndian.Uint16(length)
			if size > MaxDataLength {
				return nil, fmt.Errorf("Data exceeds maximum data length, got %v", size)
			}
			data := make([]byte, size)
			_, err = io.ReadFull(r, data)
			if err != nil {
				return nil, err
			}
			return data, err
		}()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case errChan <- err:
			}
		}
		select {
		case <-ctx.Done():
		case out <- data:
		}
	}()
	var data []byte
	var err error
	select {
	case <-ctx.Done():
		err = conn.Close()
		if err == nil {
			err = ctx.Err()
		}
	case data = <-out:
	case err = <-errChan:
	}
	return data, err
}

// Write writes bytes to given stream.
//
// @input - context, network stream, data.
//
// @output - error.
func Write(ctx context.Context, conn network.Stream, data []byte) error {
	errChan := make(chan error)
	go func() {
		err := func() error {
			size := len(data)
			if size > MaxDataLength {
				return fmt.Errorf("Data exceeds maximum data length, got %v", size)
			}
			// New writer
			w := bufio.NewWriter(conn)
			// Get the length
			length := make([]byte, 2)
			binary.BigEndian.PutUint16(length, uint16(size))
			// Write length and data
			_, err := w.Write(append(length, data...))
			if err != nil {
				return err
			}
			return w.Flush()
		}()
		select {
		case <-ctx.Done():
		case errChan <- err:
		}
	}()
	var err error
	select {
	case <-ctx.Done():
		err = conn.Close()
		if err == nil {
			err = ctx.Err()
		}
	case err = <-errChan:
	}
	return err
}
