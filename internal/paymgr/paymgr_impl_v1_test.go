package paymgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	badgerds "github.com/ipfs/go-ds-badger"
	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/fcr/internal/paymgr/chainmgr"
	"github.com/wcgcyx/fcr/internal/payoffer"
	"github.com/wcgcyx/fcr/internal/payofferstore"
)

const (
	testDB      = "./testdb"
	testDB1     = "./testdb1"
	testDB2     = "./testdb2"
	testDB3     = "./testdb3"
	testDB4     = "./testdb4"
	testDB5     = "./testdb5"
	testKeyDB   = "./testkeydb"
	testPrvKey1 = "c36b6afe6c722c9c8dcdfc34a7ebf60e391ce8bd35aa4b84d1714859453b5b7d"
	testPrvKey2 = "5ec4c96ebb5843dce880dc3e985b32f0d03aa6648f43a239be33778167f12335"
	testPrvKey3 = "672cb8709adfd269138152c4fa46da2da5242f24521ebe7bddd97dee1c542323"
	testPrvKey4 = "6473485c551d942ae4540f31d55cdbe4ac924ac7c17eef34f7b2da1c6bc9c891"
	testPrvKey5 = "d10a3513c23196af699fd2c8aed4908431c36b1386dfa5a5e1d159adf584ae25"
)

func TestNewPaymentManager(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	os.RemoveAll(testKeyDB)
	os.Mkdir(testKeyDB, os.ModePerm)
	defer os.RemoveAll(testKeyDB)
	ctx, cancel := context.WithCancel(context.Background())
	// New mock chain manager
	mockChain, err := chainmgr.NewMockChainImplV1(time.Minute, "", 0)
	assert.Empty(t, err)
	assert.NotEmpty(t, mockChain)
	mockChainManager, err := chainmgr.NewMockChainManagerImplV1(mockChain.GetAddr())
	assert.Empty(t, err)
	assert.NotEmpty(t, mockChainManager)
	// Set ID
	id := "1"
	// New payoffer store
	os, err := payofferstore.NewPayOfferStoreImplV1(ctx, testDB)
	assert.Empty(t, err)
	assert.NotEmpty(t, os)
	// New payment manager
	mgr, err := makePaymentManager(ctx, map[uint64]string{0: testPrvKey1}, id, mockChainManager, os, testDB)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr)

	_, err = mgr.GetPrvKey(1)
	assert.NotEmpty(t, err)

	_, err = mgr.GetPrvKey(0)
	assert.Empty(t, err)

	_, err = mgr.GetRootAddress(1)
	assert.NotEmpty(t, err)

	_, err = mgr.GetRootAddress(0)
	assert.Empty(t, err)

	cancel()
	time.Sleep(1 * time.Second)
}

func TestCreateCh(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	os.RemoveAll(testKeyDB)
	os.Mkdir(testKeyDB, os.ModePerm)
	defer os.RemoveAll(testKeyDB)
	ctx, cancel := context.WithCancel(context.Background())
	// New mock chain manager
	mockChain, err := chainmgr.NewMockChainImplV1(time.Minute, "", 0)
	assert.Empty(t, err)
	assert.NotEmpty(t, mockChain)
	mockChainManager, err := chainmgr.NewMockChainManagerImplV1(mockChain.GetAddr())
	assert.Empty(t, err)
	assert.NotEmpty(t, mockChainManager)
	// Set ID
	id := "1"
	// New payoffer store
	os, err := payofferstore.NewPayOfferStoreImplV1(ctx, testDB)
	assert.Empty(t, err)
	assert.NotEmpty(t, os)
	// New payment manager
	mgr, err := makePaymentManager(ctx, map[uint64]string{0: testPrvKey1}, id, mockChainManager, os, testDB)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr)

	_, err = mgr.CreateOutboundCh(ctx, 1, "target-test", big.NewInt(10), time.Now().Add(time.Hour).Unix())
	assert.NotEmpty(t, err)

	_, err = mgr.CreateOutboundCh(ctx, 0, "target-test", big.NewInt(10), time.Now().Add(2*time.Second).Unix())
	assert.Empty(t, err)

	_, err = mgr.CreateOutboundCh(ctx, 0, "target-test", big.NewInt(10), time.Now().Add(time.Hour).Unix())
	assert.NotEmpty(t, err)

	time.Sleep(2 * time.Second)

	_, err = mgr.CreateOutboundCh(ctx, 0, "target-test", big.NewInt(10), time.Now().Add(1*time.Second).Unix())
	assert.Empty(t, err)

	res, err := mgr.ListActiveOutboundChs(ctx)
	assert.Empty(t, err)
	assert.Equal(t, 1, len(res[0]))

	res, err = mgr.ListOutboundChs(ctx)
	assert.Empty(t, err)
	assert.Equal(t, 2, len(res[0]))

	_, _, _, _, _, _, _, err = mgr.InspectOutboundCh(ctx, 1, res[0][0])
	assert.NotEmpty(t, err)
	_, _, _, _, _, _, _, err = mgr.InspectOutboundCh(ctx, 1, "testaddr")
	assert.NotEmpty(t, err)

	_, _, _, _, active0, _, _, err := mgr.InspectOutboundCh(ctx, 0, res[0][0])
	assert.Empty(t, err)
	_, _, _, _, active1, _, _, err := mgr.InspectOutboundCh(ctx, 0, res[0][1])
	assert.Empty(t, err)
	assert.True(t, active0 || active1)

	time.Sleep(1 * time.Second)

	_, _, _, _, active0, _, _, err = mgr.InspectOutboundCh(ctx, 0, res[0][0])
	assert.Empty(t, err)
	_, _, _, _, active1, _, _, err = mgr.InspectOutboundCh(ctx, 0, res[0][1])
	assert.Empty(t, err)
	assert.False(t, active0 || active1)

	chAddr, err := mgr.CreateOutboundCh(ctx, 0, "target-test", big.NewInt(10), time.Now().Add(time.Second).Unix())
	assert.Empty(t, err)

	err = mockChainManager.Settle(ctx, []byte{}, chAddr, "")
	assert.Empty(t, err)

	_, _, balance, _, _, _, _, err := mgr.InspectOutboundCh(ctx, 0, chAddr)
	assert.Empty(t, err)
	assert.Equal(t, "10", balance.String())

	err = mgr.TopupOutboundCh(ctx, 1, "target-test", big.NewInt(10))
	assert.NotEmpty(t, err)

	err = mgr.TopupOutboundCh(ctx, 0, "target-test2", big.NewInt(10))
	assert.NotEmpty(t, err)

	err = mgr.TopupOutboundCh(ctx, 0, "target-test", big.NewInt(10))
	assert.Empty(t, err)

	time.Sleep(1 * time.Second)

	err = mgr.TopupOutboundCh(ctx, 0, "target-test", big.NewInt(10))
	assert.NotEmpty(t, err)

	_, _, balance, _, _, _, _, err = mgr.InspectOutboundCh(ctx, 0, chAddr)
	assert.Empty(t, err)
	assert.Equal(t, "20", balance.String())

	cancel()
	time.Sleep(1 * time.Second)
}

func TestSettleCh(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	os.RemoveAll(testKeyDB)
	os.Mkdir(testKeyDB, os.ModePerm)
	defer os.RemoveAll(testKeyDB)
	ctx, cancel := context.WithCancel(context.Background())
	// New mock chain manager
	mockChain, err := chainmgr.NewMockChainImplV1(time.Minute, "", 0)
	assert.Empty(t, err)
	assert.NotEmpty(t, mockChain)
	mockChainManager, err := chainmgr.NewMockChainManagerImplV1(mockChain.GetAddr())
	assert.Empty(t, err)
	assert.NotEmpty(t, mockChainManager)
	// Set ID
	id := "1"
	// New payoffer store
	os, err := payofferstore.NewPayOfferStoreImplV1(ctx, testDB)
	assert.Empty(t, err)
	assert.NotEmpty(t, os)
	// New payment manager
	mgr, err := makePaymentManager(ctx, map[uint64]string{0: testPrvKey1}, id, mockChainManager, os, testDB)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr)

	chAddr, err := mgr.CreateOutboundCh(ctx, 0, "target-test", big.NewInt(10), time.Now().Add(1*time.Second).Unix())
	assert.Empty(t, err)

	err = mgr.SettleOutboundCh(ctx, 1, chAddr)
	assert.NotEmpty(t, err)

	err = mgr.SettleOutboundCh(ctx, 0, "testch")
	assert.NotEmpty(t, err)

	err = mgr.SettleOutboundCh(ctx, 0, chAddr)
	assert.Empty(t, err)

	time.Sleep(1 * time.Second)

	chAddr, err = mgr.CreateOutboundCh(ctx, 0, "target-test", big.NewInt(10), time.Now().Add(1*time.Second).Unix())
	assert.Empty(t, err)

	err = mockChainManager.Settle(ctx, []byte{}, chAddr, "")
	assert.Empty(t, err)

	err = mgr.SettleOutboundCh(ctx, 0, chAddr)
	assert.Empty(t, err)

	cancel()
	time.Sleep(1 * time.Second)
}

