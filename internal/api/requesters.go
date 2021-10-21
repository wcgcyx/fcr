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
	"strings"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wcgcyx/fcr/internal/cidoffer"
	"github.com/wcgcyx/fcr/internal/paychoffer"
	"github.com/wcgcyx/fcr/internal/payoffer"
)

// cidNetQueryOfferRequestJson is used for serialization.
type cidNetQueryOfferRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	CID        string `json:"cid"`
	Max        int    `json:"max"`
}

// RequestCIDNetQueryOffer requests a cidnet query offer.
// It takes a context, the api addr, a currency id, a content id, max as arguments.
// It returns a list of cid offers and error.
func RequestCIDNetQueryOffer(ctx context.Context, apiAddr string, currencyID uint64, id string, max int) ([]cidoffer.CIDOffer, error) {
	data, _ := json.Marshal(cidNetQueryOfferRequestJson{
		CurrencyID: currencyID,
		CID:        id,
		Max:        max,
	})
	data, err := Request(apiAddr, cidNetQueryOffer, data)
	if err != nil {
		return nil, err
	}
	offerStrs := strings.Split(string(data), ";")
	offers := make([]cidoffer.CIDOffer, 0)
	for _, offerStr := range offerStrs {
		if offerStr == "" {
			continue
		}
		offerBytes, err := hex.DecodeString(offerStr)
		if err != nil {
			return nil, err
		}
		offer, err := cidoffer.FromBytes(offerBytes)
		if err != nil {
			return nil, err
		}
		offers = append(offers, *offer)
	}
	return offers, nil
}

// cidNetQueryConnectedOfferRequestJson is used for serialization.
type cidNetQueryConnectedOfferRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	CID        string `json:"cid"`
}

// RequestCIDNetQueryConnectedOffer requests a cidnet query connected offer.
// It takes a context, the api addr, a currency id, a content id, max as arguments.
// It returns a list of cid offers and error.
func RequestCIDNetQueryConnectedOffer(ctx context.Context, apiAddr string, currencyID uint64, id string) ([]cidoffer.CIDOffer, error) {
	data, _ := json.Marshal(cidNetQueryConnectedOfferRequestJson{
		CurrencyID: currencyID,
		CID:        id,
	})
	data, err := Request(apiAddr, cidNetQueryConnectedOffer, data)
	if err != nil {
		return nil, err
	}
	offerStrs := strings.Split(string(data), ";")
	offers := make([]cidoffer.CIDOffer, 0)
	for _, offerStr := range offerStrs {
		if offerStr == "" {
			continue
		}
		offerBytes, err := hex.DecodeString(offerStr)
		if err != nil {
			return nil, err
		}
		offer, err := cidoffer.FromBytes(offerBytes)
		if err != nil {
			return nil, err
		}
		offers = append(offers, *offer)
	}
	return offers, nil
}

// RequestCIDNetPieceImport requests a cidnet piece import.
// It takes a context, the api addr, a filename as arguments.
// It returns the cid imported in string and error.
func RequestCIDNetPieceImport(ctx context.Context, apiAddr string, filename string) (string, error) {
	id, err := Request(apiAddr, cidNetPieceImport, []byte(filename))
	if err != nil {
		return "", err
	}
	return string(id), nil
}

// RequestCIDNetPieceImportCar requests a cidnet piece import car.
// It takes a context, the api addr, a filename as arguments.
// It returns the cid imported in string and error.
func RequestCIDNetPieceImportCar(ctx context.Context, apiAddr string, filename string) (string, error) {
	id, err := Request(apiAddr, cidNetPieceImportCar, []byte(filename))
	if err != nil {
		return "", err
	}
	return string(id), nil
}

// RequestCIDNetPieceImportSector requests a cidnet piece import sector.
// It takes a context, the api addr, a filename, a copy as arguments.
// It returns the cids imported in string array and error.
func RequestCIDNetPieceImportSector(ctx context.Context, apiAddr string, filename string, copy bool) ([]string, error) {
	copyByte := 0
	if copy {
		copyByte = 1
	}
	ids, err := Request(apiAddr, cidNetPieceImportSector, append([]byte{byte(copyByte)}, []byte(filename)...))
	if err != nil {
		return nil, err
	}
	return strings.Split(string(ids), ";"), nil
}

