package chainmgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	crypto2 "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wcgcyx/fcr/internal/crypto"
	"github.com/wcgcyx/fcr/internal/io"
)

// MockChainManagerImplV1 implements ChainManager, it is a mocked version.
type MockChainManagerImplV1 struct {
	h          host.Host
	serverAddr peer.AddrInfo
}

// NewMockChainManagerImplV1 creates a mocked chain manager.
// It takes a server addr as the argument.
// It returns a chain manager and error.
func NewMockChainManagerImplV1(serverAddr peer.AddrInfo) (ChainManager, error) {
	h, err := libp2p.New(context.Background())
	if err != nil {
		return nil, err
	}
	return &MockChainManagerImplV1{h: h, serverAddr: serverAddr}, nil
}

// createJson is for creating a payment channel.
type createJson struct {
	FromAddr string `json:"from_addr"`
	ToAddr   string `json:"to_addr"`
	Amt      string `json:"amt"`
}

// Create is used to create a payment channel.
// It takes a context, private key, recipient address and amount as arguments.
// It returns the channel address and error.
func (mgr *MockChainManagerImplV1) Create(ctx context.Context, prv []byte, toAddr string, amt *big.Int) (string, error) {
	fromAddr, err := crypto.GetAddress(prv)
	if err != nil {
		return "", err
	}
	err = mgr.h.Connect(ctx, mgr.serverAddr)
	if err != nil {
		return "", err
	}
	conn, err := mgr.h.NewStream(ctx, mgr.serverAddr.ID, "create")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	data, _ := json.Marshal(createJson{
		FromAddr: fromAddr,
		ToAddr:   toAddr,
		Amt:      amt.String(),
	})
	err = io.Write(conn, data, time.Minute)
	if err != nil {
		return "", err
	}
	data, err = io.Read(conn, time.Minute)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// topupJson is for topup a payment channel.
type topupJson struct {
	ChAddr string `json:"from_addr"`
	Amt    string `json:"amt"`
}

// Topup is used to topup a payment channel.
// It takes a context, private key, channel address and amount as arguments.
// It returns the error.
func (mgr *MockChainManagerImplV1) Topup(ctx context.Context, prv []byte, chAddr string, amt *big.Int) error {
	err := mgr.h.Connect(ctx, mgr.serverAddr)
	if err != nil {
		return err
	}
	conn, err := mgr.h.NewStream(ctx, mgr.serverAddr.ID, "topup")
	if err != nil {
		return err
	}
	defer conn.Close()
	data, _ := json.Marshal(topupJson{
		ChAddr: chAddr,
		Amt:    amt.String(),
	})
	err = io.Write(conn, data, time.Minute)
	if err != nil {
		return err
	}
	data, err = io.Read(conn, time.Minute)
	if err != nil {
		return err
	}
	if data[0] == 0 {
		return fmt.Errorf("error in topup channel")
	}
	return nil
}

// updateJson is for updating a payment channel.
type updateJson struct {
	ChAddr  string `json:"from_addr"`
	Voucher string `json:"voucher"`
}

// Settle is used to settle a payment channel.
// It takes a context, private key, channel address and final voucher as arguments.
// It returns the error.
func (mgr *MockChainManagerImplV1) Settle(ctx context.Context, prv []byte, chAddr string, voucher string) error {
	settlingAt, curHeight, _, _, _, err := mgr.Check(ctx, chAddr)
	if err != nil {
		return err
	}
	err = mgr.h.Connect(ctx, mgr.serverAddr)
	if err != nil {
		return err
	}
	if settlingAt == 0 {
		// Settle first
		conn, err := mgr.h.NewStream(ctx, mgr.serverAddr.ID, "settle")
		if err != nil {
			return err
		}
		defer conn.Close()
		err = io.Write(conn, []byte(chAddr), time.Minute)
		if err != nil {
			return err
		}
		data, err := io.Read(conn, time.Minute)
		if err != nil {
			return err
		}
		if data[0] == 0 {
			return fmt.Errorf("error in settling channel")
		}
	}
	if voucher != "" {
		if curHeight >= settlingAt {
			return fmt.Errorf("Channel has settled, unable to collect voucher")
		}
		// Update channel
		conn, err := mgr.h.NewStream(ctx, mgr.serverAddr.ID, "update")
		if err != nil {
			return err
		}
		defer conn.Close()
		data, _ := json.Marshal(updateJson{
			ChAddr:  chAddr,
			Voucher: voucher,
		})
		err = io.Write(conn, []byte(chAddr), time.Minute)
		if err != nil {
			return err
		}
		data, err = io.Read(conn, time.Minute)
		if err != nil {
			return err
		}
		if data[0] == 0 {
			return fmt.Errorf("error in updating channel")
		}
	}
	return nil
}

// Collect is used to collect a payment channel.
// It takes a context, private key, channel address as arguments.
// It returns the error.
func (mgr *MockChainManagerImplV1) Collect(ctx context.Context, prv []byte, chAddr string) error {
	err := mgr.h.Connect(ctx, mgr.serverAddr)
	if err != nil {
		return err
	}
	conn, err := mgr.h.NewStream(ctx, mgr.serverAddr.ID, "collect")
	if err != nil {
		return err
	}
	defer conn.Close()
	err = io.Write(conn, []byte(chAddr), time.Minute)
	if err != nil {
		return err
	}
	data, err := io.Read(conn, time.Minute)
	if err != nil {
		return err
	}
	if data[0] == 0 {
		return fmt.Errorf("error in collecting channel")
	}
	return nil
}

// statusJson is the status of a paych.
type statusJson struct {
	FromAddr      string `json:"from_addr"`
	ToAddr        string `json:"to_addr"`
	SettlingAt    int64  `json:"settling_at"`
	CurrentHeight int64  `json:"current_height"`
	Balance       string `json:"balance"`
}

// Check is used to check the status of a payment channel.
// It takes a context, channel address as arguments.
// It returns a settling height, current height, channel balance, sender address, recipient address and error.
func (mgr *MockChainManagerImplV1) Check(ctx context.Context, chAddr string) (int64, int64, *big.Int, string, string, error) {
	err := mgr.h.Connect(ctx, mgr.serverAddr)
	if err != nil {
		return 0, 0, nil, "", "", err
	}
	conn, err := mgr.h.NewStream(ctx, mgr.serverAddr.ID, "check")
	if err != nil {
		return 0, 0, nil, "", "", err
	}
	defer conn.Close()
	err = io.Write(conn, []byte(chAddr), time.Minute)
	if err != nil {
		return 0, 0, nil, "", "", err
	}
	data, err := io.Read(conn, time.Minute)
	if err != nil {
		return 0, 0, nil, "", "", err
	}
	status := statusJson{}
	json.Unmarshal(data, &status)
	balance, _ := big.NewInt(0).SetString(status.Balance, 10)
	return status.SettlingAt, status.CurrentHeight, balance, status.FromAddr, status.ToAddr, nil
}

// voucherJson is the voucher.
type voucherJson struct {
	FromAddr string `json:"from_addr"`
	ChAddr   string `json:"chAddr"`
	Lane     uint64 `json:"lane"`
	Nonce    uint64 `json:"nonce"`
	Redeemed string `json:"redeemed"`
}

// GenerateVoucher is used to generate a voucher.
// It takes the private key, channel address, lane number, nonce, redeemed amount as arguments.
// It returns voucher and error.
func (mgr *MockChainManagerImplV1) GenerateVoucher(prv []byte, chAddr string, lane uint64, nonce uint64, redeemed *big.Int) (string, error) {
	fromAddr, err := crypto.GetAddress(prv)
	if err != nil {
		return "", err
	}
	data, _ := json.Marshal(voucherJson{
		FromAddr: fromAddr,
		ChAddr:   chAddr,
		Lane:     lane,
		Nonce:    nonce,
		Redeemed: redeemed.String(),
	})
	return hex.EncodeToString(data), nil
}

// VerifyVoucher is used to verify a voucher.
// It takes the voucher as the argument.
// It returns the sender's address, channel address, lane number, nonce, redeemed and error.
func (mgr *MockChainManagerImplV1) VerifyVoucher(voucher string) (string, string, uint64, uint64, *big.Int, error) {
	data, err := hex.DecodeString(voucher)
	if err != nil {
		return "", "", 0, 0, nil, err
	}
	v := voucherJson{}
	json.Unmarshal(data, &v)
	redeemed, _ := big.NewInt(0).SetString(v.Redeemed, 10)
	return v.FromAddr, v.ChAddr, v.Lane, v.Nonce, redeemed, nil
}

// mockedChState is the mocked version of a channel state.
type mockedChState struct {
	fromAddr   string
	toAddr     string
	settlingAt int64
	balance    *big.Int
}

// MockChainImplV1 represents a mocked chain.
type MockChainImplV1 struct {
	h host.Host

	blockTime time.Duration

	curHeight   int64
	chStateLock sync.RWMutex
	chState     map[string]*mockedChState
}

// NewMockChainImplV1 creates a mocked chain.
func NewMockChainImplV1(blockTime time.Duration, key string, port int) (*MockChainImplV1, error) {
	var prv crypto2.PrivKey
	var err error
	if key == "" {
		prv, _, err = crypto2.GenerateKeyPair(
			crypto2.Ed25519,
			-1,
		)
		if err != nil {
			return nil, err
		}
	} else {
		key, err := base64.StdEncoding.DecodeString(key)
		if err != nil {
			return nil, err
		}
		prv, err = crypto2.UnmarshalEd25519PrivateKey(key)
		if err != nil {
			return nil, err
		}
	}
	h, err := libp2p.New(
		context.Background(),
		libp2p.Identity(prv),
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%v", port),
		),
	)
	if err != nil {
		return nil, err
	}
	mc := &MockChainImplV1{h: h, blockTime: blockTime, curHeight: 0, chStateLock: sync.RWMutex{}, chState: make(map[string]*mockedChState)}
	h.SetStreamHandler("create", mc.handleCreate)
	h.SetStreamHandler("topup", mc.handleTopup)
	h.SetStreamHandler("settle", mc.handleSettle)
	h.SetStreamHandler("update", mc.handleUpdate)
	h.SetStreamHandler("collect", mc.handleCollect)
	h.SetStreamHandler("check", mc.handleCheck)
	go mc.routine()
	return mc, nil
}

