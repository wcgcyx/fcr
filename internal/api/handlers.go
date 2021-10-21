package api

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/wcgcyx/fcr/internal/cidoffer"
	"github.com/wcgcyx/fcr/internal/paychoffer"
	"github.com/wcgcyx/fcr/internal/payoffer"
)

// handleCIDNetQueryOffer handles cidnet query offer request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleCIDNetQueryOffer(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(a.ctx, contextTimeout)
	defer cancel()

	req := cidNetQueryOfferRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	id, err := cid.Parse(req.CID)
	if err != nil {
		return nil, fmt.Errorf("error parsing cid: %v", err.Error())
	}

	offers := a.node.CIDNetworkManager().SearchOffers(ctx, req.CurrencyID, id, req.Max)
	offerStrs := make([]string, 0)
	for _, offer := range offers {
		offerStrs = append(offerStrs, hex.EncodeToString(offer.ToBytes()))
	}
	return []byte(strings.Join(offerStrs, ";")), nil
}

// handleCIDNetQueryConnectedOffer handles cidnet query connected offer request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleCIDNetQueryConnectedOffer(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(a.ctx, contextTimeout)
	defer cancel()

	req := cidNetQueryConnectedOfferRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	id, err := cid.Parse(req.CID)
	if err != nil {
		return nil, fmt.Errorf("error parsing cid: %v", err.Error())
	}

	outsMap, err := a.node.PaymentNetworkManager().ListActiveOutboundChs(ctx)
	if err != nil {
		return nil, err
	}

	offerStrs := make([]string, 0)
	lock := &sync.RWMutex{}
	outs, ok := outsMap[req.CurrencyID]
	if ok {
		var wg sync.WaitGroup
		for _, out := range outs {
			wg.Add(1)
			go func(toAddr string) {
				defer wg.Done()
				pid, blocked, _, _, err := a.node.PaymentNetworkManager().GetPeerInfo(req.CurrencyID, toAddr)
				if err != nil || blocked {
					return
				}
				offer, err := a.node.CIDNetworkManager().QueryOffer(ctx, pid, req.CurrencyID, id)
				if err != nil {
					return
				}
				lock.Lock()
				offerStrs = append(offerStrs, hex.EncodeToString(offer.ToBytes()))
				lock.Unlock()
			}(out)
		}
		// Wait for all routine to finish
		wg.Wait()
	}
	return []byte(strings.Join(offerStrs, ";")), nil
}

// handleCIDNetPieceImport handles cidnet piece import request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleCIDNetPieceImport(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(a.ctx, contextTimeout)
	defer cancel()

	filename := string(data)
	id, err := a.node.CIDNetworkManager().Import(ctx, filename)
	if err != nil {
		return nil, fmt.Errorf("error importing %v", err.Error())
	}
	return []byte(id.String()), nil
}

// handleCIDNetPieceImportCar handles cidnet piece import car request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleCIDNetPieceImportCar(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(a.ctx, contextTimeout)
	defer cancel()

	filename := string(data)
	id, err := a.node.CIDNetworkManager().ImportCar(ctx, filename)
	if err != nil {
		return nil, fmt.Errorf("error importing car %v", err.Error())
	}
	return []byte(id.String()), nil
}

// handleCIDNetPieceImportSector handles cidnet piece import sector request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleCIDNetPieceImportSector(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(a.ctx, contextTimeout)
	defer cancel()

	copy := false
	if data[0] == 1 {
		copy = true
	}
	filename := string(data[1:])
	ids, err := a.node.CIDNetworkManager().ImportSector(ctx, filename, copy)
	if err != nil {
		return nil, fmt.Errorf("error importing sector %v", err.Error())
	}
	res := make([]string, 0)
	for _, id := range ids {
		res = append(res, id.String())
	}
	return []byte(strings.Join(res, ";")), nil
}