// RequestCIDNetPieceList requests a cidnet piece list.
// It takes a context, the api addr as arguments.
// It returns the cids listed in string array and error.
func RequestCIDNetPieceList(ctx context.Context, apiAddr string) ([]string, error) {
	ids, err := Request(apiAddr, cidNetPieceList, []byte{0})
	if err != nil {
		return nil, err
	}
	return strings.Split(string(ids), ";"), nil
}

// RequestCIDNetPieceInspect requests a cidnet piece inspect.
// It takes a context, the api addr, a content id as arguments.
// It returns the path, the index, the size, a boolean indicating if a copy is kept and error.
func RequestCIDNetPieceInspect(ctx context.Context, apiAddr string, id string) (string, int, int64, bool, error) {
	data, err := Request(apiAddr, cidNetPieceInspect, []byte(id))
	if err != nil {
		return "", 0, 0, false, err
	}
	resp := cidNetPieceInspectResponseJson{}
	err = json.Unmarshal(data, &resp)
	return resp.Path, resp.Index, resp.Size, resp.Copy, err
}

// RequestCIDNetPieceRemove requests a cidnet piece remove.
// It takes a context, the api addr, a content id as arguments.
// It returns the error.
func RequestCIDNetPieceRemove(ctx context.Context, apiAddr string, id string) error {
	_, err := Request(apiAddr, cidNetPieceInspect, []byte(id))
	if err != nil {
		return err
	}
	return nil
}

// cidNetServingServeRequestJson is used for serialization.
type cidNetServingServeRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	CID        string `json:"cid"`
	PPB        string `json:"ppb"`
}

// RequestCIDNetServing requests a cidnet serving serve.
// It takes a context, the api addr, a currency id, a content id, a ppb (price per byte) as arguments.
// It returns the error.
func RequestCIDNetServingServe(ctx context.Context, apiAddr string, currencyID uint64, id string, ppb string) error {
	data, _ := json.Marshal(cidNetServingServeRequestJson{
		CurrencyID: currencyID,
		CID:        id,
		PPB:        ppb,
	})
	data, err := Request(apiAddr, cidNetServingServe, data)
	if err != nil {
		return err
	}
	if data[0] != 0 {
		return fmt.Errorf("error serving")
	}
	return nil
}

// RequestCIDNetServingList requests a cidnet serving list.
// It takes a context, the api addr as arguments.
// It returns a map from currency id to servings and the error.
func RequestCIDNetServingList(ctx context.Context, apiAddr string) (map[uint64][]string, error) {
	data, err := Request(apiAddr, cidNetServingList, []byte{0})
	if err != nil {
		return nil, err
	}
	var res map[uint64][]string
	err = json.Unmarshal(data, &res)
	return res, err
}

// cidNetServingInspectRequestJson is used for serialization.
type cidNetServingInspectRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	CID        string `json:"cid"`
}

// RequestCIDNetServingInspect requests a cidnet serving inspect.
// It takes a context, the api addr, a currency id, a content id as arguments.
// It returns the ppb (price per byte), the expiration and error.
func RequestCIDNetServingInspect(ctx context.Context, apiAddr string, currencyID uint64, id string) (*big.Int, int64, error) {
	data, _ := json.Marshal(cidNetServingInspectRequestJson{
		CurrencyID: currencyID,
		CID:        id,
	})
	data, err := Request(apiAddr, cidNetServingInspect, data)
	if err != nil {
		return nil, 0, err
	}
	resp := cidNetServingInspectResponseJson{}
	err = json.Unmarshal(data, &resp)
	ppb, _ := big.NewInt(0).SetString(resp.PPB, 10)
	return ppb, resp.Expiration, err
}

// cidNetServingRetireRequestJson is used for serialization.
type cidNetServingRetireRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	CID        string `json:"cid"`
}

// RequestCIDNetServingRetire requests a cidnet serving retire.
// It takes a context, the api addr, a currency id, a content id as arguments.
// It returns the error.
func RequestCIDNetServingRetire(ctx context.Context, apiAddr string, currencyID uint64, id string) error {
	data, _ := json.Marshal(cidNetServingInspectRequestJson{
		CurrencyID: currencyID,
		CID:        id,
	})
	data, err := Request(apiAddr, cidNetServingInspect, data)
	if err != nil {
		return err
	}
	if data[0] != 0 {
		return fmt.Errorf("error retiring")
	}
	return nil
}