func TestCollectCh(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	os.RemoveAll(testKeyDB)
	os.Mkdir(testKeyDB, os.ModePerm)
	defer os.RemoveAll(testKeyDB)
	ctx, cancel := context.WithCancel(context.Background())
	// New mock chain manager
	mockChain, err := chainmgr.NewMockChainImplV1(time.Second, "", 0)
	assert.Empty(t, err)
	assert.NotEmpty(t, mockChain)
	mockChainManager, err := chainmgr.NewMockChainManagerImplV1(mockChain.GetAddr())
	assert.Empty(t, err)
	assert.NotEmpty(t, mockChainManager)
	// Set ID
	id := "1"
	// New payoffer store
	os, err := payofferstore.NewPayOfferStoreImplV1(ctx, testDB)
	assert.Empty(t, err)
	assert.NotEmpty(t, os)
	// New payment manager
	mgr, err := makePaymentManager(ctx, map[uint64]string{0: testPrvKey1}, id, mockChainManager, os, testDB)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr)

	chAddr, err := mgr.CreateOutboundCh(ctx, 0, "target-test", big.NewInt(10), time.Now().Add(1*time.Second).Unix())
	assert.Empty(t, err)

	err = mgr.SettleOutboundCh(ctx, 1, chAddr)
	assert.NotEmpty(t, err)

	err = mgr.SettleOutboundCh(ctx, 0, "testch")
	assert.NotEmpty(t, err)

	err = mgr.SettleOutboundCh(ctx, 0, chAddr)
	assert.Empty(t, err)

	err = mgr.CollectOutboundCh(ctx, 1, chAddr)
	assert.NotEmpty(t, err)

	err = mgr.CollectOutboundCh(ctx, 1, "testch")
	assert.NotEmpty(t, err)

	err = mgr.CollectOutboundCh(ctx, 0, chAddr)
	assert.NotEmpty(t, err)

	time.Sleep(4 * time.Second)

	err = mgr.CollectOutboundCh(ctx, 0, chAddr)
	assert.Empty(t, err)

	chAddr, err = mgr.CreateOutboundCh(ctx, 0, "target-test", big.NewInt(10), time.Now().Add(1*time.Second).Unix())
	assert.Empty(t, err)

	err = mockChainManager.Settle(ctx, []byte{}, chAddr, "")
	assert.Empty(t, err)

	err = mgr.CollectOutboundCh(ctx, 0, chAddr)
	assert.NotEmpty(t, err)

	cancel()
	time.Sleep(1 * time.Second)
}