// handleCIDNetPieceList handles cidnet piece list request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleCIDNetPieceList(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(a.ctx, contextTimeout)
	defer cancel()

	ids, err := a.node.CIDNetworkManager().ListImported(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing: %v", err.Error())
	}
	res := make([]string, 0)
	for _, id := range ids {
		res = append(res, id.String())
	}
	return []byte(strings.Join(res, ";")), nil
}

// cidNetPieceInspectResponseJson is used for serialization.
type cidNetPieceInspectResponseJson struct {
	Path  string `json:"path"`
	Index int    `json:"index"`
	Size  int64  `json:"size"`
	Copy  bool   `json:"copy"`
}

// handleCIDNetPieceInspect handles a cidnet piece inspect request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleCIDNetPieceInspect(data []byte) ([]byte, error) {
	idStr := string(data)
	id, err := cid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing cid: %v", err.Error())
	}
	path, index, size, copy, err := a.node.CIDNetworkManager().Inspect(id)
	if err != nil {
		return nil, fmt.Errorf("error inspecting cid: %v", err.Error())
	}
	return json.Marshal(cidNetPieceInspectResponseJson{
		Path:  path,
		Index: index,
		Size:  size,
		Copy:  copy,
	})
}

// handleCIDNetPieceRemove handles a cidnet piece remove request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleCIDNetPieceRemove(data []byte) ([]byte, error) {
	idStr := string(data)
	id, err := cid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing cid: %v", err.Error())
	}
	err = a.node.CIDNetworkManager().Remove(id)
	if err != nil {
		return nil, fmt.Errorf("error removing cid: %v", err.Error())
	}
	return []byte{0}, nil
}

// handleCIDNetServingServe handles a cidnet serving serve request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleCIDNetServingServe(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(a.ctx, contextTimeout)
	defer cancel()

	req := cidNetServingServeRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	id, err := cid.Parse(req.CID)
	if err != nil {
		return nil, fmt.Errorf("error parsing cid: %v", err.Error())
	}
	ppb, ok := big.NewInt(0).SetString(req.PPB, 10)
	if !ok {
		return nil, fmt.Errorf("fail to parse ppb as big int")
	}
	err = a.node.CIDNetworkManager().StartServing(ctx, req.CurrencyID, id, ppb)
	if err != nil {
		return nil, fmt.Errorf("error serving: %v", err.Error())
	}
	return []byte{0}, nil
}

// handleCIDNetServingList handles a cidnet serving list request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleCIDNetServingList(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(a.ctx, contextTimeout)
	defer cancel()

	res, err := a.node.CIDNetworkManager().ListServings(ctx)
	if err != nil {
		return nil, fmt.Errorf("error in listing: %v", err.Error())
	}
	resStr := make(map[uint64][]string)
	for currencyID, ids := range res {
		resStr[currencyID] = make([]string, 0)
		for _, id := range ids {
			resStr[currencyID] = append(resStr[currencyID], id.String())
		}
	}
	return json.Marshal(resStr)
}

// cidNetServingServeInspectJson is used for serialization.
type cidNetServingInspectResponseJson struct {
	PPB        string `json:"ppb"`
	Expiration int64  `json:"expiration"`
}

// handleCIDNetServingInspect handles a cidnet serving inspect request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleCIDNetServingInspect(data []byte) ([]byte, error) {
	req := cidNetServingInspectRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	id, err := cid.Parse(req.CID)
	if err != nil {
		return nil, fmt.Errorf("error parsing cid: %v", err.Error())
	}
	ppb, expiration, err := a.node.CIDNetworkManager().InspectServing(req.CurrencyID, id)
	if err != nil {
		return nil, fmt.Errorf("error inspecting serving: %v", err.Error())
	}
	return json.Marshal(cidNetServingInspectResponseJson{
		PPB:        ppb.String(),
		Expiration: expiration,
	})
}

