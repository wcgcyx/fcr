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
	"net/http"
	"strings"
)

// AuthHandler is used to verify incoming API call.
type AuthHandler struct {
	token string
	Next  http.HandlerFunc
}

// ServeHTTP is used to serve HTTP.
//
// @input - response writer, http request.
func (h *AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.FormValue("token")
		if token != "" {
			token = "Bearer " + token
		}
	}

	if token != "" {
		if !strings.HasPrefix(token, "Bearer ") {
			log.Warn("missing Bearer prefix in auth header")
			w.WriteHeader(401)
			return
		}
		token = strings.TrimPrefix(token, "Bearer ")

		if token != h.token {
			log.Warnf("JWT Verification failed (originating from %s): token %s not allowed", r.RemoteAddr, token)
			w.WriteHeader(401)
			return
		}
	} else {
		log.Warnf("JWT Verification failed (originating from %s): empty token", r.RemoteAddr)
		w.WriteHeader(401)
		return
	}

	h.Next(w, r.WithContext(ctx))
}