func TestTransactions(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	os.RemoveAll(testDB1)
	os.Mkdir(testDB1, os.ModePerm)
	defer os.RemoveAll(testDB1)
	os.RemoveAll(testDB2)
	os.Mkdir(testDB2, os.ModePerm)
	defer os.RemoveAll(testDB2)
	os.RemoveAll(testDB3)
	os.Mkdir(testDB3, os.ModePerm)
	defer os.RemoveAll(testDB3)
	os.RemoveAll(testDB4)
	os.Mkdir(testDB4, os.ModePerm)
	defer os.RemoveAll(testDB4)
	os.RemoveAll(testDB5)
	os.Mkdir(testDB5, os.ModePerm)
	defer os.RemoveAll(testDB5)
	os.RemoveAll(testKeyDB)
	os.Mkdir(testKeyDB, os.ModePerm)
	defer os.RemoveAll(testKeyDB)
	ctx, cancel := context.WithCancel(context.Background())
	// New mock chain manager
	mockChain, err := chainmgr.NewMockChainImplV1(time.Second, "", 0)
	assert.Empty(t, err)
	assert.NotEmpty(t, mockChain)
	mockChainManager, err := chainmgr.NewMockChainManagerImplV1(mockChain.GetAddr())
	assert.Empty(t, err)
	assert.NotEmpty(t, mockChainManager)
	// New payoffer store
	os, err := payofferstore.NewPayOfferStoreImplV1(ctx, testDB)
	assert.Empty(t, err)
	assert.NotEmpty(t, os)
	// Manager 1
	id := "1"
	mgr1, err := makePaymentManager(ctx, map[uint64]string{0: testPrvKey1, 1: testPrvKey1}, id, mockChainManager, os, testDB1)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr1)
	// Manager 2
	id = "2"
	mgr2, err := makePaymentManager(ctx, map[uint64]string{0: testPrvKey2, 1: testPrvKey2}, id, mockChainManager, os, testDB2)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr2)
	// Manager 3
	id = "3"
	mgr3, err := makePaymentManager(ctx, map[uint64]string{0: testPrvKey3, 1: testPrvKey3}, id, mockChainManager, os, testDB3)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr3)
	// Manager 4
	id = "4"
	mgr4, err := makePaymentManager(ctx, map[uint64]string{0: testPrvKey4, 1: testPrvKey4}, id, mockChainManager, os, testDB4)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr4)
	// Manager 5
	id = "5"
	mgr5, err := makePaymentManager(ctx, map[uint64]string{0: testPrvKey5, 1: testPrvKey5}, id, mockChainManager, os, testDB5)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr5)
	// Obtain address
	addr1, err := mgr1.GetRootAddress(0)
	assert.Empty(t, err)
	addr2, err := mgr2.GetRootAddress(0)
	assert.Empty(t, err)
	addr3, err := mgr3.GetRootAddress(0)
	assert.Empty(t, err)
	addr4, err := mgr4.GetRootAddress(0)
	assert.Empty(t, err)
	addr5, err := mgr5.GetRootAddress(0)
	assert.Empty(t, err)
	// Establish paychs for currency 0
	chAddr, err := mgr1.CreateOutboundCh(ctx, 0, addr3, big.NewInt(10000), time.Now().Add(time.Second).Unix())
	assert.Empty(t, err)
	err = mgr3.AddInboundCh(2, chAddr, time.Now().Add(time.Hour).Unix())
	assert.NotEmpty(t, err)
	err = mgr2.AddInboundCh(0, chAddr, time.Now().Add(time.Hour).Unix())
	assert.NotEmpty(t, err)
	err = mgr3.AddInboundCh(0, "testch", time.Now().Add(time.Hour).Unix())
	assert.NotEmpty(t, err)
	chAddr, err = mgr2.CreateOutboundCh(ctx, 0, addr3, big.NewInt(10000), time.Now().Add(time.Hour).Unix())
	assert.Empty(t, err)
	err = mgr3.AddInboundCh(0, chAddr, time.Now().Add(time.Hour).Unix())
	assert.Empty(t, err)
	chAddr, err = mgr3.CreateOutboundCh(ctx, 0, addr4, big.NewInt(10000), time.Now().Add(time.Hour).Unix())
	assert.Empty(t, err)
	err = mgr4.AddInboundCh(0, chAddr, time.Now().Add(time.Hour).Unix())
	assert.Empty(t, err)
	chAddr, err = mgr4.CreateOutboundCh(ctx, 0, addr5, big.NewInt(10000), time.Now().Add(10*time.Second).Unix())
	assert.Empty(t, err)
	err = mgr5.AddInboundCh(0, chAddr, time.Now().Add(10*time.Second).Unix())
	assert.Empty(t, err)
	// There will be 5 concurrent payment happening at the same time.
	// 1 -> 3 (1 -> 3 for 150 units)
	// 1 -> 3 -> 4 -> 5 (1 -> 5 for 1130 units)
	// 2 -> 3 -> 4 -> 5 (2 -> 5 for 1030 units)
	// 2 -> 3 -> 4 (2 -> 4 for 520 units)
	// 3 -> 4 (3 -> 4 for 3400 units)
	prv3, _ := hex.DecodeString(testPrvKey3)
	prv4, _ := hex.DecodeString(testPrvKey4)
	// Creating offers for 1 -> 3
	// Reserve
	err = mgr1.ReserveForSelf(2, addr3, big.NewInt(150))
	assert.NotEmpty(t, err)

	err = mgr1.ReserveForSelf(0, addr3, big.NewInt(-1))
	assert.NotEmpty(t, err)

	err = mgr1.ReserveForSelf(0, addr3, big.NewInt(100000000))
	assert.NotEmpty(t, err)

	err = mgr1.ReserveForSelf(0, "testaddr", big.NewInt(150))
	assert.NotEmpty(t, err)

	time.Sleep(1 * time.Second)
	err = mgr1.ReserveForSelf(0, addr3, big.NewInt(150))
	assert.NotEmpty(t, err)
	chAddrNew, err := mgr1.CreateOutboundCh(ctx, 0, addr3, big.NewInt(10000), time.Now().Add(time.Second).Unix())
	assert.Empty(t, err)

	// Creating offers for 1 -> 3 -> 4 -> 5
	offer1, err := payoffer.NewPayOffer(prv4, 0, addr5, big.NewInt(10), big.NewInt(100), big.NewInt(1130), time.Minute, time.Now().Add(time.Minute).Unix(), cid.Undef)
	assert.Empty(t, err)
	err = os.AddOffer(offer1)
	assert.Empty(t, err)
	offer2, err := payoffer.NewPayOffer(prv3, 0, addr4, big.NewInt(20), big.NewInt(100), big.NewInt(1130), time.Minute, time.Now().Add(time.Minute).Unix(), offer1.ID())
	assert.Empty(t, err)
	err = os.AddOffer(offer2)
	assert.Empty(t, err)
	// Reserve
	err = mgr4.ReserveForOthers(nil)
	assert.NotEmpty(t, err)

	offerTest, err := payoffer.NewPayOffer(prv4, 0, addr5, big.NewInt(10), big.NewInt(100), big.NewInt(1130), time.Minute, time.Now().Unix()-1, cid.Undef)
	assert.Empty(t, err)
	err = mgr4.ReserveForOthers(offerTest)
	assert.NotEmpty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv4, 0, addr5, big.NewInt(10), big.NewInt(100), big.NewInt(-1), time.Minute, time.Now().Add(time.Minute).Unix(), cid.Undef)
	assert.Empty(t, err)
	err = mgr4.ReserveForOthers(offerTest)
	assert.NotEmpty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv4, 0, addr5, big.NewInt(10), big.NewInt(100), big.NewInt(100000000), time.Minute, time.Now().Add(time.Minute).Unix(), cid.Undef)
	assert.Empty(t, err)
	err = mgr4.ReserveForOthers(offerTest)
	assert.NotEmpty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv4, 2, addr5, big.NewInt(10), big.NewInt(100), big.NewInt(1130), time.Minute, time.Now().Add(time.Minute).Unix(), cid.Undef)
	assert.Empty(t, err)
	err = mgr4.ReserveForOthers(offerTest)
	assert.NotEmpty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv4, 0, addr1, big.NewInt(10), big.NewInt(100), big.NewInt(1130), time.Minute, time.Now().Add(time.Minute).Unix(), cid.Undef)
	assert.Empty(t, err)
	err = mgr4.ReserveForOthers(offerTest)
	assert.NotEmpty(t, err)

	err = mgr4.ReserveForOthers(offer1)
	assert.Empty(t, err)
	err = mgr3.ReserveForOthers(offer2)
	assert.Empty(t, err)
	err = mgr1.ReserveForSelfWithOffer(nil)
	assert.NotEmpty(t, err)
	offerTest, err = payoffer.NewPayOffer(prv3, 0, addr4, big.NewInt(20), big.NewInt(100), big.NewInt(-1), time.Minute, time.Now().Add(time.Minute).Unix(), offer1.ID())
	assert.Empty(t, err)
	err = mgr1.ReserveForSelfWithOffer(offerTest)
	assert.NotEmpty(t, err)
	offerTest, err = payoffer.NewPayOffer(prv3, 0, addr4, big.NewInt(20), big.NewInt(100), big.NewInt(100000000), time.Minute, time.Now().Add(time.Minute).Unix(), offer1.ID())
	assert.Empty(t, err)
	err = mgr1.ReserveForSelfWithOffer(offerTest)
	assert.NotEmpty(t, err)
	offerTest, err = payoffer.NewPayOffer(prv3, 2, addr4, big.NewInt(20), big.NewInt(100), big.NewInt(1130), time.Minute, time.Now().Add(time.Minute).Unix(), offer1.ID())
	assert.Empty(t, err)
	err = mgr1.ReserveForSelfWithOffer(offerTest)
	assert.NotEmpty(t, err)
	offerTest, err = payoffer.NewPayOffer(prv4, 0, addr4, big.NewInt(20), big.NewInt(100), big.NewInt(1130), time.Minute, time.Now().Add(time.Minute).Unix(), offer1.ID())
	assert.Empty(t, err)
	err = mgr1.ReserveForSelfWithOffer(offerTest)
	assert.NotEmpty(t, err)
	offerTest, err = payoffer.NewPayOffer(prv3, 0, addr4, big.NewInt(20), big.NewInt(100), big.NewInt(1130), time.Minute, time.Now().Unix()-1, offer1.ID())
	assert.Empty(t, err)
	err = mgr1.ReserveForSelfWithOffer(offerTest)
	assert.NotEmpty(t, err)

	time.Sleep(1 * time.Second)

	err = mgr1.ReserveForSelfWithOffer(offer2)
	assert.NotEmpty(t, err)

	chAddrNew, err = mgr1.CreateOutboundCh(ctx, 0, addr3, big.NewInt(10000), time.Now().Add(time.Hour).Unix())
	assert.Empty(t, err)
	err = mgr3.AddInboundCh(0, chAddrNew, time.Now().Add(time.Hour).Unix())
	assert.Empty(t, err)
	err = mgr1.ReserveForSelf(0, addr3, big.NewInt(150))
	assert.Empty(t, err)

	err = mgr1.ReserveForSelfWithOffer(offer2)
	assert.Empty(t, err)
	// Creating offers for 2 -> 3 -> 4 -> 5
	offer3, err := payoffer.NewPayOffer(prv4, 0, addr5, big.NewInt(15), big.NewInt(105), big.NewInt(1030), time.Minute, time.Now().Add(time.Minute).Unix(), cid.Undef)
	assert.Empty(t, err)
	err = os.AddOffer(offer3)
	assert.Empty(t, err)
	offer4, err := payoffer.NewPayOffer(prv3, 0, addr4, big.NewInt(25), big.NewInt(95), big.NewInt(1030), time.Minute, time.Now().Add(time.Minute).Unix(), offer3.ID())
	assert.Empty(t, err)
	err = os.AddOffer(offer4)
	assert.Empty(t, err)
	// Reserve
	err = mgr4.ReserveForOthers(offer3)
	assert.Empty(t, err)
	err = mgr3.ReserveForOthers(offer4)
	assert.Empty(t, err)
	err = mgr2.ReserveForSelfWithOffer(offer4)
	assert.Empty(t, err)
	// Creating offers for 2 -> 3 -> 4
	offer5, err := payoffer.NewPayOffer(prv3, 0, addr4, big.NewInt(10), big.NewInt(100), big.NewInt(520), time.Minute, time.Now().Add(time.Minute).Unix(), cid.Undef)
	assert.Empty(t, err)
	err = mgr3.ReserveForOthers(offer5)
	assert.Empty(t, err)
	err = mgr2.ReserveForSelfWithOffer(offer5)
	assert.Empty(t, err)
	// Creating offers for 3 -> 4
	// Reserve
	err = mgr3.ReserveForSelf(0, addr4, big.NewInt(3400))
	assert.Empty(t, err)
	// Start 5 go routines to concurrent doing the payment
	// Routine for 1 -> 3
	routine1 := func(wg *sync.WaitGroup) {
		// It takes 5 seconds in total
		defer wg.Done()
		for i := 0; i < 5; i++ {
			// Pay
			voucher, commit1, _, err := mgr1.PayForSelf(0, addr3, big.NewInt(30))
			assert.Empty(t, err)
			// Receive
			amt, commit2, _, err := mgr3.RecordReceiveForSelf(0, addr1, voucher, addr1)
			assert.Empty(t, err)
			assert.Equal(t, big.NewInt(30), amt)
			commit2()
			commit1()
			time.Sleep(1 * time.Second)
		}
	}
	// Routine for 1 -> 3 -> 4 -> 5
	routine2 := func(wg *sync.WaitGroup) {
		// It takes 3 seconds in total
		defer wg.Done()
		for i := 0; i < 6; i++ {
			// Pay
			var originAmt *big.Int
			if i == 5 {
				originAmt = big.NewInt(30)
			} else {
				originAmt = big.NewInt(220)
			}
			voucher, commit1, _, err := mgr1.PayForSelfWithOffer(originAmt, offer2)
			assert.Empty(t, err)
			// Proxy
			amt, commit2, _, err := mgr3.RecordReceiveForOthers(addr1, voucher, offer2)
			assert.Empty(t, err)
			voucher, commit3, _, err := mgr3.PayForOthers(amt, offer2)
			assert.Empty(t, err)
			amt, commit4, _, err := mgr4.RecordReceiveForOthers(addr3, voucher, offer1)
			assert.Empty(t, err)
			voucher, commit5, _, err := mgr4.PayForOthers(amt, offer1)
			assert.Empty(t, err)
			// Receive
			amt, commit6, _, err := mgr5.RecordReceiveForSelf(0, addr4, voucher, addr1)
			assert.Empty(t, err)
			assert.Equal(t, originAmt, amt)
			commit6()
			commit5()
			commit4()
			commit3()
			commit2()
			commit1()
			time.Sleep(500 * time.Millisecond)
		}
	}
	// Routine 2 -> 3 -> 4 -> 5
	routine3 := func(wg *sync.WaitGroup) {
		// It takes 4 seconds in total
		defer wg.Done()
		for i := 0; i < 206; i++ {
			// Pay
			voucher, commit1, _, err := mgr2.PayForSelfWithOffer(big.NewInt(5), offer4)
			assert.Empty(t, err)
			// Proxy
			amt, commit2, _, err := mgr3.RecordReceiveForOthers(addr2, voucher, offer4)
			assert.Empty(t, err)
			voucher, commit3, _, err := mgr3.PayForOthers(amt, offer4)
			assert.Empty(t, err)
			amt, commit4, _, err := mgr4.RecordReceiveForOthers(addr3, voucher, offer3)
			assert.Empty(t, err)
			voucher, commit5, _, err := mgr4.PayForOthers(amt, offer3)
			assert.Empty(t, err)
			// Receive
			amt, commit6, _, err := mgr5.RecordReceiveForSelf(0, addr4, voucher, addr2)
			assert.Empty(t, err)
			assert.Equal(t, big.NewInt(5), amt)
			commit6()
			commit5()
			commit4()
			commit3()
			commit2()
			commit1()
			time.Sleep(20 * time.Millisecond)
		}
	}
	// Routine 2 -> 3 -> 4
	routine4 := func(wg *sync.WaitGroup) {
		// It takes 2 seconds in total
		defer wg.Done()
		for i := 0; i < 20; i++ {
			// Pay
			voucher, commit1, _, err := mgr2.PayForSelfWithOffer(big.NewInt(26), offer5)
			assert.Empty(t, err)
			// Proxy
			amt, commit2, _, err := mgr3.RecordReceiveForOthers(addr2, voucher, offer5)
			assert.Empty(t, err)
			voucher, commit3, _, err := mgr3.PayForOthers(amt, offer5)
			assert.Empty(t, err)
			// Receive
			amt, commit4, _, err := mgr4.RecordReceiveForSelf(0, addr3, voucher, addr2)
			assert.Empty(t, err)
			assert.Equal(t, big.NewInt(26), amt)
			commit4()
			commit3()
			commit2()
			commit1()
			time.Sleep(100 * time.Millisecond)
		}
	}
	// Routine 3 -> 4
	routine5 := func(wg *sync.WaitGroup) {
		// It takes 2 seconds in total
		defer wg.Done()
		for i := 0; i < 100; i++ {
			// Pay
			voucher, commit1, _, err := mgr3.PayForSelf(0, addr4, big.NewInt(34))
			assert.Empty(t, err)
			// Receive
			amt, commit2, _, err := mgr4.RecordReceiveForSelf(0, addr3, voucher, addr3)
			assert.Empty(t, err)
			assert.Equal(t, big.NewInt(34), amt)
			commit2()
			commit1()
			time.Sleep(20 * time.Millisecond)
		}
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go routine1(&wg)
	wg.Add(1)
	go routine2(&wg)
	wg.Add(1)
	go routine3(&wg)
	wg.Add(1)
	go routine4(&wg)
	wg.Add(1)
	go routine5(&wg)
	// Wait for routine to finish.
	wg.Wait()
	// Verify
	amt, err := mgr5.Receive(0, addr1)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1130), amt)
	amt, err = mgr5.Receive(0, addr2)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1030), amt)
	amt, err = mgr5.Receive(0, addr3)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(0), amt)
	amt, err = mgr4.Receive(0, addr2)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(520), amt)
	amt, err = mgr4.Receive(0, addr3)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(3400), amt)
	_, err = mgr3.Receive(2, addr1)
	assert.NotEmpty(t, err)
	amt, err = mgr3.Receive(0, addr1)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(150), amt)

	// Try to pay more than expected
	data, _ := json.Marshal(voucherJson{
		FromAddr: addr4,
		ChAddr:   chAddr,
		Lane:     0,
		Nonce:    1000,
		Redeemed: big.NewInt(100000).String(),
	})
	_, _, _, err = mgr5.RecordReceiveForSelf(0, addr4, hex.EncodeToString(data), addr4)
	assert.NotEmpty(t, err)

	mockChainManager.Topup(ctx, []byte{0}, chAddr, big.NewInt(1000000))
	_, _, _, err = mgr5.RecordReceiveForSelf(0, addr4, hex.EncodeToString(data), addr4)
	assert.Empty(t, err)

	cancel()
	time.Sleep(3 * time.Second)
}

