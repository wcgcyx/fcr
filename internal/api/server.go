package api

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/wcgcyx/fcr/internal/config"
	"github.com/wcgcyx/fcr/internal/node"
)

var log = logging.Logger("apiserver")

const (
	cidNetQueryOffer = iota + 1
	cidNetQueryConnectedOffer
	cidNetPieceImport
	cidNetPieceImportCar
	cidNetPieceImportSector
	cidNetPieceList
	cidNetPieceInspect
	cidNetPieceRemove
	cidNetServingServe
	cidNetServingList
	cidNetServingInspect
	cidNetServingRetire
	cidNetServingForcePublish
	cidNetServingListInUse
	cidNetPeerList
	cidNetPeerInspect
	cidNetPeerBlock
	cidNetPeerUnblock
	payNetRoot
	payNetQueryOffer
	payNetOutboundQueryPaych
	payNetOutboundCreate
	payNetOutboundTopup
	payNetOutboundList
	payNetOutboundInspect
	payNetOutboundSettle
	payNetOutboundCollect
	payNetInboundList
	payNetInboundInspect
	payNetInboundSettle
	payNetInboundCollect
	payNetServingServe
	payNetServingList
	payNetServingInspect
	payNetServingRetire
	payNetServingForcePublish
	payNetPeerList
	payNetPeerInspect
	payNetPeerBlock
	payNetPeerUnblock
	systemAddr
	systemBootstrap
	systemCacheSize
	systemCachePrune
	retrievalCacheAPI
	retrievalAPI

	contextTimeout = 1 * time.Minute
)

// apiServer serves api.
type apiServer struct {
	ctx context.Context

	node     *node.Node
	server   *http.Server
	handlers map[byte]func(data []byte) ([]byte, error)
}