// handleCIDNetServingRetire handles a cidnet serving retire request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleCIDNetServingRetire(data []byte) ([]byte, error) {
	req := cidNetServingRetireRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	id, err := cid.Parse(req.CID)
	if err != nil {
		return nil, fmt.Errorf("error parsing cid: %v", err.Error())
	}
	err = a.node.CIDNetworkManager().StopServing(req.CurrencyID, id)
	if err != nil {
		return nil, err
	}
	return []byte{0}, err
}

// handleCIDNetServingForcePublish handles a cidnet serving force publish request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleCIDNetServingForcePublish(data []byte) ([]byte, error) {
	a.node.CIDNetworkManager().ForcePublishServings()
	return []byte{0}, nil
}

// handleCIDNetServingList handles a cidnet serving list in use request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleCIDNetServingListInUse(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(a.ctx, contextTimeout)
	defer cancel()

	res, err := a.node.CIDNetworkManager().ListInUseServings(ctx)
	if err != nil {
		return nil, fmt.Errorf("error in listing: %v", err.Error())
	}
	resStr := make([]string, 0)
	for _, id := range res {
		resStr = append(resStr, id.String())
	}
	return []byte(strings.Join(resStr, ";")), nil
}

// handleCIDNetPeerList handles a cidnet peer list request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleCIDNetPeerList(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(a.ctx, contextTimeout)
	defer cancel()

	res, err := a.node.CIDNetworkManager().ListPeers(ctx)
	if err != nil {
		return nil, err
	}
	return json.Marshal(res)
}

// cidNetPeerInspectResponseJson is used for serialization.
type cidNetPeerInspectResponseJson struct {
	PeerID  string `json:"currency_id"`
	Blocked bool   `json:"blocked"`
	Success int    `json:"success"`
	Fail    int    `json:"fail"`
}

// handleCIDNetPeerInspect handles a cienet peer inspect request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleCIDNetPeerInspect(data []byte) ([]byte, error) {
	req := cidNetPeerInspectRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	id, blocked, success, fail, err := a.node.CIDNetworkManager().GetPeerInfo(req.CurrencyID, req.ToAddr)
	if err != nil {
		return nil, err
	}
	return json.Marshal(cidNetPeerInspectResponseJson{
		PeerID:  id.String(),
		Blocked: blocked,
		Success: success,
		Fail:    fail,
	})
}

// handleCIDNetPeerBlock handles a cidnet peer block request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleCIDNetPeerBlock(data []byte) ([]byte, error) {
	req := cidNetPeerInspectRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	err = a.node.CIDNetworkManager().BlockPeer(req.CurrencyID, req.ToAddr)
	if err != nil {
		return nil, err
	}
	return []byte{0}, nil
}

// handleCIDNetPeerUnblock handles a cidnet peer unblock request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleCIDNetPeerUnblock(data []byte) ([]byte, error) {
	req := cidNetPeerInspectRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	err = a.node.CIDNetworkManager().UnblockPeer(req.CurrencyID, req.ToAddr)
	if err != nil {
		return nil, err
	}
	return []byte{0}, nil
}

// handlePayNetRoot handles a peynet root request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetRoot(data []byte) ([]byte, error) {
	currencyID, err := strconv.ParseUint(string(data), 10, 64)
	if err != nil {
		return nil, err
	}
	root, err := a.node.PaymentNetworkManager().GetRootAddress(currencyID)
	if err != nil {
		return nil, err
	}
	return []byte(root), nil
}

// handlePayNetQueryOffer handles a paynet query offer request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetQueryOffer(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(a.ctx, contextTimeout)
	defer cancel()

	req := payNetQueryOfferRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	amt, ok := big.NewInt(0).SetString(req.Amt, 10)
	if !ok {
		return nil, fmt.Errorf("fail to parse amount")
	}
	offers, err := a.node.PaymentNetworkManager().SearchPayOffer(ctx, req.CurrencyID, req.ToAddr, amt)
	if err != nil {
		return nil, err
	}

	offerStrs := make([]string, 0)
	for _, offer := range offers {
		offerStrs = append(offerStrs, hex.EncodeToString(offer.ToBytes()))
	}
	return []byte(strings.Join(offerStrs, ";")), nil
}

