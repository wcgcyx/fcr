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
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	lotusbig "github.com/filecoin-project/go-state-types/big"
	lotuscrypto "github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	init7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/init"
	paych7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/paych"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

// MockFil is a mocked filecoin network.
type MockFil struct {
	Server *httptest.Server

	Lock       sync.RWMutex
	stop       chan bool
	blockTime  time.Duration
	Height     int
	PaychBals  map[string]*big.Int
	PaychStats map[string]*paych7.State
	transRes   map[string][]byte
}

// NewMockFil starts a new mocked filecoin network.
//
// @output - mocked fil.
func NewMockFil(blockTime time.Duration) *MockFil {
	res := MockFil{}
	server := jsonrpc.NewServer()
	server.Register("Filecoin", &res)
	res.Server = httptest.NewServer(server)
	res.Lock = sync.RWMutex{}
	res.stop = make(chan bool)
	res.blockTime = blockTime
	res.Height = 0
	res.PaychBals = make(map[string]*big.Int)
	res.PaychStats = make(map[string]*paych7.State)
	res.transRes = make(map[string][]byte)
	go res.tickRoutine()
	return &res
}

// Shutdown shuts down the mocked filecoin network.
func (m *MockFil) Shutdown() {
	m.Server.Close()
}

// GetAPI gets the API address.
//
// @output - api address.
func (m *MockFil) GetAPI() string {
	return "http://" + m.Server.Listener.Addr().String()
}

// tickRoutine is for advancing block height.
func (m *MockFil) tickRoutine() {
	// Default Block time is 3 seconds.
	var tick *time.Ticker
	if m.blockTime == 0 {
		tick = time.NewTicker(3 * time.Second)
	} else {
		tick = time.NewTicker(m.blockTime)
	}
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			m.Lock.Lock()
			m.Height += 1
			m.Lock.Unlock()
		case <-m.stop:
			return
		}
	}
}

// Lotus API mock: ChainHead.
func (m *MockFil) ChainHead(ctx context.Context) (*types.TipSet, error) {
	addr, err := address.NewIDAddress(12512063)
	if err != nil {
		return nil, err
	}
	c, err := cid.Decode("bafyreicmaj5hhoy5mgqvamfhgexxyergw7hdeshizghodwkjg6qmpoco7i")
	if err != nil {
		return nil, err
	}
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	return types.NewTipSet([]*types.BlockHeader{
		{
			Miner: addr,
			Ticket: &types.Ticket{
				VRFProof: []byte("vrf proof0000000vrf proof0000000"),
			},
			ElectionProof: &types.ElectionProof{
				VRFProof: []byte("vrf proof0000000vrf proof0000000"),
			},
			Parents:               []cid.Cid{c, c},
			ParentMessageReceipts: c,
			BLSAggregate:          &lotuscrypto.Signature{Type: lotuscrypto.SigTypeBLS, Data: []byte("boo! im a signature")},
			ParentWeight:          lotusbig.NewInt(123125126212),
			Messages:              c,
			Height:                abi.ChainEpoch(m.Height),
			ParentStateRoot:       c,
			BlockSig:              &lotuscrypto.Signature{Type: lotuscrypto.SigTypeBLS, Data: []byte("boo! im a signature")},
			ParentBaseFee:         lotusbig.NewInt(3432432843291),
		},
	})
}

// Lotus API mock: MpoolGetNonce.
func (m *MockFil) MpoolGetNonce(context.Context, address.Address) (uint64, error) {
	// Nonce does not matter in this mock.
	return 1, nil
}

// Lotus API mock: GasEstimateGasLimit.
func (m *MockFil) GasEstimateGasLimit(context.Context, *types.Message, types.TipSetKey) (int64, error) {
	// Gas does not matter in this mock.
	return 100, nil
}

// Lotus API mock: GasEstimateGasPremium.
func (m *MockFil) GasEstimateGasPremium(context.Context, uint64, address.Address, int64, types.TipSetKey) (types.BigInt, error) {
	// Gas does not matter in this mock.
	return lotusbig.NewFromGo(big.NewInt(100)), nil
}