// routine increments chain height.
func (mc *MockChainImplV1) routine() {
	for {
		tc := time.After(mc.blockTime)
		select {
		case <-tc:
			mc.curHeight += 1
		}
	}
}

// GetAddr gets the addr of this server.
func (mc *MockChainImplV1) GetAddr() peer.AddrInfo {
	return peer.AddrInfo{
		ID:    mc.h.ID(),
		Addrs: mc.h.Addrs(),
	}
}

// handleCreate handles create request.
func (mc *MockChainImplV1) handleCreate(conn network.Stream) {
	defer conn.Close()
	data, err := io.Read(conn, time.Minute)
	if err != nil {
		return
	}
	msg := createJson{}
	json.Unmarshal(data, &msg)
	mc.chStateLock.Lock()
	defer mc.chStateLock.Unlock()
	// Generate random channel address
	_, addr, _ := crypto.GenerateKeyPair()
	balance, _ := big.NewInt(0).SetString(msg.Amt, 10)
	mc.chState[addr] = &mockedChState{
		fromAddr:   msg.FromAddr,
		toAddr:     msg.ToAddr,
		settlingAt: 0,
		balance:    balance,
	}
	io.Write(conn, []byte(addr), time.Minute)
}

// handleTopup handles topup request.
func (mc *MockChainImplV1) handleTopup(conn network.Stream) {
	defer conn.Close()
	data, err := io.Read(conn, time.Minute)
	if err != nil {
		return
	}
	msg := topupJson{}
	json.Unmarshal(data, &msg)
	mc.chStateLock.RLock()
	state, ok := mc.chState[msg.ChAddr]
	mc.chStateLock.RUnlock()
	if !ok {
		io.Write(conn, []byte{0}, time.Minute)
		return
	}
	mc.chStateLock.Lock()
	defer mc.chStateLock.Unlock()
	amt, _ := big.NewInt(0).SetString(msg.Amt, 10)
	state.balance.Add(state.balance, amt)
	io.Write(conn, []byte{1}, time.Minute)
}

