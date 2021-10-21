package io

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"bufio"
	"encoding/binary"
	"io"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
)

// Read reads bytes from given stream.
func Read(conn network.Stream, timeout time.Duration) ([]byte, error) {
	// Set timeout
	err := conn.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		return nil, err
	}
	// New reader
	r := bufio.NewReader(conn)
	// Read the length
	length := make([]byte, 2)
	_, err = io.ReadFull(r, length)
	if err != nil {
		return nil, err
	}
	// Read the data
	data := make([]byte, binary.BigEndian.Uint16(length))
	_, err = io.ReadFull(r, data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Write writes bytes to given stream.
func Write(conn network.Stream, data []byte, timeout time.Duration) error {
	// Set timeout
	err := conn.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		return err
	}
	// New writer
	w := bufio.NewWriter(conn)
	// Get the length
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(data)))
	// Write length and data
	_, err = w.Write(append(length, data...))
	if err != nil {
		return err
	}
	return w.Flush()
}