// Lotus API mock: GasEstimateFeeCap.
func (m *MockFil) GasEstimateFeeCap(context.Context, *types.Message, int64, types.TipSetKey) (types.BigInt, error) {
	// Gas does not matter in this mock.
	return lotusbig.NewFromGo(big.NewInt(100)), nil
}

// Lotus API mock: MpoolPush.
func (m *MockFil) MpoolPush(ctx context.Context, msg *types.SignedMessage) (cid.Cid, error) {
	// There are a few possible messages used in this system.
	// Value transfer.
	// Paych creation.
	// Paych update.
	// Paych settle.
	// Paych collect.
	m.Lock.Lock()
	defer m.Lock.Unlock()
	// Create tx hash.
	pref := cid.Prefix{
		Version:  0,
		Codec:    cid.DagProtobuf,
		MhType:   multihash.SHA2_256,
		MhLength: -1, // Default length
	}
	seed := make([]byte, 32)
	rand.Read(seed)
	txHash, err := pref.Sum(seed)
	if err != nil {
		return cid.Cid{}, err
	}
	if msg.Message.Method == 0 {
		// It is topup.
		bal, ok := m.PaychBals[msg.Message.To.String()]
		if ok {
			bal.Add(bal, msg.Message.Value.Int)
		}
	} else if msg.Message.Method == 2 && msg.Message.To.String()[1:] == "01" {
		// It is create.
		// Create - 2 & to init (string repr is f01... x01) with value to be the initial balance.
		var decodedParams init7.ExecParams
		err := decodedParams.UnmarshalCBOR(bytes.NewReader(msg.Message.Params))
		if err != nil {
			return cid.Cid{}, err
		}
		var constructorParams paych7.ConstructorParams
		err = constructorParams.UnmarshalCBOR(bytes.NewReader(decodedParams.ConstructorParams))
		if err != nil {
			return cid.Cid{}, err
		}
		randomAddr := make([]byte, 33)
		rand.Read(randomAddr)
		chAddr, err := address.NewSecp256k1Address(randomAddr)
		if err != nil {
			return cid.Cid{}, err
		}
		ret := init7.ExecReturn{
			IDAddress:     chAddr,
			RobustAddress: chAddr,
		}
		var b bytes.Buffer
		tmp := bufio.NewWriter(&b)
		err = ret.MarshalCBOR(tmp)
		if err != nil {
			return cid.Cid{}, err
		}
		tmp.Flush()
		m.transRes[txHash.String()] = b.Bytes()
		// New paych state
		m.PaychBals[chAddr.String()] = msg.Message.Value.Int
		pref = cid.Prefix{
			Version:  0,
			Codec:    cid.DagProtobuf,
			MhType:   multihash.SHA2_256,
			MhLength: -1, // Default length
		}
		objectHash, err := pref.Sum([]byte(chAddr.String()))
		if err != nil {
			return cid.Cid{}, err
		}
		m.PaychStats[objectHash.String()] = &paych7.State{
			From:            constructorParams.From,
			To:              constructorParams.To,
			ToSend:          abi.NewTokenAmount(0),
			SettlingAt:      abi.ChainEpoch(0),
			MinSettleHeight: abi.ChainEpoch(0),
			// Lane states does not matter.
			LaneStates: txHash,
		}
	} else if msg.Message.Method == 2 {
		// It is update.
		// Update - 2 & to chAddr.
		var updateParams paych7.UpdateChannelStateParams
		err := updateParams.UnmarshalCBOR(bytes.NewReader(msg.Message.Params))
		if err != nil {
			return cid.Cid{}, err
		}
		chAddr := updateParams.Sv.ChannelAddr
		newRedeemed := updateParams.Sv.Amount
		pref = cid.Prefix{
			Version:  0,
			Codec:    cid.DagProtobuf,
			MhType:   multihash.SHA2_256,
			MhLength: -1, // Default length
		}
		objectHash, err := pref.Sum([]byte(chAddr.String()))
		if err != nil {
			return cid.Cid{}, err
		}
		cs, ok := m.PaychStats[objectHash.String()]
		if ok {
			cs.ToSend = newRedeemed
		}
	} else if msg.Message.Method == 3 {
		// It is settle.
		// Settle - 3 & to chAddr.
		pref = cid.Prefix{
			Version:  0,
			Codec:    cid.DagProtobuf,
			MhType:   multihash.SHA2_256,
			MhLength: -1, // Default length
		}
		objectHash, err := pref.Sum([]byte(msg.Message.To.String()))
		if err != nil {
			return cid.Cid{}, err
		}
		cs, ok := m.PaychStats[objectHash.String()]
		if !ok || (ok && cs.SettlingAt != 0) {
			return cid.Cid{}, fmt.Errorf("fail to settle channel")
		}
		// Default settling height is 1000.
		cs.SettlingAt = abi.ChainEpoch(m.Height + 1000)
	} else if msg.Message.Method == 4 {
		// It is collect.
		// Collect - 4 & to chAddr.
		pref = cid.Prefix{
			Version:  0,
			Codec:    cid.DagProtobuf,
			MhType:   multihash.SHA2_256,
			MhLength: -1, // Default length
		}
		objectHash, err := pref.Sum([]byte(msg.Message.To.String()))
		if err != nil {
			return cid.Cid{}, err
		}
		cs, ok := m.PaychStats[objectHash.String()]
		if !ok || (ok && cs.SettlingAt >= abi.ChainEpoch(m.Height)) {
			return cid.Cid{}, fmt.Errorf("fail to collect channel")
		}
		delete(m.PaychStats, objectHash.String())
	}
	return txHash, nil
}