// handlePayNetOutboundQueryPaych handles a paynet outbound query paych request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetOutboundQueryPaych(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(a.ctx, contextTimeout)
	defer cancel()

	req := payNetQueryOutboundPaychRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	piData, err := hex.DecodeString(req.PAddr)
	if err != nil {
		return nil, err
	}
	pi := &peer.AddrInfo{}
	err = pi.UnmarshalJSON(piData)
	if err != nil {
		return nil, err
	}
	offer, err := a.node.PaymentNetworkManager().QueryOutboundChOffer(ctx, req.CurrencyID, req.ToAddr, *pi)
	if err != nil {
		return nil, err
	}
	return offer.ToBytes(), nil
}

// handlePayNetOutboundCreate handles a paynet outbound create request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetOutboundCreate(data []byte) ([]byte, error) {
	// Use a bit more context timeout for chain to confirm.
	ctx, cancel := context.WithTimeout(a.ctx, 5*contextTimeout)
	defer cancel()

	req := payNetOutboundCreateRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	piData, err := hex.DecodeString(req.PAddr)
	if err != nil {
		return nil, err
	}
	pi := &peer.AddrInfo{}
	err = pi.UnmarshalJSON(piData)
	if err != nil {
		return nil, err
	}
	offerData, err := hex.DecodeString(req.Offer)
	if err != nil {
		return nil, err
	}
	offer, err := paychoffer.FromBytes(offerData)
	if err != nil {
		return nil, err
	}
	amt, ok := big.NewInt(0).SetString(req.Amt, 10)
	if !ok {
		return nil, fmt.Errorf("error parsing amt")
	}
	err = a.node.PaymentNetworkManager().CreateOutboundCh(ctx, offer, amt, *pi)
	if err != nil {
		return nil, err
	}
	return []byte{0}, nil
}

// handlePayNetOutboundTopup handles a paynet outbound topup request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetOutboundTopup(data []byte) ([]byte, error) {
	// Use a bit more context timeout for chain to confirm.
	ctx, cancel := context.WithTimeout(a.ctx, 5*contextTimeout)
	defer cancel()

	req := payNetOutboundTopupRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	amt, ok := big.NewInt(0).SetString(req.Amt, 10)
	if !ok {
		return nil, fmt.Errorf("error parsing amt")
	}
	err = a.node.PaymentNetworkManager().TopupOutboundCh(ctx, req.CurrencyID, req.ToAddr, amt)
	if err != nil {
		return nil, err
	}
	return []byte{0}, nil
}

// handlePayNetOutboundList handles a paynet outbound list request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetOutboundList(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(a.ctx, contextTimeout)
	defer cancel()

	res, err := a.node.PaymentNetworkManager().ListOutboundChs(ctx)
	if err != nil {
		return nil, fmt.Errorf("error in listing: %v", err.Error())
	}
	return json.Marshal(res)
}

// payNetOutboundInspectResponseJson is used for serialization.
type payNetOutboundInspectResponseJson struct {
	ToAddr     string `json:"to_addr"`
	Redeemed   string `json:"redeemed"`
	Balance    string `json:"balance"`
	Settlement int64  `json:"settlement"`
	Active     bool   `json:"active"`
	CurHeight  int64  `json:"cur_height"`
	SettlingAt int64  `json:"settling_at"`
}