// RequestCIDNetServingForcePublish requests a cidnet serving force publish.
// It takes a context, the api addr as arguments.
// It returns the error.
func RequestCIDNetServingForcePublish(ctx context.Context, apiAddr string) error {
	data, err := Request(apiAddr, cidNetServingForcePublish, []byte{0})
	if err != nil {
		return err
	}
	if data[0] != 0 {
		return fmt.Errorf("error force publish")
	}
	return nil
}

// RequestCIDNetServingListInUse requests a cidnet serving list in use.
// It takes a context, the api addr as arguments.
// It returns the cids in use as a string array and error.
func RequestCIDNetServingListInUse(ctx context.Context, apiAddr string) ([]string, error) {
	ids, err := Request(apiAddr, cidNetServingListInUse, []byte{0})
	if err != nil {
		return nil, err
	}
	return strings.Split(string(ids), ";"), nil
}

// RequestCIDNetPeerList requests a cidnet peer list.
// It takes a context, the api addr as arguments.
// It returns a map from currency id to peer addrs and the error.
func RequestCIDNetPeerList(ctx context.Context, apiAddr string) (map[uint64][]string, error) {
	data, err := Request(apiAddr, cidNetPeerList, []byte{0})
	if err != nil {
		return nil, err
	}
	var res map[uint64][]string
	err = json.Unmarshal(data, &res)
	return res, err
}

// cidNetPeerInspectRequestJson is used for serialization.
type cidNetPeerInspectRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	ToAddr     string `json:"to_addr"`
}

// RequestCIDNetPeerInspect requests a cidnet peer inspect.
// It takes a context, the api addr, currency id, recipient address as arguments.
// It returns peer id, a boolean indicating if peer is blocked, success count, failed count and error.
func RequestCIDNetPeerInspect(ctx context.Context, apiAddr string, currencyID uint64, toAddr string) (string, bool, int, int, error) {
	data, _ := json.Marshal(cidNetPeerInspectRequestJson{
		CurrencyID: currencyID,
		ToAddr:     toAddr,
	})
	data, err := Request(apiAddr, cidNetPeerInspect, data)
	if err != nil {
		return "", false, 0, 0, err
	}
	resp := cidNetPeerInspectResponseJson{}
	err = json.Unmarshal(data, &resp)
	return resp.PeerID, resp.Blocked, resp.Success, resp.Fail, err
}

// cidNetPeerBlockRequestJson is used for serialization.
type cidNetPeerBlockRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	ToAddr     string `json:"to_addr"`
}

// RequestCIDNetPeerBlock requests a cidnet peer block.
// It takes a context, the api addr, currency id, recipient address as arguments.
// It returns the error.
func RequestCIDNetPeerBlock(ctx context.Context, apiAddr string, currencyID uint64, toAddr string) error {
	data, _ := json.Marshal(cidNetPeerBlockRequestJson{
		CurrencyID: currencyID,
		ToAddr:     toAddr,
	})
	_, err := Request(apiAddr, cidNetPeerBlock, data)
	return err
}

// cidNetPeerUnblockRequestJson is used for serialization.
type cidNetPeerUnblockRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	ToAddr     string `json:"to_addr"`
}

// RequestCIDNetPeerUnblock requests a cidnet peer unblock.
// It takes a context, the api addr, currency id, recipient address as arguments.
// It returns the error.
func RequestCIDNetPeerUnblock(ctx context.Context, apiAddr string, currencyID uint64, toAddr string) error {
	data, _ := json.Marshal(cidNetPeerUnblockRequestJson{
		CurrencyID: currencyID,
		ToAddr:     toAddr,
	})
	_, err := Request(apiAddr, cidNetPeerUnblock, data)
	return err
}

// RequestPayNetRoot requests a paynet root.
// It takes a context, the api addr, currency id as arguments.
// It returns the root address and error.
func RequestPayNetRoot(ctx context.Context, apiAddr string, currencyID uint64) (string, error) {
	data, err := Request(apiAddr, payNetRoot, []byte(fmt.Sprintf("%v", currencyID)))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// payNetQueryOfferRequestJson is used for serialization.
type payNetQueryOfferRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	ToAddr     string `json:"to_addr"`
	Amt        string `json:"amt"`
}

