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
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/filecoin-project/go-jsonrpc"
)

// NewClient creates a new client.
//
// @input - context, port, token file.
//
// @output - user API client, closer, error.
func NewClient(ctx context.Context, port int, tokenFile string) (UserAPI, jsonrpc.ClientCloser, error) {
	var client UserAPI
	token, err := getToken(tokenFile)
	if err != nil {
		return UserAPI{}, nil, err
	}
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+token)
	closer, err := jsonrpc.NewClient(ctx, fmt.Sprintf("ws://localhost:%v", port), "FCR", &client, headers)
	return client, closer, err
}

// NewDevClient creates a new dev client.
//
// @input - context, port, token file.
//
// @output - dev API client, closer, error.
func NewDevClient(ctx context.Context, port int, tokenFile string) (DevAPI, jsonrpc.ClientCloser, error) {
	var client DevAPI
	token, err := getToken(tokenFile)
	if err != nil {
		return DevAPI{}, nil, err
	}
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+token)
	closer, err := jsonrpc.NewClient(ctx, fmt.Sprintf("ws://localhost:%v", port), "FCR-DEV", &client, headers)
	return client, closer, err
}

// getToken is used to get access token from given token file.
//
// @input - input file.
//
// @output - token, error.
func getToken(tokenFile string) (string, error) {
	var err error
	if tokenFile == "" {
		// Use default token file.
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		tokenFile = path.Join(home, ".fcr/token")
	}
	token, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", err
	}
	return string(token), nil
}