// handlePayNetOutboundInspect handles a paynet outbound inspect request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetOutboundInspect(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(a.ctx, contextTimeout)
	defer cancel()

	req := payNetOutboundInspectRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	toAddr, redeemed, balance, settlement, active, curHeight, settlingAt, err := a.node.PaymentNetworkManager().InspectOutboundCh(ctx, req.CurrencyID, req.ChAddr)
	if err != nil {
		return nil, err
	}
	return json.Marshal(payNetOutboundInspectResponseJson{
		ToAddr:     toAddr,
		Redeemed:   redeemed.String(),
		Balance:    balance.String(),
		Settlement: settlement,
		Active:     active,
		CurHeight:  curHeight,
		SettlingAt: settlingAt,
	})
}

// handlePayNetOutboundSettle handles a paynet outbound settle request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetOutboundSettle(data []byte) ([]byte, error) {
	// Use a bit more context timeout for chain to confirm.
	ctx, cancel := context.WithTimeout(a.ctx, 5*contextTimeout)
	defer cancel()

	req := payNetOutboundSettleRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	err = a.node.PaymentNetworkManager().SettleOutboundCh(ctx, req.CurrencyID, req.ChAddr)
	if err != nil {
		return nil, err
	}
	return []byte{0}, nil
}

// handlePayNetOutboundCollect handles a paynet outbound collect request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetOutboundCollect(data []byte) ([]byte, error) {
	// Use a bit more context timeout for chain to confirm.
	ctx, cancel := context.WithTimeout(a.ctx, 5*contextTimeout)
	defer cancel()

	req := payNetOutboundCollectRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	err = a.node.PaymentNetworkManager().CollectOutboundCh(ctx, req.CurrencyID, req.ChAddr)
	if err != nil {
		return nil, err
	}
	return []byte{0}, nil
}

// handlePayNetInboundList handles a paynet inbound list request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetInboundList(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(a.ctx, contextTimeout)
	defer cancel()

	res, err := a.node.PaymentNetworkManager().ListInboundChs(ctx)
	if err != nil {
		return nil, fmt.Errorf("error in listing: %v", err.Error())
	}
	return json.Marshal(res)
}

// payNetInboundInspectResponseJson is used for serialization.
type payNetInboundInspectResponseJson struct {
	ToAddr     string `json:"to_addr"`
	Redeemed   string `json:"redeemed"`
	Balance    string `json:"balance"`
	Settlement int64  `json:"settlement"`
	Active     bool   `json:"active"`
	CurHeight  int64  `json:"cur_height"`
	SettlingAt int64  `json:"settling_at"`
}

// handlePayNetInboundInspect handles a paynet inbound inspect request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetInboundInspect(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(a.ctx, contextTimeout)
	defer cancel()

	req := payNetInboundInspectRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	toAddr, redeemed, balance, settlement, active, curHeight, settlingAt, err := a.node.PaymentNetworkManager().InspectInboundCh(ctx, req.CurrencyID, req.ChAddr)
	if err != nil {
		return nil, err
	}
	return json.Marshal(payNetInboundInspectResponseJson{
		ToAddr:     toAddr,
		Redeemed:   redeemed.String(),
		Balance:    balance.String(),
		Settlement: settlement,
		Active:     active,
		CurHeight:  curHeight,
		SettlingAt: settlingAt,
	})
}

// handlePayNetInboundSettle handles a paynet inbound settle request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetInboundSettle(data []byte) ([]byte, error) {
	// Use a bit more context timeout for chain to confirm.
	ctx, cancel := context.WithTimeout(a.ctx, 5*contextTimeout)
	defer cancel()

	req := payNetInboundSettleRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	err = a.node.PaymentNetworkManager().SettleInboundCh(ctx, req.CurrencyID, req.ChAddr)
	if err != nil {
		return nil, err
	}
	return []byte{0}, nil
}

// handlePayNetInboundCollect handles a paynet inbound collect request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetInboundCollect(data []byte) ([]byte, error) {
	// Use a bit more context timeout for chain to confirm.
	ctx, cancel := context.WithTimeout(a.ctx, 5*contextTimeout)
	defer cancel()

	req := payNetInboundCollectRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	err = a.node.PaymentNetworkManager().CollectInboundCh(ctx, req.CurrencyID, req.ChAddr)
	if err != nil {
		return nil, err
	}
	return []byte{0}, nil
}