// RequestPayNetQueryOffer requests a paynet query offer.
// It takes a context, the api addr, currency id, recipient address, amount as arguments.
// It returns a list of pay offers and error.
func RequestPayNetQueryOffer(ctx context.Context, apiAddr string, currencyID uint64, toAddr string, amt string) ([]payoffer.PayOffer, error) {
	data, _ := json.Marshal(payNetQueryOfferRequestJson{
		CurrencyID: currencyID,
		ToAddr:     toAddr,
		Amt:        amt,
	})
	data, err := Request(apiAddr, payNetQueryOffer, data)
	if err != nil {
		return nil, err
	}
	offerStrs := strings.Split(string(data), ";")
	offers := make([]payoffer.PayOffer, 0)
	for _, offerStr := range offerStrs {
		if offerStr == "" {
			continue
		}
		offerBytes, err := hex.DecodeString(offerStr)
		if err != nil {
			return nil, err
		}
		offer, err := payoffer.FromBytes(offerBytes)
		if err != nil {
			return nil, err
		}
		offers = append(offers, *offer)
	}
	return offers, nil
}

// payNetQueryOutboundPaychRequestJson is used for serialization.
type payNetQueryOutboundPaychRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	ToAddr     string `json:"to_addr"`
	PAddr      string `json:"pi_addr"`
}

// RequestPayNetOutboundQueryPaych requests a paynet outbound query paych.
// It takes a context, the api addr, currency id, recipient address, peer addr info as arguments.
// It returns a paych offer and error.
func RequestPayNetOutboundQueryPaych(ctx context.Context, apiAddr string, currencyID uint64, toAddr string, pi peer.AddrInfo) (*paychoffer.PaychOffer, error) {
	piData, err := json.Marshal(pi)
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(payNetQueryOutboundPaychRequestJson{
		CurrencyID: currencyID,
		ToAddr:     toAddr,
		PAddr:      hex.EncodeToString(piData),
	})
	data, err = Request(apiAddr, payNetOutboundQueryPaych, data)
	if err != nil {
		return nil, err
	}
	return paychoffer.FromBytes(data)
}

// payNetOutboundCreateJson is used for serialization.
type payNetOutboundCreateRequestJson struct {
	Offer string `json:"offer"`
	Amt   string `json:"amt"`
	PAddr string `json:"pi_addr"`
}

// RequestPayNetOutboundCreate requests a paynet outbound create paych.
// It takes a context, the api addr, a paych offer, amount, and peer addr info as arguments.
// It returns the error.
func RequestPayNetOutboundCreate(ctx context.Context, apiAddr string, offer *paychoffer.PaychOffer, amt string, pi peer.AddrInfo) error {
	piData, err := json.Marshal(pi)
	if err != nil {
		return err
	}
	data, _ := json.Marshal(payNetOutboundCreateRequestJson{
		Offer: hex.EncodeToString(offer.ToBytes()),
		Amt:   amt,
		PAddr: hex.EncodeToString(piData),
	})
	data, err = Request(apiAddr, payNetOutboundCreate, data)
	if err != nil {
		return err
	}
	if data[0] != 0 {
		return fmt.Errorf("fail to create payment channel")
	}
	return nil
}

// payNetOutboundTopupRequestJson is used for serialization.
type payNetOutboundTopupRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	ToAddr     string `json:"to_addr"`
	Amt        string `json:"amt"`
}

// RequestPayNetOutboundTopup requests a paynet outbound topup.
// It takes a context, the api addr, a currency id, a recipient address, amount as arguments.
// It returns the error.
func RequestPayNetOutboundTopup(ctx context.Context, apiAddr string, currencyID uint64, toAddr string, amt string) error {
	data, _ := json.Marshal(payNetOutboundTopupRequestJson{
		CurrencyID: currencyID,
		ToAddr:     toAddr,
		Amt:        amt,
	})
	data, err := Request(apiAddr, payNetOutboundTopup, data)
	if err != nil {
		return err
	}
	if data[0] != 0 {
		return fmt.Errorf("fail to topup payment channel")
	}
	return nil
}

