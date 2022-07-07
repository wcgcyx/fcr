package trans

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
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/filecoin-project/go-address"
	lotuscrypto2 "github.com/filecoin-project/go-crypto"
	"github.com/filecoin-project/go-jsonrpc"
	lotusbig "github.com/filecoin-project/go-state-types/big"
	paychtypes "github.com/filecoin-project/go-state-types/builtin/v8/paych"
	lotuscrypto "github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/actors/builtin/paych"
	"github.com/filecoin-project/lotus/chain/types"
	init8 "github.com/filecoin-project/specs-actors/v8/actors/builtin/init"
	paych8 "github.com/filecoin-project/specs-actors/v8/actors/builtin/paych"
	"github.com/minio/blake2b-simd"
	"github.com/wcgcyx/fcr/crypto"
)

// filAdapter is the adapter to access payment channel on Filecoin network.
type filAdapter struct {
	signer     crypto.Signer
	lotusAP    string
	authToken  string
	confidence uint64
}

// newFilAdapter creates a new filAdapter
//
// @input - context, signer, lotus ap, auth token.
//
// @output - filAdapter, error.
func newFilAdapter(ctx context.Context, signer crypto.Signer, lotusAP string, authToken string, confidence uint64) (*filAdapter, error) {
	adapter := &filAdapter{
		signer:     signer,
		lotusAP:    lotusAP,
		authToken:  authToken,
		confidence: confidence,
	}
	apis, closer, err := adapter.getLotusAPI(ctx)
	if err != nil {
		return nil, err
	}
	if closer != nil {
		defer closer()
	}
	// Do a test connection.
	_, err = apis.ChainHead(ctx)
	if err != nil {
		return nil, err
	}
	return adapter, nil
}

// Create is used to create a payment chanel with given recipient and amount.
//
// @input - context, recipient address, amount.
//
// @output - channel address, error.
func (a *filAdapter) Create(ctx context.Context, toAddr string, amt *big.Int) (string, error) {
	_, fromAddr, err := a.signer.GetAddr(ctx, crypto.FIL)
	if err != nil {
		log.Warnf("Fail to get addr for %v: %v", crypto.FIL, err.Error())
		return "", err
	}
	from, err := address.NewFromString(fromAddr)
	if err != nil {
		log.Errorf("Fail to get address from %v: %v", fromAddr, err.Error())
		return "", err
	}
	to, err := address.NewFromString(toAddr)
	if err != nil {
		log.Debugf("Fail to get address from %v: %v", toAddr, err.Error())
		return "", err
	}
	if toAddr[0] != 'f' {
		log.Debugf("Recipient address %v does not start with f", toAddr)
		return "", fmt.Errorf("recipient address %v does not start with f", toAddr)
	}
	// Get lotus API
	api, closer, err := a.getLotusAPI(ctx)
	if err != nil {
		log.Warnf("Fail to get lotus API: %v", err.Error())
		return "", err
	}
	if closer != nil {
		defer closer()
	}
	// Build message
	builder := paych.Message(actors.Version8, from)
	msg, err := builder.Create(to, lotusbig.NewFromGo(amt))
	if err != nil {
		log.Errorf("Fail to build message: %v", err.Error())
		return "", err
	}
	signedMsg, err := a.getSignedMessage(ctx, api, msg)
	if err != nil {
		log.Warnf("Fail to get signed message : %v", err.Error())
		return "", err
	}
	// Push message
	txHash, err := api.MpoolPush(ctx, signedMsg)
	if err != nil {
		log.Warnf("Fail to push message: %v", err.Error())
		return "", err
	}
	receipt, err := api.StateWaitMsg(ctx, txHash, a.confidence)
	if err != nil {
		log.Warnf("Fail to wait for %v: %v", txHash.String(), err.Error())
		return "", fmt.Errorf("fail to wait for %v: %v", txHash.String(), err.Error())
	}
	if receipt.Receipt.ExitCode != 0 {
		log.Warnf("Fail to execute %v: %v", txHash.String(), receipt.Receipt.ExitCode.Error())
		return "", fmt.Errorf("fail to execute %v: %v", txHash.String(), receipt.Receipt.ExitCode.Error())
	}
	var decodedReturn init8.ExecReturn
	err = decodedReturn.UnmarshalCBOR(bytes.NewReader(receipt.Receipt.Return))
	if err != nil {
		log.Errorf("Fail to decode return from %v: %v", txHash.String(), err.Error())
		return "", fmt.Errorf("fail to decode return from %v: %v", txHash.String(), err.Error())
	}
	// Force address to start with "f"
	return "f" + decodedReturn.RobustAddress.String()[1:], nil
}