// handlePayNetServingServe handles a paynet serving serve request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetServingServe(data []byte) ([]byte, error) {
	req := payNetServingServeRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	ppp, ok := big.NewInt(0).SetString(req.PPP, 10)
	if !ok {
		return nil, fmt.Errorf("fail to parse ppp as big int")
	}
	period, ok := big.NewInt(0).SetString(req.Period, 10)
	if !ok {
		return nil, fmt.Errorf("fail to parse period as big int")
	}
	err = a.node.PaymentNetworkManager().StartServing(req.CurrencyID, req.ToAddr, ppp, period)
	if err != nil {
		return nil, fmt.Errorf("error serving: %v", err.Error())
	}
	return []byte{0}, nil
}

// handlePayNetServingList handles a paynet serving list request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetServingList(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(a.ctx, contextTimeout)
	defer cancel()

	res, err := a.node.PaymentNetworkManager().ListServings(ctx)
	if err != nil {
		return nil, fmt.Errorf("error in listing: %v", err.Error())
	}
	return json.Marshal(res)
}

// payNetServingServeInspectJson is used for serialization.
type payNetServingInspectResponseJson struct {
	PPP        string `json:"ppp"`
	Period     string `json:"period"`
	Expiration int64  `json:"expiration"`
}

// handlePayNetServingInspect handles a paynet serving inspect request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetServingInspect(data []byte) ([]byte, error) {
	req := payNetServingInspectRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	ppp, period, expiration, err := a.node.PaymentNetworkManager().InspectServing(req.CurrencyID, req.ToAddr)
	if err != nil {
		return nil, fmt.Errorf("error inspecting serving: %v", err.Error())
	}
	return json.Marshal(payNetServingInspectResponseJson{
		PPP:        ppp.String(),
		Period:     period.String(),
		Expiration: expiration,
	})
}

// handlePayNetServingRetire handles a paynet serving retire request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetServingRetire(data []byte) ([]byte, error) {
	req := payNetServingRetireRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	err = a.node.PaymentNetworkManager().StopServing(req.CurrencyID, req.ToAddr)
	if err != nil {
		return nil, err
	}
	return []byte{0}, err
}

// handlePayNetServingForcePublish handles a paynet serving force publish request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetServingForcePublish(data []byte) ([]byte, error) {
	a.node.PaymentNetworkManager().ForcePublishServings()
	return []byte{0}, nil
}

// handlePayNetPeerList handles a paynet peer list request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetPeerList(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(a.ctx, contextTimeout)
	defer cancel()

	res, err := a.node.PaymentNetworkManager().ListPeers(ctx)
	if err != nil {
		return nil, err
	}
	return json.Marshal(res)
}

// payNetPeerInspectResponseJson is used for serialization.
type payNetPeerInspectResponseJson struct {
	PeerID  string `json:"currency_id"`
	Blocked bool   `json:"blocked"`
	Success int    `json:"success"`
	Fail    int    `json:"fail"`
}

// handlePayNetPeerInspect handles a cienet peer inspect request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetPeerInspect(data []byte) ([]byte, error) {
	req := payNetPeerInspectRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	id, blocked, success, fail, err := a.node.PaymentNetworkManager().GetPeerInfo(req.CurrencyID, req.ToAddr)
	if err != nil {
		return nil, err
	}
	return json.Marshal(payNetPeerInspectResponseJson{
		PeerID:  id.String(),
		Blocked: blocked,
		Success: success,
		Fail:    fail,
	})
}

// handlePayNetPeerBlock handles a paynet peer block request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetPeerBlock(data []byte) ([]byte, error) {
	req := payNetPeerInspectRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	err = a.node.PaymentNetworkManager().BlockPeer(req.CurrencyID, req.ToAddr)
	if err != nil {
		return nil, err
	}
	return []byte{0}, nil
}