// RequestPayNetOutboundList requests a paynet outbound list.
// It takes a context, the api addr as arguments.
// It returns a map from currency id to channel addrs and the error.
func RequestPayNetOutboundList(ctx context.Context, apiAddr string) (map[uint64][]string, error) {
	data, err := Request(apiAddr, payNetOutboundList, []byte{0})
	if err != nil {
		return nil, err
	}
	var res map[uint64][]string
	err = json.Unmarshal(data, &res)
	return res, err
}

// payNetOutboundInspectRequestJson is used for serialization.
type payNetOutboundInspectRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	ChAddr     string `json:"ch_addr"`
}

// RequestPayNetOutboundInspect requests a paynet outbound inspect.
// It takes a context, the api addr, a currency id, a channel address as arguments.
// It returns recipient address, redeemed amount, total amount, settlement time of this channel (as specified when creating this channel),
// boolean indicates if channel is active, current chain height, settling height and error.
func RequestPayNetOutboundInspect(ctx context.Context, apiAddr string, currencyID uint64, chAddr string) (string, *big.Int, *big.Int, int64, bool, int64, int64, error) {
	data, _ := json.Marshal(payNetOutboundInspectRequestJson{
		CurrencyID: currencyID,
		ChAddr:     chAddr,
	})
	data, err := Request(apiAddr, payNetOutboundInspect, data)
	if err != nil {
		return "", nil, nil, 0, false, 0, 0, err
	}
	resp := payNetOutboundInspectResponseJson{}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return "", nil, nil, 0, false, 0, 0, err
	}
	redeemed, ok := big.NewInt(0).SetString(resp.Redeemed, 10)
	if !ok {
		return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error parsing redeemed")
	}
	balance, ok := big.NewInt(0).SetString(resp.Balance, 10)
	if !ok {
		return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error parsing balance")
	}
	return resp.ToAddr, redeemed, balance, resp.Settlement, resp.Active, resp.CurHeight, resp.SettlingAt, nil
}

// payNetOutboundSettleRequestJson is used for serialization.
type payNetOutboundSettleRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	ChAddr     string `json:"ch_addr"`
}

// RequestPayNetOutboundSettle requests a paynet outbound settle.
// It takes a context, the api addr, a currency id, a channel address as arguments.
// It returns the error.
func RequestPayNetOutboundSettle(ctx context.Context, apiAddr string, currencyID uint64, chAddr string) error {
	data, _ := json.Marshal(payNetOutboundSettleRequestJson{
		CurrencyID: currencyID,
		ChAddr:     chAddr,
	})
	data, err := Request(apiAddr, payNetOutboundSettle, data)
	if err != nil {
		return err
	}
	if data[0] != 0 {
		return fmt.Errorf("fail to settle payment channel")
	}
	return nil
}

// payNetOutboundCollectRequestJson is used for serialization.
type payNetOutboundCollectRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	ChAddr     string `json:"ch_addr"`
}

// RequestPayNetOutboundCollect requests a paynet outbound collect.
// It takes a context, the api addr, a currency id, a channel address as arguments.
// It returns the error.
func RequestPayNetOutboundCollect(ctx context.Context, apiAddr string, currencyID uint64, chAddr string) error {
	data, _ := json.Marshal(payNetOutboundCollectRequestJson{
		CurrencyID: currencyID,
		ChAddr:     chAddr,
	})
	data, err := Request(apiAddr, payNetOutboundCollect, data)
	if err != nil {
		return err
	}
	if data[0] != 0 {
		return fmt.Errorf("fail to collect payment channel")
	}
	return nil
}

// RequestPayNetInboundList requests a paynet inbound list.
// It takes a context, the api addr as arguments.
// It returns a map from currency id to channel addrs and the error.
func RequestPayNetInboundList(ctx context.Context, apiAddr string) (map[uint64][]string, error) {
	data, err := Request(apiAddr, payNetInboundList, []byte{0})
	if err != nil {
		return nil, err
	}
	var res map[uint64][]string
	err = json.Unmarshal(data, &res)
	return res, err
}

// payNetInboundInspectRequestJson is used for serialization.
type payNetInboundInspectRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	ChAddr     string `json:"ch_addr"`
}