// Topup is used to topup a given payment channel with given amount.
//
// @input - context, channel address, amount.
//
// @output - error.
func (a *filAdapter) Topup(ctx context.Context, chAddr string, amt *big.Int) error {
	_, fromAddr, err := a.signer.GetAddr(ctx, crypto.FIL)
	if err != nil {
		log.Warnf("Fail to get addr for %v: %v", crypto.FIL, err.Error())
		return err
	}
	from, err := address.NewFromString(fromAddr)
	if err != nil {
		log.Errorf("Fail to get address from %v: %v", fromAddr, err.Error())
		return err
	}
	ch, err := address.NewFromString(chAddr)
	if err != nil {
		log.Debugf("Fail to get address from %v: %v", chAddr, err.Error())
		return err
	}
	if chAddr[0] != 'f' {
		log.Debugf("Channel address %v does not start with f", chAddr)
		return fmt.Errorf("Channel address %v does not start with f", chAddr)
	}
	// Get lotus API
	api, closer, err := a.getLotusAPI(ctx)
	if err != nil {
		log.Warnf("Fail to get lotus API: %v", err.Error())
		return err
	}
	if closer != nil {
		defer closer()
	}
	// Build message
	msg := &types.Message{
		To:     ch,
		From:   from,
		Value:  lotusbig.NewFromGo(amt),
		Method: 0,
	}
	signedMsg, err := a.getSignedMessage(ctx, api, msg)
	if err != nil {
		log.Warnf("Fail to get signed message : %v", err.Error())
		return err
	}
	// Push message
	txHash, err := api.MpoolPush(ctx, signedMsg)
	if err != nil {
		log.Warnf("Fail to push message: %v", err.Error())
		return err
	}
	receipt, err := api.StateWaitMsg(ctx, txHash, a.confidence)
	if err != nil {
		log.Warnf("Fail to wait for %v: %v", txHash.String(), err.Error())
		return fmt.Errorf("fail to wait for %v: %v", txHash.String(), err.Error())
	}
	if receipt.Receipt.ExitCode != 0 {
		log.Warnf("Fail to execute %v: %v", txHash.String(), receipt.Receipt.ExitCode.Error())
		return fmt.Errorf("fail to execute %v: %v", txHash.String(), receipt.Receipt.ExitCode.Error())
	}
	return nil
}