func TestIncomingChs(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	os.RemoveAll(testDB1)
	os.Mkdir(testDB1, os.ModePerm)
	defer os.RemoveAll(testDB1)
	os.RemoveAll(testDB2)
	os.Mkdir(testDB2, os.ModePerm)
	defer os.RemoveAll(testDB2)
	os.RemoveAll(testKeyDB)
	os.Mkdir(testKeyDB, os.ModePerm)
	defer os.RemoveAll(testKeyDB)
	ctx, cancel := context.WithCancel(context.Background())
	// New mock chain manager
	mockChain, err := chainmgr.NewMockChainImplV1(time.Second, "", 0)
	assert.Empty(t, err)
	assert.NotEmpty(t, mockChain)
	mockChainManager, err := chainmgr.NewMockChainManagerImplV1(mockChain.GetAddr())
	assert.Empty(t, err)
	assert.NotEmpty(t, mockChainManager)
	// New payoffer store
	os, err := payofferstore.NewPayOfferStoreImplV1(ctx, testDB)
	assert.Empty(t, err)
	assert.NotEmpty(t, os)
	// Manager 1
	id := "1"
	mgr1, err := makePaymentManager(ctx, map[uint64]string{0: testPrvKey1, 1: testPrvKey1}, id, mockChainManager, os, testDB1)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr1)
	// Manager 2
	id = "2"
	mgr2, err := makePaymentManager(ctx, map[uint64]string{0: testPrvKey2, 1: testPrvKey2}, id, mockChainManager, os, testDB2)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr2)
	// Obtain address
	addr2, err := mgr2.GetRootAddress(0)
	assert.Empty(t, err)

	// Establish channels
	chAddr, err := mgr1.CreateOutboundCh(ctx, 0, addr2, big.NewInt(10000), time.Now().Add(time.Second).Unix())
	assert.Empty(t, err)
	err = mgr2.AddInboundCh(0, chAddr, time.Now().Add(time.Second).Unix())
	assert.Empty(t, err)

	res, err := mgr2.ListInboundChs(ctx)
	assert.Empty(t, err)
	assert.Equal(t, 1, len(res[0]))

	_, _, _, _, _, _, _, err = mgr2.InspectInboundCh(ctx, 2, chAddr)
	assert.NotEmpty(t, err)

	_, _, _, _, _, _, _, err = mgr2.InspectInboundCh(ctx, 0, "testch")
	assert.NotEmpty(t, err)

	err = mgr1.TopupOutboundCh(ctx, 0, addr2, big.NewInt(10))
	assert.Empty(t, err)

	_, _, _, _, active, _, _, err := mgr2.InspectInboundCh(ctx, 0, chAddr)
	assert.Empty(t, err)
	assert.True(t, active)

	time.Sleep(1 * time.Second)

	_, _, _, _, active, _, _, err = mgr2.InspectInboundCh(ctx, 0, chAddr)
	assert.Empty(t, err)
	assert.False(t, active)

	err = mockChainManager.Settle(ctx, []byte{}, chAddr, "")
	assert.Empty(t, err)

	_, _, _, _, _, _, settlingAt, err := mgr2.InspectInboundCh(ctx, 0, chAddr)
	assert.Empty(t, err)
	assert.NotEqual(t, 0, settlingAt)

	err = mgr2.SettleInboundCh(ctx, 0, chAddr)
	assert.Empty(t, err)

	chAddr, err = mgr1.CreateOutboundCh(ctx, 0, addr2, big.NewInt(10000), time.Now().Add(time.Second).Unix())
	assert.Empty(t, err)
	err = mgr2.AddInboundCh(0, chAddr, time.Now().Add(time.Second).Unix())
	assert.Empty(t, err)

	err = mgr2.SettleInboundCh(ctx, 2, chAddr)
	assert.NotEmpty(t, err)

	err = mgr2.SettleInboundCh(ctx, 0, "testch")
	assert.NotEmpty(t, err)

	err = mgr2.SettleInboundCh(ctx, 0, chAddr)
	assert.NotEmpty(t, err)

	time.Sleep(1 * time.Second)

	err = mgr2.SettleInboundCh(ctx, 0, chAddr)
	assert.Empty(t, err)

	err = mgr2.CollectInboundCh(ctx, 2, chAddr)
	assert.NotEmpty(t, err)

	err = mgr2.CollectInboundCh(ctx, 0, "testch")
	assert.NotEmpty(t, err)

	err = mgr2.CollectInboundCh(ctx, 0, chAddr)
	assert.NotEmpty(t, err)

	time.Sleep(5 * time.Second)

	err = mgr2.CollectInboundCh(ctx, 0, chAddr)
	assert.Empty(t, err)

	chAddr, err = mgr1.CreateOutboundCh(ctx, 0, addr2, big.NewInt(10000), time.Now().Add(time.Second).Unix())
	assert.Empty(t, err)

	err = mockChainManager.Settle(ctx, []byte{}, chAddr, "")
	assert.Empty(t, err)

	err = mgr2.AddInboundCh(0, chAddr, time.Now().Add(time.Second).Unix())
	assert.NotEmpty(t, err)

	err = mgr1.SettleOutboundCh(ctx, 0, chAddr)
	assert.Empty(t, err)

	chAddr, err = mgr1.CreateOutboundCh(ctx, 0, addr2, big.NewInt(10000), time.Now().Add(time.Second).Unix())
	assert.Empty(t, err)

	err = mgr2.AddInboundCh(0, chAddr, time.Now().Add(time.Second).Unix())
	assert.Empty(t, err)

	err = mockChainManager.Settle(ctx, []byte{}, chAddr, "")
	assert.Empty(t, err)

	err = mgr2.CollectInboundCh(ctx, 0, chAddr)
	assert.NotEmpty(t, err)

	time.Sleep(5 * time.Second)

	err = mgr2.CollectInboundCh(ctx, 0, chAddr)
	assert.NotEmpty(t, err)

	_, _, _, _, _, _, _, err = mgr2.InspectInboundCh(ctx, 0, chAddr)
	assert.Empty(t, err)

	err = mgr2.CollectInboundCh(ctx, 0, chAddr)
	assert.Empty(t, err)

	cancel()
	time.Sleep(1 * time.Second)
}