// Lotus API mock: StateWaitMsg.
func (m *MockFil) StateWaitMsg(ctx context.Context, cid cid.Cid, confidence uint64) (*api.MsgLookup, error) {
	m.Lock.Lock()
	old := m.Height
	m.Lock.Unlock()
	for {
		m.Lock.Lock()
		if m.Height > old {
			defer m.Lock.Unlock()
			break
		}
		m.Lock.Unlock()
		time.Sleep(1 * time.Millisecond)
	}
	ret, ok := m.transRes[cid.String()]
	if ok {
		delete(m.transRes, cid.String())
		return &api.MsgLookup{Message: cid, Receipt: types.MessageReceipt{ExitCode: 0, Return: ret, GasUsed: 100}}, nil
	} else {
		return &api.MsgLookup{Message: cid, Receipt: types.MessageReceipt{ExitCode: 0, Return: []byte{0}, GasUsed: 100}}, nil
	}
}

// Lotus API mock: StateGetActor.
func (m *MockFil) StateGetActor(ctx context.Context, actor address.Address, tsk types.TipSetKey) (*types.Actor, error) {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	pref := cid.Prefix{
		Version:  0,
		Codec:    cid.DagProtobuf,
		MhType:   multihash.SHA2_256,
		MhLength: -1, // Default length
	}
	objectHash, err := pref.Sum([]byte(actor.String()))
	if err != nil {
		return nil, err
	}
	bal, ok := m.PaychBals[actor.String()]
	if ok {
		return &types.Actor{Head: objectHash, Balance: lotusbig.NewFromGo(bal)}, nil
	}
	return &types.Actor{Head: objectHash, Balance: lotusbig.NewFromGo(big.NewInt(0))}, nil
}

// Lotus API mock: ChainReadObj.
func (m *MockFil) ChainReadObj(ctx context.Context, cid cid.Cid) ([]byte, error) {
	res, ok := m.PaychStats[cid.String()]
	if !ok {
		return nil, fmt.Errorf("cannot find object %v", cid.String())
	}
	var b bytes.Buffer
	tmp := bufio.NewWriter(&b)
	err := res.MarshalCBOR(tmp)
	if err != nil {
		return nil, err
	}
	tmp.Flush()
	return b.Bytes(), nil
}

// Lotus API mock: StateAccountKey.
func (m *MockFil) StateAccountKey(ctx context.Context, addr address.Address, tx types.TipSetKey) (address.Address, error) {
	return addr, nil
}