// Check is used to check the status of a given payment channel.
//
// @input - context, channel address.
//
// @output - settling height, min settling height, current height, channel redeemed, channel balance, sender address, recipient address and error.
func (a *filAdapter) Check(ctx context.Context, chAddr string) (int64, int64, int64, *big.Int, *big.Int, string, string, error) {
	ch, err := address.NewFromString(chAddr)
	if err != nil {
		log.Debugf("Fail to get address from %v: %v", chAddr, err.Error())
		return 0, 0, 0, nil, nil, "", "", err
	}
	if chAddr[0] != 'f' {
		log.Debugf("Channel address %v does not start with f", chAddr)
		return 0, 0, 0, nil, nil, "", "", fmt.Errorf("channel address %v does not start with f", chAddr)
	}
	// Get lotus API
	api, closer, err := a.getLotusAPI(ctx)
	if err != nil {
		log.Warnf("Fail to get lotus API: %v", err.Error())
		return 0, 0, 0, nil, nil, "", "", err
	}
	if closer != nil {
		defer closer()
	}
	// Get actor state
	actor, err := api.StateGetActor(ctx, ch, types.EmptyTSK)
	if err != nil {
		log.Warnf("Fail to get actor state for %v: %v", chAddr, err.Error())
		return 0, 0, 0, nil, nil, "", "", err
	}
	data, err := api.ChainReadObj(ctx, actor.Head)
	if err != nil {
		log.Warnf("Fail to read chain object for %v: %v", actor.Head, err.Error())
		return 0, 0, 0, nil, nil, "", "", err
	}
	state := paych8.State{}
	err = state.UnmarshalCBOR(bytes.NewReader(data))
	if err != nil {
		log.Errorf("Fail to decode data %v into paych state: %v", data, err.Error())
		return 0, 0, 0, nil, nil, "", "", err
	}
	recipient, err := api.StateAccountKey(ctx, state.To, types.EmptyTSK)
	if err != nil {
		log.Warnf("Fail to read recipient %v: %v", state.To, err.Error())
		return 0, 0, 0, nil, nil, "", "", err
	}
	sender, err := api.StateAccountKey(ctx, state.From, types.EmptyTSK)
	if err != nil {
		log.Warnf("Fail to read sender %v: %v", state.From, err.Error())
		return 0, 0, 0, nil, nil, "", "", err
	}
	head, err := api.ChainHead(ctx)
	if err != nil {
		log.Warnf("Fail to get chain head: %v", err.Error())
		return 0, 0, 0, nil, nil, "", "", err
	}
	return int64(state.SettlingAt), int64(state.MinSettleHeight), int64(head.Height()), state.ToSend.Int, actor.Balance.Int, "f" + sender.String()[1:], "f" + recipient.String()[1:], nil
}

// Update is used to update channel state with a voucher.
//
// @input - context, channel address, voucher.
//
// @output - error.
func (a *filAdapter) Update(ctx context.Context, chAddr string, voucher string) error {
	_, fromAddr, err := a.signer.GetAddr(ctx, crypto.FIL)
	if err != nil {
		log.Warnf("Fail to get addr for %v: %v", crypto.FIL, err.Error())
		return err
	}
	from, err := address.NewFromString(fromAddr)
	if err != nil {
		log.Errorf("Fail to get address from %v: %v", fromAddr, err.Error())
		return err
	}
	ch, err := address.NewFromString(chAddr)
	if err != nil {
		log.Debugf("Fail to get address from %v: %v", chAddr, err.Error())
		return err
	}
	if chAddr[0] != 'f' {
		log.Debugf("Channel address %v does not start with f", chAddr)
		return fmt.Errorf("recipient address %v does not start with f", chAddr)
	}
	sv, err := paych.DecodeSignedVoucher(voucher)
	if err != nil {
		log.Errorf("Fail to decode signed voucher %v: %v", voucher, err.Error())
		return err
	}
	// Get lotus API
	api, closer, err := a.getLotusAPI(ctx)
	if err != nil {
		log.Warnf("Fail to get lotus API: %v", err.Error())
		return err
	}
	if closer != nil {
		defer closer()
	}
	// Build message
	builder := paych.Message(actors.Version8, from)
	msg, err := builder.Update(ch, sv, nil)
	if err != nil {
		log.Errorf("Fail to build message: %v", err.Error())
		return err
	}
	signedMsg, err := a.getSignedMessage(ctx, api, msg)
	if err != nil {
		log.Warnf("Fail to get signed message : %v", err.Error())
		return err
	}
	// Push message
	txHash, err := api.MpoolPush(ctx, signedMsg)
	if err != nil {
		log.Warnf("Fail to push message: %v", err.Error())
		return err
	}
	receipt, err := api.StateWaitMsg(ctx, txHash, a.confidence)
	if err != nil {
		log.Warnf("Fail to wait for %v: %v", txHash.String(), err.Error())
		return fmt.Errorf("fail to wait for %v: %v", txHash.String(), err.Error())
	}
	if receipt.Receipt.ExitCode != 0 {
		log.Warnf("Fail to execute %v: %v", txHash.String(), receipt.Receipt.ExitCode.Error())
		return fmt.Errorf("fail to execute %v: %v", txHash.String(), receipt.Receipt.ExitCode.Error())
	}
	return nil
}