func TestPaymentFailure(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	os.RemoveAll(testDB1)
	os.Mkdir(testDB1, os.ModePerm)
	defer os.RemoveAll(testDB1)
	os.RemoveAll(testDB2)
	os.Mkdir(testDB2, os.ModePerm)
	defer os.RemoveAll(testDB2)
	os.RemoveAll(testDB3)
	os.Mkdir(testDB3, os.ModePerm)
	defer os.RemoveAll(testDB3)
	os.RemoveAll(testKeyDB)
	os.Mkdir(testKeyDB, os.ModePerm)
	defer os.RemoveAll(testKeyDB)
	ctx, cancel := context.WithCancel(context.Background())
	// New mock chain manager
	mockChain, err := chainmgr.NewMockChainImplV1(time.Second, "", 0)
	assert.Empty(t, err)
	assert.NotEmpty(t, mockChain)
	mockChainManager, err := chainmgr.NewMockChainManagerImplV1(mockChain.GetAddr())
	assert.Empty(t, err)
	assert.NotEmpty(t, mockChainManager)
	// New payoffer store
	os, err := payofferstore.NewPayOfferStoreImplV1(ctx, testDB)
	assert.Empty(t, err)
	assert.NotEmpty(t, os)
	// Manager 1
	id := "1"
	mgr1, err := makePaymentManager(ctx, map[uint64]string{0: testPrvKey1, 1: testPrvKey1}, id, mockChainManager, os, testDB1)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr1)
	// Manager 2
	id = "2"
	mgr2, err := makePaymentManager(ctx, map[uint64]string{0: testPrvKey2, 1: testPrvKey2}, id, mockChainManager, os, testDB2)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr2)
	// Manager 3
	id = "3"
	mgr3, err := makePaymentManager(ctx, map[uint64]string{0: testPrvKey3, 1: testPrvKey3}, id, mockChainManager, os, testDB3)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr3)
	// Keys
	prv1, _ := hex.DecodeString(testPrvKey1)
	prv2, _ := hex.DecodeString(testPrvKey2)
	prv3, _ := hex.DecodeString(testPrvKey3)
	// Obtain address
	addr1, err := mgr1.GetRootAddress(0)
	assert.Empty(t, err)
	addr2, err := mgr2.GetRootAddress(0)
	assert.Empty(t, err)
	addr3, err := mgr3.GetRootAddress(0)
	assert.Empty(t, err)
	// Establish channels
	chAddr, err := mgr1.CreateOutboundCh(ctx, 0, addr2, big.NewInt(10000), time.Now().Add(time.Second).Unix())
	assert.Empty(t, err)
	chAddr, err = mgr2.CreateOutboundCh(ctx, 0, addr3, big.NewInt(10000), time.Now().Add(time.Second).Unix())
	assert.Empty(t, err)
	_, _, _, err = mgr1.PayForSelf(0, addr2, big.NewInt(-1))
	assert.NotEmpty(t, err)
	_, _, _, err = mgr1.PayForSelf(2, addr2, big.NewInt(1))
	assert.NotEmpty(t, err)
	_, _, _, err = mgr1.PayForSelf(0, addr3, big.NewInt(1))
	assert.NotEmpty(t, err)
	_, _, _, err = mgr1.PayForSelf(0, addr2, big.NewInt(1))
	assert.NotEmpty(t, err)
	err = mgr1.ReserveForSelf(0, addr2, big.NewInt(1))
	assert.Empty(t, err)
	_, _, _, err = mgr1.PayForSelf(0, addr2, big.NewInt(100000))
	assert.NotEmpty(t, err)

	_, _, _, err = mgr1.PayForSelfWithOffer(big.NewInt(0), nil)
	assert.NotEmpty(t, err)

	_, _, _, err = mgr1.PayForSelfWithOffer(big.NewInt(1), nil)
	assert.NotEmpty(t, err)

	offerTest, err := payoffer.NewPayOffer(prv2, 0, addr3, big.NewInt(10), big.NewInt(1000), big.NewInt(20), time.Minute, time.Now().Unix()-1, cid.Undef)
	assert.Empty(t, err)

	_, _, _, err = mgr1.PayForSelfWithOffer(big.NewInt(1), offerTest)
	assert.NotEmpty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv2, 2, addr3, big.NewInt(10), big.NewInt(1000), big.NewInt(20), time.Minute, time.Now().Unix()+1, cid.Undef)
	assert.Empty(t, err)

	_, _, _, err = mgr1.PayForSelfWithOffer(big.NewInt(1), offerTest)
	assert.NotEmpty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv3, 0, addr3, big.NewInt(10), big.NewInt(1000), big.NewInt(20), time.Minute, time.Now().Unix()+1, cid.Undef)
	assert.Empty(t, err)

	_, _, _, err = mgr1.PayForSelfWithOffer(big.NewInt(1), offerTest)
	assert.NotEmpty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv2, 0, addr3, big.NewInt(10), big.NewInt(1000), big.NewInt(20), time.Minute, time.Now().Unix()+1, cid.Undef)
	assert.Empty(t, err)

	_, _, _, err = mgr1.PayForSelfWithOffer(big.NewInt(10000), offerTest)
	assert.NotEmpty(t, err)

	err = mgr1.ReserveForSelfWithOffer(offerTest)
	assert.Empty(t, err)

	_, _, _, err = mgr1.PayForSelfWithOffer(big.NewInt(10000), offerTest)
	assert.NotEmpty(t, err)

	time.Sleep(1 * time.Second)
	_, _, _, err = mgr1.PayForSelf(0, addr2, big.NewInt(1))
	assert.NotEmpty(t, err)

	chAddr, err = mgr1.CreateOutboundCh(ctx, 0, addr2, big.NewInt(10000), time.Now().Unix())
	assert.Empty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv2, 0, addr3, big.NewInt(10), big.NewInt(1000), big.NewInt(20), time.Minute, time.Now().Unix()+1, cid.Undef)
	assert.Empty(t, err)

	_, _, _, err = mgr1.PayForSelfWithOffer(big.NewInt(1), offerTest)
	assert.NotEmpty(t, err)

	chAddr, err = mgr1.CreateOutboundCh(ctx, 0, addr2, big.NewInt(10000), time.Now().Unix())
	assert.Empty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv2, 0, addr3, big.NewInt(10), big.NewInt(1000), big.NewInt(20), time.Minute, time.Now().Unix()+1, cid.Undef)
	assert.Empty(t, err)

	err = mgr1.ReserveForOthers(offerTest)
	assert.NotEmpty(t, err)

	chAddr, err = mgr1.CreateOutboundCh(ctx, 0, addr2, big.NewInt(10000), time.Now().Unix())
	assert.Empty(t, err)

	_, _, _, err = mgr1.PayForSelfWithOffer(big.NewInt(1), offerTest)
	assert.NotEmpty(t, err)

	_, _, _, err = mgr1.PayForOthers(big.NewInt(-1), nil)
	assert.NotEmpty(t, err)

	_, _, _, err = mgr1.PayForOthers(big.NewInt(1), nil)
	assert.NotEmpty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv2, 0, addr3, big.NewInt(10), big.NewInt(1000), big.NewInt(20), time.Minute, time.Now().Unix()-1, cid.Undef)
	assert.Empty(t, err)

	_, _, _, err = mgr1.PayForOthers(big.NewInt(1), offerTest)
	assert.NotEmpty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv2, 3, addr3, big.NewInt(10), big.NewInt(1000), big.NewInt(20), time.Minute, time.Now().Unix()+1, cid.Undef)
	assert.Empty(t, err)

	_, _, _, err = mgr1.PayForOthers(big.NewInt(1), offerTest)
	assert.NotEmpty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv3, 0, addr3, big.NewInt(10), big.NewInt(1000), big.NewInt(20), time.Minute, time.Now().Unix()+1, cid.Undef)
	assert.Empty(t, err)

	_, _, _, err = mgr1.PayForOthers(big.NewInt(1), offerTest)
	assert.NotEmpty(t, err)

	chAddr, err = mgr1.CreateOutboundCh(ctx, 0, addr2, big.NewInt(10000), time.Now().Unix())
	assert.Empty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv1, 0, addr2, big.NewInt(10), big.NewInt(1000), big.NewInt(20), time.Minute, time.Now().Unix()+1, cid.Undef)
	assert.Empty(t, err)

	_, _, _, err = mgr1.PayForOthers(big.NewInt(1), offerTest)
	assert.NotEmpty(t, err)

	chAddr, err = mgr1.CreateOutboundCh(ctx, 0, addr2, big.NewInt(10000), time.Now().Unix())
	assert.Empty(t, err)

	err = mgr1.ReserveForOthers(offerTest)
	assert.NotEmpty(t, err)

	chAddr, err = mgr1.CreateOutboundCh(ctx, 0, addr2, big.NewInt(10000), time.Now().Unix()+1)
	assert.Empty(t, err)

	err = mgr1.ReserveForOthers(offerTest)
	assert.Empty(t, err)

	_, _, _, err = mgr1.PayForOthers(big.NewInt(100000), offerTest)
	assert.NotEmpty(t, err)

	time.Sleep(1 * time.Second)

	chAddr, err = mgr1.CreateOutboundCh(ctx, 0, addr2, big.NewInt(10000), time.Now().Unix()+1)
	assert.Empty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv1, 0, addr2, big.NewInt(10), big.NewInt(1000), big.NewInt(20), time.Minute, time.Now().Unix()+1, cid.Undef)
	assert.Empty(t, err)

	_, _, _, err = mgr1.PayForOthers(big.NewInt(1), offerTest)
	assert.NotEmpty(t, err)

	time.Sleep(1 * time.Second)

	_, _, _, err = mgr1.RecordReceiveForSelf(3, addr2, "", addr2)
	assert.NotEmpty(t, err)

	_, _, _, err = mgr1.RecordReceiveForSelf(0, addr2, "", addr2)
	assert.NotEmpty(t, err)

	chAddr, err = mgr2.CreateOutboundCh(ctx, 0, addr1, big.NewInt(1000), time.Now().Unix())
	assert.Empty(t, err)

	err = mgr1.AddInboundCh(0, chAddr, time.Now().Unix())
	_, _, _, err = mgr1.RecordReceiveForSelf(0, addr2, "", addr2)
	assert.NotEmpty(t, err)

	chAddr, err = mgr2.CreateOutboundCh(ctx, 0, addr1, big.NewInt(1000), time.Now().Unix()+1)
	assert.Empty(t, err)

	err = mgr1.AddInboundCh(0, chAddr, time.Now().Unix()+1)
	_, _, _, err = mgr1.RecordReceiveForSelf(0, addr2, "invalid", addr2)
	assert.NotEmpty(t, err)

	data, _ := json.Marshal(voucherJson{
		FromAddr: addr2,
		ChAddr:   chAddr,
		Lane:     1,
		Nonce:    1,
		Redeemed: big.NewInt(10000).String(),
	})

	_, _, _, err = mgr1.RecordReceiveForSelf(0, addr2, hex.EncodeToString(data), addr2)
	assert.NotEmpty(t, err)

	data, _ = json.Marshal(voucherJson{
		FromAddr: addr2,
		ChAddr:   chAddr,
		Lane:     0,
		Nonce:    1,
		Redeemed: big.NewInt(10000).String(),
	})

	_, _, _, err = mgr1.RecordReceiveForSelf(0, addr2, hex.EncodeToString(data), addr2)
	assert.NotEmpty(t, err)

	err = mgr2.TopupOutboundCh(ctx, 0, addr1, big.NewInt(1))
	assert.Empty(t, err)

	data, _ = json.Marshal(voucherJson{
		FromAddr: addr2,
		ChAddr:   chAddr,
		Lane:     0,
		Nonce:    1,
		Redeemed: big.NewInt(10000).String(),
	})

	_, _, _, err = mgr1.RecordReceiveForSelf(0, addr2, hex.EncodeToString(data), addr2)
	assert.NotEmpty(t, err)

	data, _ = json.Marshal(voucherJson{
		FromAddr: addr2,
		ChAddr:   chAddr,
		Lane:     0,
		Nonce:    1,
		Redeemed: big.NewInt(0).String(),
	})

	_, _, _, err = mgr1.RecordReceiveForSelf(0, addr2, hex.EncodeToString(data), addr2)
	assert.NotEmpty(t, err)

	_, _, _, err = mgr1.RecordReceiveForOthers(addr3, "", nil)
	assert.NotEmpty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv1, 0, addr2, big.NewInt(10), big.NewInt(1000), big.NewInt(20), time.Minute, time.Now().Unix()-1, cid.Undef)
	assert.Empty(t, err)

	_, _, _, err = mgr1.RecordReceiveForOthers(addr3, "", offerTest)
	assert.NotEmpty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv1, 3, addr2, big.NewInt(10), big.NewInt(1000), big.NewInt(20), time.Minute, time.Now().Unix()+1, cid.Undef)
	assert.Empty(t, err)

	_, _, _, err = mgr1.RecordReceiveForOthers(addr3, "", offerTest)
	assert.NotEmpty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv1, 0, addr2, big.NewInt(10), big.NewInt(1000), big.NewInt(20), time.Minute, time.Now().Unix()+1, cid.Undef)
	assert.Empty(t, err)

	_, _, _, err = mgr1.RecordReceiveForOthers(addr3, "", offerTest)
	assert.NotEmpty(t, err)

	chAddr, err = mgr3.CreateOutboundCh(ctx, 0, addr1, big.NewInt(1000), time.Now().Unix())
	assert.Empty(t, err)
	err = mgr1.AddInboundCh(0, chAddr, time.Now().Unix())
	assert.Empty(t, err)

	_, _, _, err = mgr1.RecordReceiveForOthers(addr3, "", offerTest)
	assert.NotEmpty(t, err)

	chAddr, err = mgr3.CreateOutboundCh(ctx, 0, addr1, big.NewInt(1000), time.Now().Unix()+1)
	assert.Empty(t, err)
	err = mgr1.AddInboundCh(0, chAddr, time.Now().Unix()+1)
	assert.Empty(t, err)

	_, _, _, err = mgr1.RecordReceiveForOthers(addr3, "invalid", offerTest)
	assert.NotEmpty(t, err)

	data, _ = json.Marshal(voucherJson{
		FromAddr: addr3,
		ChAddr:   chAddr,
		Lane:     1,
		Nonce:    1,
		Redeemed: big.NewInt(100).String(),
	})
	_, _, _, err = mgr1.RecordReceiveForOthers(addr3, hex.EncodeToString(data), offerTest)
	assert.NotEmpty(t, err)

	data, _ = json.Marshal(voucherJson{
		FromAddr: addr3,
		ChAddr:   chAddr,
		Lane:     0,
		Nonce:    1,
		Redeemed: big.NewInt(10000).String(),
	})
	_, _, _, err = mgr1.RecordReceiveForOthers(addr3, hex.EncodeToString(data), offerTest)
	assert.NotEmpty(t, err)

	err = mgr3.TopupOutboundCh(ctx, 0, addr1, big.NewInt(1))
	assert.Empty(t, err)

	data, _ = json.Marshal(voucherJson{
		FromAddr: addr3,
		ChAddr:   chAddr,
		Lane:     0,
		Nonce:    1,
		Redeemed: big.NewInt(10000).String(),
	})
	_, _, _, err = mgr1.RecordReceiveForOthers(addr3, hex.EncodeToString(data), offerTest)
	assert.NotEmpty(t, err)

	data, _ = json.Marshal(voucherJson{
		FromAddr: addr3,
		ChAddr:   chAddr,
		Lane:     0,
		Nonce:    1,
		Redeemed: big.NewInt(0).String(),
	})
	_, _, _, err = mgr1.RecordReceiveForOthers(addr3, hex.EncodeToString(data), offerTest)
	assert.NotEmpty(t, err)

	cancel()
	time.Sleep(1 * time.Second)
}