// handleSettle handles settle request.
func (mc *MockChainImplV1) handleSettle(conn network.Stream) {
	defer conn.Close()
	data, err := io.Read(conn, time.Minute)
	if err != nil {
		return
	}
	chAddr := string(data)
	mc.chStateLock.RLock()
	state, ok := mc.chState[chAddr]
	mc.chStateLock.RUnlock()
	if !ok || state.settlingAt != 0 {
		io.Write(conn, []byte{0}, time.Minute)
		return
	}
	mc.chStateLock.Lock()
	defer mc.chStateLock.Unlock()
	// 4 blocks settlement delay
	state.settlingAt = mc.curHeight + 4
	io.Write(conn, []byte{1}, time.Minute)
}

// handleUpdate handles update request.
func (mc *MockChainImplV1) handleUpdate(conn network.Stream) {
	defer conn.Close()
	data, err := io.Read(conn, time.Minute)
	if err != nil {
		return
	}
	msg := updateJson{}
	json.Unmarshal(data, &msg)
	mc.chStateLock.RLock()
	defer mc.chStateLock.RUnlock()
	state, ok := mc.chState[msg.ChAddr]
	if !ok || state.settlingAt >= mc.curHeight {
		io.Write(conn, []byte{0}, time.Minute)
		return
	}
	io.Write(conn, []byte{1}, time.Minute)
}

// handleCollect handles collect request.
func (mc *MockChainImplV1) handleCollect(conn network.Stream) {
	defer conn.Close()
	data, err := io.Read(conn, time.Minute)
	if err != nil {
		return
	}
	chAddr := string(data)
	mc.chStateLock.RLock()
	defer mc.chStateLock.RUnlock()
	state, ok := mc.chState[chAddr]
	if !ok || state.settlingAt > mc.curHeight {
		io.Write(conn, []byte{0}, time.Minute)
		return
	}
	io.Write(conn, []byte{1}, time.Minute)
}

// handleCheck handles check request.
func (mc *MockChainImplV1) handleCheck(conn network.Stream) {
	defer conn.Close()
	data, err := io.Read(conn, time.Minute)
	if err != nil {
		return
	}
	chAddr := string(data)
	mc.chStateLock.RLock()
	defer mc.chStateLock.RUnlock()
	state, ok := mc.chState[chAddr]
	if !ok {
		io.Write(conn, []byte{0}, time.Minute)
		return
	}
	data, _ = json.Marshal(statusJson{
		FromAddr:      state.fromAddr,
		ToAddr:        state.toAddr,
		SettlingAt:    state.settlingAt,
		CurrentHeight: mc.curHeight,
		Balance:       state.balance.String(),
	})
	io.Write(conn, data, time.Minute)
}
