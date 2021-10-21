package chainmgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/filecoin-project/go-address"
	crypto3 "github.com/filecoin-project/go-crypto"
	"github.com/filecoin-project/go-jsonrpc"
	lotusbig "github.com/filecoin-project/go-state-types/big"
	crypto2 "github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/actors/builtin/paych"
	"github.com/filecoin-project/lotus/chain/types"
	init4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/init"
	paych2 "github.com/filecoin-project/specs-actors/v4/actors/builtin/paych"
	"github.com/ipfs/go-cid"
	"github.com/minio/blake2b-simd"
	"github.com/wcgcyx/fcr/internal/crypto"
)

// FILChainManagerImplV1 implements ChainManager, it if for FIL.
type FILChainManagerImplV1 struct {
	lotusAPIAddr string
	authToken    string
}

// NewFILChainManagerImplV1 creates a filecoin chain manager.
// It takes a lotus api addr and auth token as arguments.
// It returns a chain manager and error.
func NewFILChainManagerImplV1(lotusAPIAddr string, authToken string) (ChainManager, error) {
	return &FILChainManagerImplV1{lotusAPIAddr: lotusAPIAddr, authToken: authToken}, nil
}

// Create is used to create a payment channel.
// It takes a context, private key, recipient address and amount as arguments.
// It returns the channel address and error.
func (mgr *FILChainManagerImplV1) Create(ctx context.Context, prv []byte, toAddrStr string, amt *big.Int) (string, error) {
	fromAddrStr, err := crypto.GetAddress(prv)
	if err != nil {
		return "", err
	}
	fromAddr, err := address.NewFromString(fromAddrStr)
	if err != nil {
		return "", err
	}
	toAddr, err := address.NewFromString(toAddrStr)
	if err != nil {
		return "", err
	}
	// Get API
	api, closer, err := getRemoteLotusAPI(ctx, mgr.authToken, mgr.lotusAPIAddr)
	if err != nil {
		return "", err
	}
	if closer != nil {
		defer closer()
	}
	// Message builder
	builder := paych.Message(actors.Version4, fromAddr)
	msg, err := builder.Create(toAddr, lotusbig.NewFromGo(amt))
	if err != nil {
		return "", err
	}
	// Get signed message
	signedMsg, err := fillMsg(ctx, prv, api, msg)
	if err != nil {
		return "", err
	}
	contentID, err := api.MpoolPush(ctx, signedMsg)
	if err != nil {
		return "", err
	}
	receipt, err := waitReceipt(ctx, &contentID, api)
	if err != nil {
		return "", err
	}
	if receipt.ExitCode != 0 {
		return "", fmt.Errorf("Transaction fails to execute: %s", receipt.ExitCode.Error())
	}
	var decodedReturn init4.ExecReturn
	err = decodedReturn.UnmarshalCBOR(bytes.NewReader(receipt.Return))
	if err != nil {
		return "", fmt.Errorf("Payment manager has error unmarshal receipt: %v", receipt)
	}
	return "f" + decodedReturn.RobustAddress.String()[1:], nil
}

// Topup is used to topup a payment channel.
// It takes a context, private key, channel address and amount as arguments.
// It returns the error.
func (mgr *FILChainManagerImplV1) Topup(ctx context.Context, prv []byte, chAddr string, amt *big.Int) error {
	fromAddrStr, err := crypto.GetAddress(prv)
	if err != nil {
		return err
	}
	fromAddr, err := address.NewFromString(fromAddrStr)
	if err != nil {
		return err
	}
	toAddr, err := address.NewFromString(chAddr)
	if err != nil {
		return err
	}
	// Get API
	api, closer, err := getRemoteLotusAPI(ctx, mgr.authToken, mgr.lotusAPIAddr)
	if err != nil {
		return err
	}
	if closer != nil {
		defer closer()
	}
	msg := &types.Message{
		To:     toAddr,
		From:   fromAddr,
		Value:  lotusbig.NewFromGo(amt),
		Method: 0,
	}
	// Get signed message
	signedMsg, err := fillMsg(ctx, prv, api, msg)
	if err != nil {
		return err
	}
	contentID, err := api.MpoolPush(ctx, signedMsg)
	if err != nil {
		return err
	}
	receipt, err := waitReceipt(ctx, &contentID, api)
	if err != nil {
		return err
	}
	if receipt.ExitCode != 0 {
		return fmt.Errorf("Transaction fails to execute: %s", receipt.ExitCode.Error())
	}
	return nil
}