// RequestPayNetInboundInspect requests a paynet inbound inspect.
// It takes a context, the api addr, a currency id, a channel address as arguments.
// It returns recipient address, redeemed amount, total amount, settlement time of this channel (as specified when creating this channel),
// boolean indicates if channel is active, current chain height, settling height and error.
func RequestPayNetInboundInspect(ctx context.Context, apiAddr string, currencyID uint64, chAddr string) (string, *big.Int, *big.Int, int64, bool, int64, int64, error) {
	data, _ := json.Marshal(payNetInboundInspectRequestJson{
		CurrencyID: currencyID,
		ChAddr:     chAddr,
	})
	data, err := Request(apiAddr, payNetInboundInspect, data)
	if err != nil {
		return "", nil, nil, 0, false, 0, 0, err
	}
	resp := payNetInboundInspectResponseJson{}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return "", nil, nil, 0, false, 0, 0, err
	}
	redeemed, ok := big.NewInt(0).SetString(resp.Redeemed, 10)
	if !ok {
		return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error parsing redeemed")
	}
	balance, ok := big.NewInt(0).SetString(resp.Balance, 10)
	if !ok {
		return "", nil, nil, 0, false, 0, 0, fmt.Errorf("error parsing balance")
	}
	return resp.ToAddr, redeemed, balance, resp.Settlement, resp.Active, resp.CurHeight, resp.SettlingAt, nil
}

// payNetInboundSettleRequestJson is used for serialization.
type payNetInboundSettleRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	ChAddr     string `json:"ch_addr"`
}

// RequestPayNetInboundSettle requests a paynet inbound settle.
// It takes a context, the api addr, a currency id, a channel address as arguments.
// It returns the error.
func RequestPayNetInboundSettle(ctx context.Context, apiAddr string, currencyID uint64, chAddr string) error {
	data, _ := json.Marshal(payNetInboundSettleRequestJson{
		CurrencyID: currencyID,
		ChAddr:     chAddr,
	})
	data, err := Request(apiAddr, payNetInboundSettle, data)
	if err != nil {
		return err
	}
	if data[0] != 0 {
		return fmt.Errorf("fail to settle payment channel")
	}
	return nil
}

// payNetInboundCollectRequestJson is used for serialization.
type payNetInboundCollectRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	ChAddr     string `json:"ch_addr"`
}

// RequestPayNetInboundCollect requests a paynet inbound collect.
// It takes a context, the api addr, a currency id, a channel address as arguments.
// It returns the error.
func RequestPayNetInboundCollect(ctx context.Context, apiAddr string, currencyID uint64, chAddr string) error {
	data, _ := json.Marshal(payNetInboundCollectRequestJson{
		CurrencyID: currencyID,
		ChAddr:     chAddr,
	})
	data, err := Request(apiAddr, payNetInboundCollect, data)
	if err != nil {
		return err
	}
	if data[0] != 0 {
		return fmt.Errorf("fail to collect payment channel")
	}
	return nil
}

// payNetServingServeRequestJson is used for serialization.
type payNetServingServeRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	ToAddr     string `json:"to_addr"`
	PPP        string `json:"ppp"`
	Period     string `json:"period"`
}

// RequestPayNetServing requests a paynet serving serve.
// It takes a context, the api addr, a currency id, a recipient address, a ppp (price per period), a period as arguments.
// It returns the error.
func RequestPayNetServingServe(ctx context.Context, apiAddr string, currencyID uint64, toAddr string, ppp string, period string) error {
	data, _ := json.Marshal(payNetServingServeRequestJson{
		CurrencyID: currencyID,
		ToAddr:     toAddr,
		PPP:        ppp,
		Period:     period,
	})
	data, err := Request(apiAddr, payNetServingServe, data)
	if err != nil {
		return err
	}
	if data[0] != 0 {
		return fmt.Errorf("error serving")
	}
	return nil
}

// RequestPayNetServingList requests a paynet serving list.
// It takes a context, the api addr as arguments.
// It returns a map from currency id to servings and the error.
func RequestPayNetServingList(ctx context.Context, apiAddr string) (map[uint64][]string, error) {
	data, err := Request(apiAddr, payNetServingList, []byte{0})
	if err != nil {
		return nil, err
	}
	var res map[uint64][]string
	err = json.Unmarshal(data, &res)
	return res, err
}

// payNetServingInspectRequestJson is used for serialization.
type payNetServingInspectRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	ToAddr     string `json:"to_addr"`
}

