package api

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
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/filecoin-project/go-jsonrpc"
	logging "github.com/ipfs/go-log"
	"github.com/wcgcyx/fcr/node"
)

// Logger
var log = logging.Logger("apiserver")

// Server is the api server.
type Server struct {
	s http.Server
}

// NewServer creates a new API server.
//
// @input - node, port, dev mode, token.
//
// @output - server, error.
func NewServer(node *node.Node, port int, dev bool, tokenPath string) (*Server, error) {
	log.Infof("Start API server...")
	token, err := loadOrCreate(tokenPath)
	if err != nil {
		return nil, err
	}
	// New jsonrpc server
	rpc := jsonrpc.NewServer()
	userHandle := userAPIHandler{
		node: node,
	}
	if !dev {
		rpc.Register("FCR", &userHandle)
	} else {
		devHandle := devHandler{
			userHandle,
			node,
		}
		rpc.Register("FCR-DEV", &devHandle)
		log.Warnf("Dev APIs are served.")
	}
	s := http.Server{
		Addr: fmt.Sprintf("localhost:%v", port),
		Handler: &AuthHandler{
			token: token,
			Next:  rpc.ServeHTTP,
		},
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	errChan := make(chan error, 1)
	go func() {
		// Start server.
		errChan <- s.ListenAndServe()
	}()
	// Wait for 3 seconds for the server to start
	tc := time.After(3 * time.Second)
	select {
	case <-tc:
		return &Server{s}, nil
	case err := <-errChan:
		return nil, err
	}
}

// Shutdown safely shuts down the component.
func (s *Server) Shutdown() {
	log.Infof("Start shutdown...")
	err := s.s.Shutdown(context.Background())
	if err != nil {
		log.Errorf("Fail to shutdown API server: %v", err.Error())
	}
}

// loadOrCreate is used to load a token from a token path or if not exists, create.
//
// @input - token path.
//
// @output - token, error.
func loadOrCreate(tokenPath string) (string, error) {
	tokenFile := path.Join(tokenPath, "token")
	token := ""
	// Check if file exists.
	_, err := os.Stat(tokenFile)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		// Not exist, generate and save a token.
		// For now, just use random 32 bytes hex string.
		// TODO: Move to JWT in the future.
		data := make([]byte, 32)
		_, err = rand.Read(data)
		if err != nil {
			return "", err
		}
		token = hex.EncodeToString(data)
		// Save
		err = os.WriteFile(tokenFile, []byte(token), os.ModePerm)
		if err != nil {
			return "", err
		}
	} else {
		// Load token
		data, err := os.ReadFile(tokenFile)
		if err != nil {
			return "", err
		}
		token = string(data)
	}
	return token, nil
}