// Settle is used to settle a payment channel.
// It takes a context, private key, channel address and final voucher as arguments.
// It returns the error.
func (mgr *FILChainManagerImplV1) Settle(ctx context.Context, prv []byte, chAddr string, voucher string) error {
	fromAddrStr, err := crypto.GetAddress(prv)
	if err != nil {
		return err
	}
	fromAddr, err := address.NewFromString(fromAddrStr)
	if err != nil {
		return err
	}
	toAddr, err := address.NewFromString(chAddr)
	if err != nil {
		return err
	}
	// Get API
	api, closer, err := getRemoteLotusAPI(ctx, mgr.authToken, mgr.lotusAPIAddr)
	if err != nil {
		return err
	}
	if closer != nil {
		defer closer()
	}
	// Message builder
	builder := paych.Message(actors.Version4, fromAddr)
	msg, err := builder.Settle(toAddr)
	if err != nil {
		return err
	}
	// Get signed message
	signedMsg, err := fillMsg(ctx, prv, api, msg)
	if err != nil {
		return err
	}
	contentID, err := api.MpoolPush(ctx, signedMsg)
	if err != nil {
		return err
	}
	receipt, err := waitReceipt(ctx, &contentID, api)
	if err != nil {
		return err
	}
	if receipt.ExitCode != 0 {
		return fmt.Errorf("Transaction fails to execute: %s", receipt.ExitCode.Error())
	}
	return nil
}

// Collect is used to collect a payment channel.
// It takes a context, private key, channel address as arguments.
// It returns the error.
func (mgr *FILChainManagerImplV1) Collect(ctx context.Context, prv []byte, chAddr string) error {
	fromAddrStr, err := crypto.GetAddress(prv)
	if err != nil {
		return err
	}
	fromAddr, err := address.NewFromString(fromAddrStr)
	if err != nil {
		return err
	}
	toAddr, err := address.NewFromString(chAddr)
	if err != nil {
		return err
	}
	// Get API
	api, closer, err := getRemoteLotusAPI(ctx, mgr.authToken, mgr.lotusAPIAddr)
	if err != nil {
		return err
	}
	if closer != nil {
		defer closer()
	}
	// Message builder
	builder := paych.Message(actors.Version4, fromAddr)
	msg, err := builder.Collect(toAddr)
	if err != nil {
		return err
	}
	// Get signed message
	signedMsg, err := fillMsg(ctx, prv, api, msg)
	if err != nil {
		return err
	}
	contentID, err := api.MpoolPush(ctx, signedMsg)
	if err != nil {
		return err
	}
	receipt, err := waitReceipt(ctx, &contentID, api)
	if err != nil {
		return err
	}
	if receipt.ExitCode != 0 {
		return fmt.Errorf("Transaction fails to execute: %s", receipt.ExitCode.Error())
	}
	return nil
}

// Check is used to check the status of a payment channel.
// It takes a context, channel address as arguments.
// It returns a settling height, current height, channel balance, sender address, recipient address and error.
func (mgr *FILChainManagerImplV1) Check(ctx context.Context, chAddr string) (int64, int64, *big.Int, string, string, error) {
	toAddr, err := address.NewFromString(chAddr)
	if err != nil {
		return 0, 0, nil, "", "", err
	}
	// Get API
	api, closer, err := getRemoteLotusAPI(ctx, mgr.authToken, mgr.lotusAPIAddr)
	if err != nil {
		return 0, 0, nil, "", "", err
	}
	if closer != nil {
		defer closer()
	}
	// Get actor state
	actor, err := api.StateGetActor(ctx, toAddr, types.EmptyTSK)
	if err != nil {
		return 0, 0, nil, "", "", err
	}
	data, err := api.ChainReadObj(ctx, actor.Head)
	if err != nil {
		return 0, 0, nil, "", "", err
	}
	state := paych2.State{}
	err = state.UnmarshalCBOR(bytes.NewReader(data))
	if err != nil {
		return 0, 0, nil, "", "", err
	}
	recipient, err := api.StateAccountKey(ctx, state.To, types.EmptyTSK)
	if err != nil {
		return 0, 0, nil, "", "", err
	}
	sender, err := api.StateAccountKey(ctx, state.From, types.EmptyTSK)
	if err != nil {
		return 0, 0, nil, "", "", err
	}
	head, err := api.ChainHead(ctx)
	if err != nil {
		return 0, 0, nil, "", "", err
	}
	return int64(state.SettlingAt), int64(head.Height()), actor.Balance.Int, "f" + sender.String()[1:], "f" + recipient.String()[1:], nil
}