// RequestPayNetServingInspect requests a paynet serving inspect.
// It takes a context, the api addr, a currency id, a recipient address as arguments.
// It returns the ppp (price per period), period, the expiration and error.
func RequestPayNetServingInspect(ctx context.Context, apiAddr string, currencyID uint64, toAddr string) (*big.Int, *big.Int, int64, error) {
	data, _ := json.Marshal(payNetServingInspectRequestJson{
		CurrencyID: currencyID,
		ToAddr:     toAddr,
	})
	data, err := Request(apiAddr, payNetServingInspect, data)
	if err != nil {
		return nil, nil, 0, err
	}
	resp := payNetServingInspectResponseJson{}
	err = json.Unmarshal(data, &resp)
	ppp, _ := big.NewInt(0).SetString(resp.PPP, 10)
	period, _ := big.NewInt(0).SetString(resp.Period, 10)
	return ppp, period, resp.Expiration, err
}

// payNetServingRetireRequestJson is used for serialization.
type payNetServingRetireRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	ToAddr     string `json:"to_addr"`
}

// RequestPayNetServingRetire requests a paynet serving retire.
// It takes a context, the api addr, a currency id, a recipient address as arguments.
// It returns the error.
func RequestPayNetServingRetire(ctx context.Context, apiAddr string, currencyID uint64, toAddr string) error {
	data, _ := json.Marshal(payNetServingInspectRequestJson{
		CurrencyID: currencyID,
		ToAddr:     toAddr,
	})
	data, err := Request(apiAddr, payNetServingInspect, data)
	if err != nil {
		return err
	}
	if data[0] != 0 {
		return fmt.Errorf("error retiring")
	}
	return nil
}

// RequestPayNetServingForcePublish requests a paynet serving force publish.
// It takes a context, the api addr as arguments.
// It returns the error.
func RequestPayNetServingForcePublish(ctx context.Context, apiAddr string) error {
	data, err := Request(apiAddr, payNetServingForcePublish, []byte{0})
	if err != nil {
		return err
	}
	if data[0] != 0 {
		return fmt.Errorf("error force publish")
	}
	return nil
}

// RequestPayNetPeerList requests a paynet peer list.
// It takes a context, the api addr as arguments.
// It returns a map from currency id to peer addrs and the error.
func RequestPayNetPeerList(ctx context.Context, apiAddr string) (map[uint64][]string, error) {
	data, err := Request(apiAddr, payNetPeerList, []byte{0})
	if err != nil {
		return nil, err
	}
	var res map[uint64][]string
	err = json.Unmarshal(data, &res)
	return res, err
}

// payNetPeerInspectRequestJson is used for serialization.
type payNetPeerInspectRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	ToAddr     string `json:"to_addr"`
}

// RequestPayNetPeerInspect requests a paynet peer inspect.
// It takes a context, the api addr, currency id, recipient address as arguments.
// It returns peer id, a boolean indicating if peer is blocked, success count, failed count and error.
func RequestPayNetPeerInspect(ctx context.Context, apiAddr string, currencyID uint64, toAddr string) (string, bool, int, int, error) {
	data, _ := json.Marshal(payNetPeerInspectRequestJson{
		CurrencyID: currencyID,
		ToAddr:     toAddr,
	})
	data, err := Request(apiAddr, payNetPeerInspect, data)
	if err != nil {
		return "", false, 0, 0, err
	}
	resp := payNetPeerInspectResponseJson{}
	err = json.Unmarshal(data, &resp)
	return resp.PeerID, resp.Blocked, resp.Success, resp.Fail, err
}

// payNetPeerBlockRequestJson is used for serialization.
type payNetPeerBlockRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	ToAddr     string `json:"to_addr"`
}

// RequestPayNetPeerBlock requests a paynet peer block.
// It takes a context, the api addr, currency id, recipient address as arguments.
// It returns the error.
func RequestPayNetPeerBlock(ctx context.Context, apiAddr string, currencyID uint64, toAddr string) error {
	data, _ := json.Marshal(payNetPeerBlockRequestJson{
		CurrencyID: currencyID,
		ToAddr:     toAddr,
	})
	_, err := Request(apiAddr, payNetPeerBlock, data)
	return err
}