// Settle is used to settle a payment channel.
//
// @input - context, channel address.
//
// @output - error.
func (a *filAdapter) Settle(ctx context.Context, chAddr string) error {
	_, fromAddr, err := a.signer.GetAddr(ctx, crypto.FIL)
	if err != nil {
		log.Warnf("Fail to get addr for %v: %v", crypto.FIL, err.Error())
		return err
	}
	from, err := address.NewFromString(fromAddr)
	if err != nil {
		log.Errorf("Fail to get address from %v: %v", fromAddr, err.Error())
		return err
	}
	ch, err := address.NewFromString(chAddr)
	if err != nil {
		log.Debugf("Fail to get address from %v: %v", chAddr, err.Error())
		return err
	}
	if chAddr[0] != 'f' {
		log.Debugf("Channel address %v does not start with f", chAddr)
		return fmt.Errorf("recipient address %v does not start with f", chAddr)
	}
	// Get lotus API
	api, closer, err := a.getLotusAPI(ctx)
	if err != nil {
		log.Warnf("Fail to get lotus API: %v", err.Error())
		return err
	}
	if closer != nil {
		defer closer()
	}
	// Build message
	builder := paych.Message(actors.Version8, from)
	msg, err := builder.Settle(ch)
	if err != nil {
		log.Errorf("Fail to build message: %v", err.Error())
		return err
	}
	signedMsg, err := a.getSignedMessage(ctx, api, msg)
	if err != nil {
		log.Warnf("Fail to get signed message : %v", err.Error())
		return err
	}
	// Push message
	txHash, err := api.MpoolPush(ctx, signedMsg)
	if err != nil {
		log.Warnf("Fail to push message: %v", err.Error())
		return err
	}
	receipt, err := api.StateWaitMsg(ctx, txHash, a.confidence)
	if err != nil {
		log.Warnf("Fail to wait for %v: %v", txHash.String(), err.Error())
		return fmt.Errorf("fail to wait for %v: %v", txHash.String(), err.Error())
	}
	if receipt.Receipt.ExitCode != 0 {
		log.Warnf("Fail to execute %v: %v", txHash.String(), receipt.Receipt.ExitCode.Error())
		return fmt.Errorf("fail to execute %v: %v", txHash.String(), receipt.Receipt.ExitCode.Error())
	}
	return nil
}

// Collect is used to collect a payment channel.
//
// @input - context, channel address.
//
// @output - error.
func (a *filAdapter) Collect(ctx context.Context, chAddr string) error {
	_, fromAddr, err := a.signer.GetAddr(ctx, crypto.FIL)
	if err != nil {
		log.Warnf("Fail to get addr for %v: %v", crypto.FIL, err.Error())
		return err
	}
	from, err := address.NewFromString(fromAddr)
	if err != nil {
		log.Errorf("Fail to get address from %v: %v", fromAddr, err.Error())
		return err
	}
	ch, err := address.NewFromString(chAddr)
	if err != nil {
		log.Debugf("Fail to get address from %v: %v", chAddr, err.Error())
		return err
	}
	if chAddr[0] != 'f' {
		log.Debugf("Channel address %v does not start with f", chAddr)
		return fmt.Errorf("recipient address %v does not start with f", chAddr)
	}
	// Get lotus API
	api, closer, err := a.getLotusAPI(ctx)
	if err != nil {
		log.Warnf("Fail to get lotus API: %v", err.Error())
		return err
	}
	if closer != nil {
		defer closer()
	}
	// Build message
	builder := paych.Message(actors.Version8, from)
	msg, err := builder.Collect(ch)
	if err != nil {
		log.Errorf("Fail to build message: %v", err.Error())
		return err
	}
	signedMsg, err := a.getSignedMessage(ctx, api, msg)
	if err != nil {
		log.Warnf("Fail to get signed message : %v", err.Error())
		return err
	}
	// Push message
	txHash, err := api.MpoolPush(ctx, signedMsg)
	if err != nil {
		log.Warnf("Fail to push message: %v", err.Error())
		return err
	}
	receipt, err := api.StateWaitMsg(ctx, txHash, a.confidence)
	if err != nil {
		log.Warnf("Fail to wait for %v: %v", txHash.String(), err.Error())
		return fmt.Errorf("fail to wait for %v: %v", txHash.String(), err.Error())
	}
	if receipt.Receipt.ExitCode != 0 {
		log.Warnf("Fail to execute %v: %v", txHash.String(), receipt.Receipt.ExitCode.Error())
		return fmt.Errorf("fail to execute %v: %v", txHash.String(), receipt.Receipt.ExitCode.Error())
	}
	return nil
}