// handlePayNetPeerUnblock handles a paynet peer unblock request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handlePayNetPeerUnblock(data []byte) ([]byte, error) {
	req := payNetPeerInspectRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}
	err = a.node.PaymentNetworkManager().UnblockPeer(req.CurrencyID, req.ToAddr)
	if err != nil {
		return nil, err
	}
	return []byte{0}, nil
}

// handleSystemAddr handles a system addr request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleSystemAddr(data []byte) ([]byte, error) {
	res := make([]string, 0)
	info := a.node.AddrInfo()
	for _, maddr := range info.Addrs {
		if manet.IsPrivateAddr(maddr) {
			// Put private addr behind
			res = append(res, maddr.String()+"/p2p/"+info.ID.String())
		} else {
			// Put public addr ahead
			res = append([]string{maddr.String() + "/p2p/" + info.ID.String()}, res...)
		}
	}
	return []byte(strings.Join(res, ";")), nil
}

// handleSystemBootstrap handles a system bootstrap request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleSystemBootstrap(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(a.ctx, contextTimeout)
	defer cancel()

	pi := &peer.AddrInfo{}
	err := pi.UnmarshalJSON(data)
	if err != nil {
		return nil, err
	}
	err = a.node.Connect(ctx, *pi)
	if err != nil {
		return nil, err
	}
	return []byte{0}, nil
}

// systemCacheSizeResponseJson is used for serialization.
type systemCacheSizeResponseJson struct {
	Size uint64 `json:"size"`
}

// handleSystemCacheSize handles a system cache size request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleSystemCacheSize(data []byte) ([]byte, error) {
	size, err := a.node.CIDNetworkManager().GetRetrievalCacheSize()
	if err != nil {
		return nil, err
	}
	return json.Marshal(systemCacheSizeResponseJson{
		Size: size,
	})
}

// handleSystemCachePrune handles a system cache prune request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleSystemCachePrune(data []byte) ([]byte, error) {
	err := a.node.CIDNetworkManager().CleanRetrievalCache()
	if err != nil {
		return nil, err
	}
	return []byte{0}, nil
}

// handleRetrievalCache handles a retrieval cache request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleRetrievalCache(data []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(a.ctx, contextTimeout)
	defer cancel()

	req := retrievalCacheRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}

	id, err := cid.Parse(req.CID)
	if err != nil {
		return nil, err
	}
	found, err := a.node.CIDNetworkManager().RetrieveFromCache(ctx, id, req.OutPath)
	if err != nil {
		return nil, err
	}
	if found {
		return []byte{1}, nil
	} else {
		return []byte{0}, nil
	}
}

// handleRetrieval handles a retrieval request.
// It takes a data bytes array as the argument.
// It returns response bytes and error.
func (a *apiServer) handleRetrieval(data []byte) ([]byte, error) {
	// Use longer context time for retrieval
	ctx, cancel := context.WithTimeout(a.ctx, 5*contextTimeout)
	defer cancel()

	req := retrievalRequestJson{}
	err := json.Unmarshal(data, &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling request, %v", err.Error())
	}

	cidOfferBytes, err := hex.DecodeString(req.CIDOffer)
	if err != nil {
		return nil, err
	}
	cidOffer, err := cidoffer.FromBytes(cidOfferBytes)
	if err != nil {
		return nil, err
	}
	var payOffer *payoffer.PayOffer
	if req.PayOffer != "" {
		payOfferBytes, err := hex.DecodeString(req.PayOffer)
		if err != nil {
			return nil, err
		}
		payOffer, err = payoffer.FromBytes(payOfferBytes)
		if err != nil {
			return nil, err
		}
	}
	err = a.node.CIDNetworkManager().Retrieve(ctx, *cidOffer, payOffer, req.OutPath)
	if err != nil {
		return nil, err
	}
	return []byte{0}, nil
}