// payNetPeerUnblockRequestJson is used for serialization.
type payNetPeerUnblockRequestJson struct {
	CurrencyID uint64 `json:"currency_id"`
	ToAddr     string `json:"to_addr"`
}

// RequestPayNetPeerUnblock requests a paynet peer unblock.
// It takes a context, the api addr, currency id, recipient address as arguments.
// It returns the error.
func RequestPayNetPeerUnblock(ctx context.Context, apiAddr string, currencyID uint64, toAddr string) error {
	data, _ := json.Marshal(payNetPeerUnblockRequestJson{
		CurrencyID: currencyID,
		ToAddr:     toAddr,
	})
	_, err := Request(apiAddr, payNetPeerUnblock, data)
	return err
}

// RequestSystemAddr requests a system addr.
// It takes a context, the api addr as arguments.
// It returns a list of p2p addrs and error.
func RequestSystemAddr(ctx context.Context, apiAddr string) ([]string, error) {
	data, err := Request(apiAddr, systemAddr, []byte{0})
	if err != nil {
		return nil, err
	}
	return strings.Split(string(data), ";"), nil
}

// RequestSystemBootstrap requests a system bootstrap.
// It takes a context, the api addr, peer addr info as arguments.
// It returns the error.
func RequestSystemBootstrap(ctx context.Context, apiAddr string, pi peer.AddrInfo) error {
	data, err := pi.MarshalJSON()
	if err != nil {
		return err
	}
	data, err = Request(apiAddr, systemBootstrap, data)
	if err != nil {
		return err
	}
	if data[0] != 0 {
		return fmt.Errorf("error bootstrap")
	}
	return nil
}

// RequestSystemCacheSize requests a system cache size.
// It takes a context, the api addr as arguments.
// It returns the cache size and error.
func RequestSystemCacheSize(ctx context.Context, apiAddr string) (uint64, error) {
	data, err := Request(apiAddr, systemCacheSize, []byte{0})
	if err != nil {
		return 0, err
	}
	resp := systemCacheSizeResponseJson{}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return 0, err
	}
	return resp.Size, nil
}

// RequestSystemCachePrune requests a system cache prune.
// It takes a context, the api addr as arguments.
// It returns the error.
func RequestSystemCachePrune(ctx context.Context, apiAddr string) error {
	data, err := Request(apiAddr, systemCachePrune, []byte{0})
	if err != nil {
		return err
	}
	if data[0] != 0 {
		return fmt.Errorf("error force publish")
	}
	return nil
}

// retrievalCacheRequestJson is used for serialization.
type retrievalCacheRequestJson struct {
	CID     string `json:"cid"`
	OutPath string `json:"out_path"`
}

// RequestRetrievalCache requests a retrieval cache.
// It takes a context, the api addr, cid and outpath as arguments.
// It returns boolean indicating if found in cache and error.
func RequestRetrievalCache(ctx context.Context, apiAddr string, id string, outPath string) (bool, error) {
	data, _ := json.Marshal(retrievalCacheRequestJson{
		CID:     id,
		OutPath: outPath,
	})
	data, err := Request(apiAddr, retrievalCacheAPI, data)
	if err != nil {
		return false, err
	}
	if data[0] == 1 {
		return true, nil
	} else {
		return false, nil
	}
}

// retrievalRequestJson is used for serialization.
type retrievalRequestJson struct {
	CIDOffer string `json:"cid_offer"`
	PayOffer string `json:"pay_offer"`
	OutPath  string `json:"out_path"`
}

// RequestRetrieval requests a retrieval.
// It takes a context, the api addr, cid offer, pay offer (can be nil) and outpath as arguments.
// It returns the error.
func RequestRetrieval(ctx context.Context, apiAddr string, cidOffer *cidoffer.CIDOffer, payOffer *payoffer.PayOffer, outPath string) error {
	payOfferStr := ""
	if payOffer != nil {
		payOfferStr = hex.EncodeToString(payOffer.ToBytes())
	}
	data, _ := json.Marshal(retrievalRequestJson{
		CIDOffer: hex.EncodeToString(cidOffer.ToBytes()),
		PayOffer: payOfferStr,
		OutPath:  outPath,
	})
	data, err := Request(apiAddr, retrievalAPI, data)
	if err != nil {
		return err
	}
	if data[0] != 0 {
		return fmt.Errorf("error retrieval")
	}
	return nil
}