// GetHeight is used to get the current height of the chain.
//
// @input - context.
//
// @output - height, error.
func (a *filAdapter) GetHeight(ctx context.Context) (int64, error) {
	api, closer, err := a.getLotusAPI(ctx)
	if err != nil {
		return 0, err
	}
	if closer != nil {
		defer closer()
	}
	head, err := api.ChainHead(ctx)
	if err != nil {
		return 0, err
	}
	return int64(head.Height()), nil
}

// GetBalance is used to get the balance of a given address.
//
// @input - context, address.
//
// @output - balance, error.
func (a *filAdapter) GetBalance(ctx context.Context, addr string) (*big.Int, error) {
	to, err := address.NewFromString(addr)
	if err != nil {
		log.Debugf("Fail to get address from %v: %v", addr, err.Error())
		return nil, err
	}
	if addr[0] != 'f' {
		log.Debugf("Address %v does not start with f", addr)
		return nil, fmt.Errorf("address %v does not start with f", addr)
	}
	// Get lotus API
	api, closer, err := a.getLotusAPI(ctx)
	if err != nil {
		log.Warnf("Fail to get lotus API: %v", err.Error())
		return nil, err
	}
	if closer != nil {
		defer closer()
	}
	actor, err := api.StateGetActor(ctx, to, types.EmptyTSK)
	if err != nil {
		if strings.Contains(err.Error(), "actor not found") {
			return big.NewInt(0), nil
		}
		log.Warnf("Fail to get actor state for %v: %v", addr, err.Error())
		return nil, err
	}
	return actor.Balance.Int, nil
}

// GenerateVoucher is used to generate a voucher.
//
// @input - context, channel address, lane number, nonce, redeemed amount.
//
// @output - voucher, error.
func (a *filAdapter) GenerateVoucher(ctx context.Context, chAddr string, lane uint64, nonce uint64, redeemed *big.Int) (string, error) {
	ch, err := address.NewFromString(chAddr)
	if err != nil {
		return "", err
	}
	sv := &paychtypes.SignedVoucher{
		ChannelAddr: ch,
		Lane:        lane,
		Nonce:       nonce,
		Amount:      lotusbig.NewFromGo(redeemed),
	}
	vb, err := sv.SigningBytes()
	if err != nil {
		return "", err
	}
	sigType, sig, err := a.signer.Sign(ctx, crypto.FIL, vb)
	if err != nil {
		return "", err
	}
	if sigType != crypto.SECP256K1 {
		return "", fmt.Errorf("Unsupported key type")
	}
	sv.Signature = &lotuscrypto.Signature{
		Type: lotuscrypto.SigTypeSecp256k1,
		Data: sig,
	}
	buf := new(bytes.Buffer)
	if err := sv.MarshalCBOR(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf.Bytes()), nil
}

