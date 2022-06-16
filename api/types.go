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
	"math/big"
	"time"

	"github.com/wcgcyx/fcr/peermgr"
)

// Structs for user API.
type SignerGetAddrRes struct {
	KeyType byte
	Addr    string
}

type PeerMgrListHistoryRes struct {
	RecID  *big.Int
	Record peermgr.Record
}

type TransactorCheckRes struct {
	SettlingAt    int64
	MinSettlingAt int64
	CurrentHeight int64
	Redeemed      *big.Int
	Balance       *big.Int
	Sender        string
	Recipient     string
}

type PservMgrInspectRes struct {
	Served bool
	PPP    *big.Int
	Period *big.Int
}

type PayMgrReserveForSelfRes struct {
	ResCh string
	ResID uint64
}

type PayMgrReserveForSelfWithOfferRes struct {
	ResCh string
	ResID uint64
}

type ReservMgrPolicyRes struct {
	Unlimited bool
	Max       *big.Int
}

type PaychMonitorCheckRes struct {
	Updated    time.Time
	Settlement time.Time
}

type PieceMgrInspectRes struct {
	Exists bool
	Path   string
	Index  int
	Size   uint64
	Copy   bool
}

type CServMgrInspectRes struct {
	Served bool
	PPB    *big.Int
}

type MPSGetMinerProofRes struct {
	Exists       bool
	MinerKeyType byte
	MinerAddr    string
	Proof        []byte
}

type RetMgrRetrieveRes struct {
	Progress string
	Err      string
}

// Structs for dev API.
type SignerSignRes struct {
	SigType byte
	Sig     []byte
}

type TransactorVerifyVoucherRes struct {
	Sender   string
	ChAddr   string
	Lane     uint64
	Nonce    uint64
	Redeemed *big.Int
}