func TestInactivity(t *testing.T) {
	os.RemoveAll(testDB)
	os.Mkdir(testDB, os.ModePerm)
	defer os.RemoveAll(testDB)
	os.RemoveAll(testDB1)
	os.Mkdir(testDB1, os.ModePerm)
	defer os.RemoveAll(testDB1)
	os.RemoveAll(testDB2)
	os.Mkdir(testDB2, os.ModePerm)
	defer os.RemoveAll(testDB2)
	os.RemoveAll(testDB3)
	os.Mkdir(testDB3, os.ModePerm)
	defer os.RemoveAll(testDB3)
	os.RemoveAll(testKeyDB)
	os.Mkdir(testKeyDB, os.ModePerm)
	defer os.RemoveAll(testKeyDB)
	ctx, cancel := context.WithCancel(context.Background())
	// New mock chain manager
	mockChain, err := chainmgr.NewMockChainImplV1(time.Second, "", 0)
	assert.Empty(t, err)
	assert.NotEmpty(t, mockChain)
	mockChainManager, err := chainmgr.NewMockChainManagerImplV1(mockChain.GetAddr())
	assert.Empty(t, err)
	assert.NotEmpty(t, mockChainManager)
	// New payoffer store
	os, err := payofferstore.NewPayOfferStoreImplV1(ctx, testDB)
	assert.Empty(t, err)
	assert.NotEmpty(t, os)
	// Manager 1
	id := "1"
	mgr1, err := makePaymentManager(ctx, map[uint64]string{0: testPrvKey1, 1: testPrvKey1}, id, mockChainManager, os, testDB1)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr1)
	// Manager 2
	id = "2"
	mgr2, err := makePaymentManager(ctx, map[uint64]string{0: testPrvKey2, 1: testPrvKey2}, id, mockChainManager, os, testDB2)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr2)
	// Manager 3
	id = "3"
	mgr3, err := makePaymentManager(ctx, map[uint64]string{0: testPrvKey3, 1: testPrvKey3}, id, mockChainManager, os, testDB3)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr3)
	// Keys
	// prv1, _ := hex.DecodeString(testPrvKey1)
	prv2, _ := hex.DecodeString(testPrvKey2)
	// prv3, _ := hex.DecodeString(testPrvKey3)
	// Obtain address
	addr1, err := mgr1.GetRootAddress(0)
	assert.Empty(t, err)
	addr2, err := mgr2.GetRootAddress(0)
	assert.Empty(t, err)
	addr3, err := mgr3.GetRootAddress(0)
	assert.Empty(t, err)

	chAddr, err := mgr1.CreateOutboundCh(ctx, 0, addr2, big.NewInt(10000), time.Now().Add(time.Minute).Unix())
	assert.Empty(t, err)
	err = mgr2.AddInboundCh(0, chAddr, time.Now().Add(time.Minute).Unix())
	assert.Empty(t, err)
	chAddr, err = mgr2.CreateOutboundCh(ctx, 0, addr3, big.NewInt(10000), time.Now().Add(time.Minute).Unix())
	assert.Empty(t, err)
	err = mgr3.AddInboundCh(0, chAddr, time.Now().Add(time.Minute).Unix())
	assert.Empty(t, err)
	// Next, ask mgr2 to reserve for mgr1
	offerTest, err := payoffer.NewPayOffer(prv2, 0, addr3, big.NewInt(10), big.NewInt(1000), big.NewInt(2000), time.Second, time.Now().Add(time.Minute).Unix(), cid.Undef)
	assert.Empty(t, err)
	err = mgr2.ReserveForOthers(offerTest)
	assert.Empty(t, err)
	err = mgr1.ReserveForSelfWithOffer(offerTest)
	assert.Empty(t, err)
	time.Sleep(2 * time.Second)
	offerTest, err = payoffer.NewPayOffer(prv2, 0, addr3, big.NewInt(10), big.NewInt(1000), big.NewInt(20000), time.Second, time.Now().Add(time.Minute).Unix(), cid.Undef)
	assert.Empty(t, err)
	err = mgr2.ReserveForOthers(offerTest)
	assert.NotEmpty(t, err)
	err = mgr1.ReserveForSelfWithOffer(offerTest)
	assert.NotEmpty(t, err)
	offerTest, err = payoffer.NewPayOffer(prv2, 0, addr3, big.NewInt(10), big.NewInt(1000), big.NewInt(2000), time.Second, time.Now().Add(time.Minute).Unix(), cid.Undef)
	assert.Empty(t, err)
	err = mgr1.ReserveForSelfWithOffer(offerTest)
	assert.Empty(t, err)
	time.Sleep(2 * time.Second)
	err = mgr1.ReserveForSelf(0, addr2, big.NewInt(100000))
	assert.NotEmpty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv2, 0, addr3, big.NewInt(10), big.NewInt(1000), big.NewInt(2000), time.Second, time.Now().Add(time.Minute).Unix(), cid.Undef)
	assert.Empty(t, err)
	err = mgr2.ReserveForOthers(offerTest)
	assert.Empty(t, err)
	err = mgr1.ReserveForSelfWithOffer(offerTest)
	assert.Empty(t, err)
	voucher, commit1, _, err := mgr1.PayForSelfWithOffer(big.NewInt(1), offerTest)
	assert.Empty(t, err)
	_, commit2, _, err := mgr2.RecordReceiveForOthers(addr1, voucher, offerTest)
	assert.Empty(t, err)
	commit2()
	commit1()
	voucher, _, revert, err := mgr1.PayForSelfWithOffer(big.NewInt(1), offerTest)
	assert.Empty(t, err)
	time.Sleep(2 * time.Second)
	_, _, _, err = mgr2.RecordReceiveForOthers(addr1, voucher, offerTest)
	assert.NotEmpty(t, err)
	revert()
	_, _, _, err = mgr1.PayForSelfWithOffer(big.NewInt(1), offerTest)
	assert.NotEmpty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv2, 0, addr3, big.NewInt(10), big.NewInt(1000), big.NewInt(2000), time.Second, time.Now().Add(time.Minute).Unix(), cid.Undef)
	assert.Empty(t, err)
	err = mgr1.ReserveForSelfWithOffer(offerTest)
	assert.Empty(t, err)
	err = mgr1.ReserveForSelf(0, addr2, big.NewInt(100))
	assert.Empty(t, err)
	voucher, commit1, _, err = mgr1.PayForSelfWithOffer(big.NewInt(1), offerTest)
	assert.Empty(t, err)
	assert.Empty(t, err)
	_, commit2, _, err = mgr2.RecordReceiveForOthers(addr1, voucher, offerTest)
	assert.Empty(t, err)
	commit2()
	commit1()
	time.Sleep(2 * time.Second)
	_, _, _, err = mgr2.RecordReceiveForSelf(0, addr1, "invalid", addr1)
	assert.NotEmpty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv2, 0, addr3, big.NewInt(10), big.NewInt(1000), big.NewInt(2000), time.Second, time.Now().Add(time.Minute).Unix(), cid.Undef)
	assert.Empty(t, err)
	err = mgr1.ReserveForSelfWithOffer(offerTest)
	assert.Empty(t, err)
	err = mgr1.ReserveForSelf(0, addr2, big.NewInt(100))
	assert.Empty(t, err)
	voucher, commit1, _, err = mgr1.PayForSelfWithOffer(big.NewInt(1), offerTest)
	assert.Empty(t, err)
	assert.Empty(t, err)
	_, commit2, _, err = mgr2.RecordReceiveForOthers(addr1, voucher, offerTest)
	assert.Empty(t, err)
	commit2()
	commit1()
	time.Sleep(2 * time.Second)

	data, _ := json.Marshal(voucherJson{
		FromAddr: addr1,
		ChAddr:   chAddr,
		Lane:     1,
		Nonce:    1,
		Redeemed: big.NewInt(10000).String(),
	})
	_, _, _, err = mgr2.RecordReceiveForSelf(0, addr1, hex.EncodeToString(data), addr1)
	assert.NotEmpty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv2, 0, addr3, big.NewInt(10), big.NewInt(1000), big.NewInt(2000), time.Second, time.Now().Add(time.Minute).Unix(), cid.Undef)
	assert.Empty(t, err)
	err = mgr1.ReserveForSelfWithOffer(offerTest)
	assert.Empty(t, err)
	err = mgr1.ReserveForSelf(0, addr2, big.NewInt(100))
	assert.Empty(t, err)
	voucher, commit1, _, err = mgr1.PayForSelfWithOffer(big.NewInt(1), offerTest)
	assert.Empty(t, err)
	assert.Empty(t, err)
	_, commit2, _, err = mgr2.RecordReceiveForOthers(addr1, voucher, offerTest)
	assert.Empty(t, err)
	commit2()
	commit1()
	time.Sleep(2 * time.Second)

	data, _ = json.Marshal(voucherJson{
		FromAddr: addr1,
		ChAddr:   chAddr,
		Lane:     0,
		Nonce:    1,
		Redeemed: big.NewInt(10000).String(),
	})
	_, _, _, err = mgr2.RecordReceiveForSelf(0, addr1, hex.EncodeToString(data), addr1)
	assert.NotEmpty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv2, 0, addr3, big.NewInt(10), big.NewInt(1000), big.NewInt(2000), time.Second, time.Now().Add(time.Minute).Unix(), cid.Undef)
	assert.Empty(t, err)
	err = mgr1.ReserveForSelfWithOffer(offerTest)
	assert.Empty(t, err)
	err = mgr1.ReserveForSelf(0, addr2, big.NewInt(100))
	assert.Empty(t, err)
	voucher, commit1, _, err = mgr1.PayForSelfWithOffer(big.NewInt(1), offerTest)
	assert.Empty(t, err)
	assert.Empty(t, err)
	_, commit2, _, err = mgr2.RecordReceiveForOthers(addr1, voucher, offerTest)
	assert.Empty(t, err)
	commit2()
	commit1()
	time.Sleep(2 * time.Second)

	data, _ = json.Marshal(voucherJson{
		FromAddr: addr1,
		ChAddr:   chAddr,
		Lane:     0,
		Nonce:    1,
		Redeemed: big.NewInt(0).String(),
	})
	_, _, _, err = mgr2.RecordReceiveForSelf(0, addr1, hex.EncodeToString(data), addr1)
	assert.NotEmpty(t, err)

	offerTest, err = payoffer.NewPayOffer(prv2, 0, addr3, big.NewInt(10), big.NewInt(1000), big.NewInt(2000), time.Second, time.Now().Add(time.Minute).Unix(), cid.Undef)
	assert.Empty(t, err)
	err = mgr1.ReserveForSelfWithOffer(offerTest)
	assert.Empty(t, err)
	err = mgr1.ReserveForSelf(0, addr2, big.NewInt(100))
	assert.Empty(t, err)
	voucher, commit1, _, err = mgr1.PayForSelfWithOffer(big.NewInt(1), offerTest)
	assert.Empty(t, err)
	assert.Empty(t, err)
	_, commit2, _, err = mgr2.RecordReceiveForOthers(addr1, voucher, offerTest)
	assert.Empty(t, err)
	commit2()
	commit1()
	time.Sleep(2 * time.Second)

	data, _ = json.Marshal(voucherJson{
		FromAddr: addr1,
		ChAddr:   chAddr,
		Lane:     0,
		Nonce:    1,
		Redeemed: big.NewInt(10000).String(),
	})
	err = mgr1.TopupOutboundCh(ctx, 0, addr2, big.NewInt(1))
	assert.Empty(t, err)
	_, _, _, err = mgr2.RecordReceiveForSelf(0, addr1, hex.EncodeToString(data), addr1)
	assert.NotEmpty(t, err)

	cancel()
	time.Sleep(3 * time.Second)

	ctx, cancel = context.WithCancel(context.Background())
	id = "6"
	mgr1, err = makePaymentManager(ctx, map[uint64]string{0: testPrvKey1, 1: testPrvKey1}, id, mockChainManager, os, testDB1)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr1)

	id = "7"
	mgr2, err = makePaymentManager(ctx, map[uint64]string{0: testPrvKey2, 1: testPrvKey2}, id, mockChainManager, os, testDB2)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr1)
	time.Sleep(1 * time.Second)

	cancel()
	time.Sleep(3 * time.Second)
}