// GenerateVoucher is used to generate a voucher.
// It takes the private key, channel address, lane number, nonce, redeemed amount as arguments.
// It returns voucher and error.
func (mgr *FILChainManagerImplV1) GenerateVoucher(prv []byte, chAddr string, lane uint64, nonce uint64, redeemed *big.Int) (string, error) {
	addr, err := address.NewFromString(chAddr)
	if err != nil {
		return "", err
	}
	sv := &paych.SignedVoucher{
		ChannelAddr: addr,
		Lane:        lane,
		Nonce:       nonce,
		Amount:      lotusbig.NewFromGo(redeemed),
	}
	vb, err := sv.SigningBytes()
	if err != nil {
		return "", err
	}
	sig, err := crypto.Sign(prv, vb)
	if err != nil {
		return "", err
	}
	sv.Signature = &crypto2.Signature{
		Type: crypto2.SigTypeSecp256k1,
		Data: sig,
	}
	voucher, err := encodedVoucher(sv)
	if err != nil {
		return "", err
	}
	return voucher, nil
}

// VerifyVoucher is used to verify a voucher.
// It takes the voucher as the argument.
// It returns the sender's address, channel address, lane number, nonce, redeemed and error.
func (mgr *FILChainManagerImplV1) VerifyVoucher(voucher string) (string, string, uint64, uint64, *big.Int, error) {
	sv, err := paych.DecodeSignedVoucher(voucher)
	if err != nil {
		return "", "", 0, 0, nil, err
	}
	vb, err := sv.SigningBytes()
	if err != nil {
		return "", "", 0, 0, nil, err
	}
	if sv.Signature.Type != crypto2.SigTypeSecp256k1 {
		return "", "", 0, 0, nil, fmt.Errorf("wrong signature type")
	}
	b2sum := blake2b.Sum256(vb)
	pub, err := crypto3.EcRecover(b2sum[:], sv.Signature.Data)
	if err != nil {
		return "", "", 0, 0, nil, err
	}
	sender, err := address.NewSecp256k1Address(pub)
	if err != nil {
		return "", "", 0, 0, nil, err
	}
	return sender.String(), sv.ChannelAddr.String(), sv.Lane, sv.Nonce, sv.Amount.Int, nil
}

// getRemoteLotusAPI gets the api that interacts with lotus for a given lotus api addr and access token.
// It takes a context, a authToken, a lotusAPIAddr as arguments.
// It returns api node, closer and error.
func getRemoteLotusAPI(ctx context.Context, authToken, lotusAPIAddr string) (*v0api.FullNodeStruct, jsonrpc.ClientCloser, error) {
	var api v0api.FullNodeStruct
	headers := http.Header{"Authorization": []string{"Bearer " + authToken}}
	closer, err := jsonrpc.NewMergeClient(ctx, lotusAPIAddr, "Filecoin", []interface{}{&api.Internal, &api.CommonStruct.Internal}, headers)
	if err != nil {
		return nil, nil, err
	}
	return &api, closer, nil
}

// fillMsg will fill the gas and sign a given message.
// It takes a context, a private key, api node, a message as arguments.
// It returns a signed message and error.
func fillMsg(ctx context.Context, prv []byte, api *v0api.FullNodeStruct, msg *types.Message) (*types.SignedMessage, error) {
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

	premium, err := api.GasEstimateGasPremium(ctx, 10, msg.From, msg.GasLimit, types.EmptyTSK)
	if err != nil {
		return nil, err
	}
	msg.GasPremium = premium

	feeCap, err := api.GasEstimateFeeCap(ctx, msg, 20, types.EmptyTSK)
	if err != nil {
		return nil, err
	}
	msg.GasFeeCap = feeCap
	// Sign message
	sig, err := crypto.Sign(prv, msg.Cid().Bytes())
	if err != nil {
		return nil, err
	}
	return &types.SignedMessage{
		Message: *msg,
		Signature: crypto2.Signature{
			Type: crypto2.SigTypeSecp256k1,
			Data: sig,
		},
	}, nil
}

// waitReceipt will wait until receipt is received for a given cid.
// It takes a context, a cid of the message, api node as arguments.
// It returns a message receipt and error.
func waitReceipt(ctx context.Context, cid *cid.Cid, api *v0api.FullNodeStruct) (*types.MessageReceipt, error) {
	// Return until recipient is returned (transaction is processed)
	var receipt *types.MessageReceipt
	var err error
	for {
		receipt, err = api.StateGetReceipt(ctx, *cid, types.EmptyTSK)
		if err != nil {
			return nil, err
		}
		if receipt != nil {
			break
		}
		tc := time.After(5 * time.Second)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-tc:
		}
	}
	return receipt, nil
}

// encodedVoucher returns the encoded string of a given signed voucher
// It takes a signed voucher as arguments.
// It returns voucher string and error.
func encodedVoucher(sv *paych.SignedVoucher) (string, error) {
	buf := new(bytes.Buffer)
	if err := sv.MarshalCBOR(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf.Bytes()), nil
}