// VerifyVoucher is used to decode a given voucher.
//
// @input - voucher.
//
// @output - sender address, channel address, lane number, nonce, redeemed, error.
func (a *filAdapter) VerifyVoucher(voucher string) (string, string, uint64, uint64, *big.Int, error) {
	sv, err := paych.DecodeSignedVoucher(voucher)
	if err != nil {
		return "", "", 0, 0, nil, err
	}
	if int64(sv.TimeLockMin) != 0 {
		return "", "", 0, 0, nil, fmt.Errorf("non emptry time lock min not supported")
	}
	if int64(sv.TimeLockMin) != 0 {
		return "", "", 0, 0, nil, fmt.Errorf("non emptry time lock max not supported")
	}
	if sv.SecretHash != nil {
		return "", "", 0, 0, nil, fmt.Errorf("non emptry secret preimage not supported")
	}
	if sv.Extra != nil {
		return "", "", 0, 0, nil, fmt.Errorf("non emptry extra not supported")
	}
	if int64(sv.MinSettleHeight) != 0 {
		return "", "", 0, 0, nil, fmt.Errorf("non emptry min settle height not supported")
	}
	vb, err := sv.SigningBytes()
	if err != nil {
		return "", "", 0, 0, nil, err
	}
	if sv.Signature.Type != lotuscrypto.SigTypeSecp256k1 {
		return "", "", 0, 0, nil, fmt.Errorf("Unsupported key type")
	}
	b2sum := blake2b.Sum256(vb)
	pub, err := lotuscrypto2.EcRecover(b2sum[:], sv.Signature.Data)
	if err != nil {
		return "", "", 0, 0, nil, err
	}
	sender, err := address.NewSecp256k1Address(pub)
	if err != nil {
		return "", "", 0, 0, nil, err
	}
	return "f" + sender.String()[1:], "f" + sv.ChannelAddr.String()[1:], sv.Lane, sv.Nonce, sv.Amount.Int, nil
}

// getLotusAPI gets lotus API.
//
// @input - context.
//
// @output - api, closer, error.
func (a *filAdapter) getLotusAPI(ctx context.Context) (v0api.FullNode, jsonrpc.ClientCloser, error) {
	var res v0api.FullNodeStruct
	headers := http.Header{}
	if a.authToken != "" {
		headers = http.Header{"Authorization": []string{"Bearer " + a.authToken}}
	}
	closer, err := jsonrpc.NewMergeClient(ctx, a.lotusAP, "Filecoin",
		api.GetInternalStructs(&res), headers)
	if err != nil {
		return nil, nil, err
	}
	return &res, closer, nil
}

// getSignedMessage is used to get the signed message from a unsigned message.
//
// @input - context, api, unsigned message.
//
// @output - signed message, error.
func (a *filAdapter) getSignedMessage(ctx context.Context, api v0api.FullNode, msg *types.Message) (*types.SignedMessage, error) {
	// Get nonce
	nonce, err := api.MpoolGetNonce(ctx, msg.From)
	if err != nil {
		return nil, err
	}
	msg.Nonce = nonce

	// Calculate gas
	limit, err := api.GasEstimateGasLimit(ctx, msg, types.EmptyTSK)
	if err != nil {
		return nil, err
	}
	msg.GasLimit = int64(float64(limit) * 1.25)

	// Calculate premium
	premium, err := api.GasEstimateGasPremium(ctx, 10, msg.From, msg.GasLimit, types.EmptyTSK)
	if err != nil {
		return nil, err
	}
	msg.GasPremium = premium

	// Calculate fee cap
	feeCap, err := api.GasEstimateFeeCap(ctx, msg, 20, types.EmptyTSK)
	if err != nil {
		return nil, err
	}
	msg.GasFeeCap = feeCap

	// Sign message
	sigType, sig, err := a.signer.Sign(ctx, crypto.FIL, msg.Cid().Bytes())
	if err != nil {
		return nil, err
	}
	if sigType != crypto.SECP256K1 {
		return nil, fmt.Errorf("Unsupported key type")
	}
	return &types.SignedMessage{
		Message: *msg,
		Signature: lotuscrypto.Signature{
			Type: lotuscrypto.SigTypeSecp256k1,
			Data: sig,
		},
	}, nil
}