func makePaymentManager(ctx context.Context, keys map[uint64]string, id string, mc chainmgr.ChainManager, os payofferstore.PayOfferStore, db string) (PaymentManager, error) {
	curIDs := make([]uint64, 0)
	cms := make(map[uint64]chainmgr.ChainManager)
	// New keystore
	dsopts := badgerds.DefaultOptions
	dsopts.SyncWrites = false
	dsopts.Truncate = true
	ks, err := badgerds.NewDatastore(filepath.Join(testKeyDB, id), &dsopts)
	if err != nil {
		return nil, err
	}
	for curID, prvKey := range keys {
		prv, err := hex.DecodeString(prvKey)
		if err != nil {
			return nil, err
		}
		err = ks.Put(datastore.NewKey(fmt.Sprintf("%v", curID)), prv)
		if err != nil {
			return nil, err
		}
		curIDs = append(curIDs, curID)
		cms[curID] = mc
	}
	return NewPaymentManagerImplV1(ctx, db, curIDs, cms, ks, os)
}

// voucherJson is the voucher.
type voucherJson struct {
	FromAddr string `json:"from_addr"`
	ChAddr   string `json:"chAddr"`
	Lane     uint64 `json:"lane"`
	Nonce    uint64 `json:"nonce"`
	Redeemed string `json:"redeemed"`
}