// NewServer creates and starts a new api server.
// It takes a context, a configuration as arguments.
// It returns error.
func NewServer(ctx context.Context, conf config.Config, node *node.Node) error {
	a := &apiServer{
		ctx:      ctx,
		node:     node,
		handlers: make(map[byte]func(data []byte) ([]byte, error)),
	}
	a.server = &http.Server{
		Addr:           fmt.Sprintf(":%v", conf.APIPort),
		Handler:        a,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	// Add handlers
	a.handlers[cidNetQueryOffer] = a.handleCIDNetQueryOffer
	a.handlers[cidNetQueryConnectedOffer] = a.handleCIDNetQueryConnectedOffer
	a.handlers[cidNetPieceImport] = a.handleCIDNetPieceImport
	a.handlers[cidNetPieceImportCar] = a.handleCIDNetPieceImportCar
	a.handlers[cidNetPieceImportSector] = a.handleCIDNetPieceImportSector
	a.handlers[cidNetPieceList] = a.handleCIDNetPieceList
	a.handlers[cidNetPieceInspect] = a.handleCIDNetPieceInspect
	a.handlers[cidNetPieceRemove] = a.handleCIDNetPieceRemove
	a.handlers[cidNetServingServe] = a.handleCIDNetServingServe
	a.handlers[cidNetServingList] = a.handleCIDNetServingList
	a.handlers[cidNetServingInspect] = a.handleCIDNetServingInspect
	a.handlers[cidNetServingRetire] = a.handleCIDNetServingRetire
	a.handlers[cidNetServingForcePublish] = a.handleCIDNetServingForcePublish
	a.handlers[cidNetServingListInUse] = a.handleCIDNetServingListInUse
	a.handlers[cidNetPeerList] = a.handleCIDNetPeerList
	a.handlers[cidNetPeerInspect] = a.handleCIDNetPeerInspect
	a.handlers[cidNetPeerBlock] = a.handleCIDNetPeerBlock
	a.handlers[cidNetPeerUnblock] = a.handleCIDNetPeerUnblock
	a.handlers[payNetRoot] = a.handlePayNetRoot
	a.handlers[payNetQueryOffer] = a.handlePayNetQueryOffer
	a.handlers[payNetOutboundQueryPaych] = a.handlePayNetOutboundQueryPaych
	a.handlers[payNetOutboundCreate] = a.handlePayNetOutboundCreate
	a.handlers[payNetOutboundTopup] = a.handlePayNetOutboundTopup
	a.handlers[payNetOutboundList] = a.handlePayNetOutboundList
	a.handlers[payNetOutboundInspect] = a.handlePayNetOutboundInspect
	a.handlers[payNetOutboundSettle] = a.handlePayNetOutboundSettle
	a.handlers[payNetOutboundCollect] = a.handlePayNetOutboundCollect
	a.handlers[payNetInboundList] = a.handlePayNetInboundList
	a.handlers[payNetInboundInspect] = a.handlePayNetInboundInspect
	a.handlers[payNetInboundSettle] = a.handlePayNetInboundSettle
	a.handlers[payNetInboundCollect] = a.handlePayNetInboundCollect
	a.handlers[payNetServingServe] = a.handlePayNetServingServe
	a.handlers[payNetServingList] = a.handlePayNetServingList
	a.handlers[payNetServingInspect] = a.handlePayNetServingInspect
	a.handlers[payNetServingRetire] = a.handlePayNetServingRetire
	a.handlers[payNetServingForcePublish] = a.handlePayNetServingForcePublish
	a.handlers[payNetPeerList] = a.handlePayNetPeerList
	a.handlers[payNetPeerInspect] = a.handlePayNetPeerInspect
	a.handlers[payNetPeerBlock] = a.handlePayNetPeerBlock
	a.handlers[payNetPeerUnblock] = a.handlePayNetPeerUnblock
	a.handlers[systemAddr] = a.handleSystemAddr
	a.handlers[systemBootstrap] = a.handleSystemBootstrap
	a.handlers[systemCacheSize] = a.handleSystemCacheSize
	a.handlers[systemCachePrune] = a.handleSystemCachePrune
	a.handlers[retrievalCacheAPI] = a.handleRetrievalCache
	a.handlers[retrievalAPI] = a.handleRetrieval
	errChan := make(chan error)
	go func() {
		// Start server.
		errChan <- a.server.ListenAndServe()
	}()
	// Wait for 3 seconds for the server to start
	tc := time.After(3 * time.Second)
	select {
	case <-tc:
		return nil
	case err := <-errChan:
		return err
	}
}

// ServeHTTP handles the api calls.
// It takes a response writer, the http request as arguments.
func (a *apiServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	content, err := ioutil.ReadAll(r.Body)
	if closeErr := r.Body.Close(); closeErr != nil {
		log.Warn("HTTP can't close request body")
	}
	if err != nil {
		log.Errorf("Error reading request %v", err.Error())
		WriteError(w, 400, fmt.Sprintf("Invalid Request: %v.", err.Error()))
		return
	}
	if len(content) <= 1 {
		log.Errorf("Received content with empty request %v", content)
		WriteError(w, 400, "Content body is empty")
		return
	}
	msgType := content[0]
	msgData := content[1:]
	handler, ok := a.handlers[msgType]
	if !ok {
		log.Errorf("Unsupported message type: %v", msgType)
		WriteError(w, 400, fmt.Sprintf("Unsupported method: %v", msgType))
		return
	}
	respData, err := handler(msgData)
	if err != nil {
		log.Errorf("Error handling request: %v", err.Error())
		WriteError(w, 400, fmt.Sprintf("Error handling request: %v", err.Error()))
		return
	}
	w.WriteHeader(200)
	_, err = w.Write(respData)
	if err != nil {
		log.Errorf("Error responding to client: %v", err.Error())
	}
}

// WriteError writes an error.
// It takes a response writer, a header and the error string as arguments.
func WriteError(w http.ResponseWriter, header int, msg string) {
	w.WriteHeader(header)
	resp, err := json.Marshal(map[string]string{"Error": msg})
	if err == nil {
		w.Write(resp)
	}
}

// Request sends a request to given addr with given key, msg type and data.
// It takes an addr, a message type, data as arguments.
// It returns the response in bytes array and error.
func Request(addr string, msgType byte, data []byte) ([]byte, error) {
	if !strings.HasPrefix(addr, "http://") {
		addr = "http://" + addr
	}
	req, err := http.NewRequest("POST", addr, bytes.NewReader(append([]byte{msgType}, data...)))
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 90 * time.Second}
	r, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(r.Body)
	if closeErr := r.Body.Close(); closeErr != nil {
		return nil, closeErr
	}
	return content, nil
}
