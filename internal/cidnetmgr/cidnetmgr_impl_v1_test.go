package cidnetmgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"bytes"
	"context"
	"crypto/rand"
	"io/ioutil"
	"math/big"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	routing "github.com/libp2p/go-libp2p-routing"
	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/fcr/internal/config"
	"github.com/wcgcyx/fcr/internal/paymgr/chainmgr"
	"github.com/wcgcyx/fcr/internal/paynetmgr"
	"github.com/wcgcyx/fcr/internal/payoffer"
)

const (
	testDB0     = "./testdb0"
	testDB1     = "./testdb1"
	testDB2     = "./testdb2"
	testDB3     = "./testdb3"
	testDB4     = "./testdb4"
	testDB5     = "./testdb5"
	testDB6     = "./testdb6"
	testDB7     = "./testdb7"
	testDB8     = "./testdb8"
	testDB9     = "./testdb9"
	testFile0   = "./testfile0"
	testFile1   = "./testfile1"
	testFile2   = "./testfile2"
	testFile3   = "./testfile3"
	testFile4   = "./testfile4"
	testFile5   = "./testfile5"
	testFile6   = "./testfile6"
	testFile7   = "./testfile7"
	testFile8   = "./testfile8"
	testFile9   = "./testfile9"
	testOut0    = "./testout0"
	testOut1    = "./testout1"
	testOut2    = "./testout2"
	testOut3    = "./testout3"
	testOut4    = "./testout4"
	testOut5    = "./testout5"
	testOut6    = "./testout6"
	testOut7    = "./testout7"
	testOut8    = "./testout8"
	testOut9    = "./testout9"
	testOut10   = "./testout10"
	testOut11   = "./testout11"
	testOut12   = "./testout12"
	testOut13   = "./testout13"
	testOut14   = "./testout14"
	testOut15   = "./testout15"
	testOut16   = "./testout16"
	testOut17   = "./testout17"
	testOut18   = "./testout18"
	testOut19   = "./testout19"
	testPrvKey0 = "D8qOgeJEISjwbiwnyfQ5UZzxVZZrVIZMgfs71wOLjNI="
	testPrvKey1 = "hEI8EHaH11RIUGrXk6Iu6Fn+IgbQ1uSULWLFvYrvARI="
	testPrvKey2 = "WBvNhBZ9X02vTCj4Vqx+aH2pZyBTmvQMwAHqOUH416M="
	testPrvKey3 = "AXsllrR8W8mWtLpxaI0O33N8dbOWnJF2DgS8CZuvvPs="
	testPrvKey4 = "eC3wbu92gyqYqbnoS4gbvw5A2RP7jpU2BDjHJWvMPSs="
	testPrvKey5 = "xhCXGUMzBk7vfBQahZPAe9Gka9mOBQFi1nYxO6uycjo="
	testPrvKey6 = "Ks8sthfbwspQKESYlczdDTMtCC48JvMQ+mOoKMO6gEc="
	testPrvKey7 = "6iqmBkeGpk2aI7ir0E92D5o+N2f5ypd/zN2zBpuYWdg="
	testPrvKey8 = "KaoMwzC4qH7EPkfv86vLUw5QrAf3L+pnxhDXDmzYBg4="
	testPrvKey9 = "L5uQYQXW1RW61eGX/QxIfp0TjQ0klkkN3NnibeL9S/U="
)

func TestFiveNodes(t *testing.T) {
	os.RemoveAll(testDB0)
	os.Mkdir(testDB0, os.ModePerm)
	defer os.RemoveAll(testDB0)
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
	// Context
	ctx, cancel := context.WithCancel(context.Background())
	// New mock chain
	mockChain, err := chainmgr.NewMockChainImplV1(time.Minute, "", 0)
	assert.Empty(t, err)
	servID := mockChain.GetAddr().ID.String()
	servAddrStrs := make([]string, 0)
	for _, addr := range mockChain.GetAddr().Addrs {
		servAddrStrs = append(servAddrStrs, addr.String())
	}
	servAddr := strings.Join(servAddrStrs, ";")
	// New managers
	cmgr0, pmgr0, _, pi0, err := makeManager(ctx, testDB0, testPrvKey0, servID, servAddr, testPrvKey0)
	assert.Empty(t, err)
	cmgr1, pmgr1, h1, _, err := makeManager(ctx, testDB1, testPrvKey1, servID, servAddr, "")
	assert.Empty(t, err)
	cmgr2, pmgr2, h2, pi2, err := makeManager(ctx, testDB2, testPrvKey2, servID, servAddr, "")
	assert.Empty(t, err)
	cmgr3, pmgr3, h3, pi3, err := makeManager(ctx, testDB3, testPrvKey3, servID, servAddr, "")
	assert.Empty(t, err)
	cmgr4, pmgr4, h4, pi4, err := makeManager(ctx, testDB4, testPrvKey4, servID, servAddr, "")
	assert.Empty(t, err)
	// Connect these peers
	err = h1.Connect(ctx, pi0)
	assert.Empty(t, err)
	err = h2.Connect(ctx, pi0)
	assert.Empty(t, err)
	err = h3.Connect(ctx, pi0)
	assert.Empty(t, err)
	err = h4.Connect(ctx, pi0)
	assert.Empty(t, err)
	// Get addrs
	// addr0, err := pmgr0.GetRootAddress(0)
	// assert.Empty(t, err)
	// addr1, err := pmgr1.GetRootAddress(0)
	// assert.Empty(t, err)
	addr2, err := pmgr2.GetRootAddress(0)
	assert.Empty(t, err)
	addr3, err := pmgr3.GetRootAddress(0)
	assert.Empty(t, err)
	addr4, err := pmgr4.GetRootAddress(0)
	assert.Empty(t, err)
	// Generate files
	os.Remove(testFile0)
	err = generateRandomFile(testFile0)
	assert.Empty(t, err)
	defer os.Remove(testFile0)
	os.Remove(testFile1)
	err = generateRandomFile(testFile1)
	assert.Empty(t, err)
	defer os.Remove(testFile1)
	os.Remove(testFile2)
	err = generateRandomFile(testFile2)
	assert.Empty(t, err)
	defer os.Remove(testFile2)
	os.Remove(testFile3)
	err = generateRandomFile(testFile3)
	assert.Empty(t, err)
	defer os.Remove(testFile3)
	os.Remove(testFile4)
	err = generateRandomFile(testFile4)
	assert.Empty(t, err)
	defer os.Remove(testFile4)
	// Import files
	cid0, err := cmgr2.Import(ctx, testFile0)
	assert.Empty(t, err)
	cid1, err := cmgr3.Import(ctx, testFile1)
	assert.Empty(t, err)
	cid2, err := cmgr3.Import(ctx, testFile2)
	assert.Empty(t, err)
	cid3, err := cmgr4.Import(ctx, testFile3)
	assert.Empty(t, err)
	cid4, err := cmgr4.Import(ctx, testFile4)
	assert.Empty(t, err)
	t.Log("node 2 imported one file with cid:", cid0.String())
	t.Log("node 3 imported one file with cid:", cid1.String())
	t.Log("node 3 imported one file with cid:", cid2.String())
	t.Log("node 4 imported one file with cid:", cid3.String())
	t.Log("node 4 imported one file with cid:", cid4.String())

	// Creating payment channels
	offer, err := pmgr1.QueryOutboundChOffer(ctx, 0, addr2, pi2)
	assert.Empty(t, err)
	err = pmgr1.CreateOutboundCh(ctx, offer, big.NewInt(1e13), pi2)
	assert.Empty(t, err)

	offer, err = pmgr2.QueryOutboundChOffer(ctx, 0, addr3, pi3)
	assert.Empty(t, err)
	err = pmgr2.CreateOutboundCh(ctx, offer, big.NewInt(1e13), pi3)
	assert.Empty(t, err)

	offer, err = pmgr3.QueryOutboundChOffer(ctx, 0, addr4, pi4)
	assert.Empty(t, err)
	err = pmgr3.CreateOutboundCh(ctx, offer, big.NewInt(1e13), pi4)
	assert.Empty(t, err)

	// Start serving channels.
	err = pmgr3.StartServing(0, addr4, big.NewInt(10), big.NewInt(200))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)
	pmgr2.StartServing(0, addr3, big.NewInt(5), big.NewInt(100))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)

	// Start serving cids.
	err = cmgr2.StartServing(ctx, 0, cid0, big.NewInt(1))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)
	err = cmgr3.StartServing(ctx, 0, cid1, big.NewInt(2))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)
	err = cmgr3.StartServing(ctx, 0, cid2, big.NewInt(3))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)
	err = cmgr4.StartServing(ctx, 0, cid3, big.NewInt(4))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)
	err = cmgr4.StartServing(ctx, 0, cid4, big.NewInt(5))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)

	// Try search cid offers
	outs, err := pmgr0.ListActiveOutboundChs(ctx)
	assert.Empty(t, err)
	assert.Equal(t, 0, len(outs[0]))
	offers := cmgr0.SearchOffers(ctx, 0, cid0, 3)
	assert.Equal(t, 1, len(offers))
	cidOffer := offers[0]
	offer, err = pmgr0.QueryOutboundChOffer(ctx, 0, cidOffer.ToAddr(), cidOffer.PeerAddr())
	assert.Empty(t, err)
	err = pmgr0.CreateOutboundCh(ctx, offer, big.NewInt(1e13), cidOffer.PeerAddr())
	assert.Empty(t, err)

	// There will be 5 concurrent retrievals.
	// Retreival 0: node 0 retrieves cid0 from node 2
	routine0 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr0.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 1, len(outs[0]))
		pid, _, _, _, err := pmgr0.GetPeerInfo(0, outs[0][0])
		assert.Empty(t, err)
		// First try to query offer directly from connected peers.
		cOffer, err := cmgr0.QueryOffer(ctx, pid, 0, cid0)
		assert.Empty(t, err)
		// Found offer from connected peers, retrieve directly.
		err = cmgr0.Retrieve(ctx, *cOffer, nil, testOut0)
		assert.Empty(t, err)
	}
	// Retrieval 1: node 0 retrieves cid3 from node 4
	routine1 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr0.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 1, len(outs[0]))
		pid, _, _, _, err := pmgr0.GetPeerInfo(0, outs[0][0])
		assert.Empty(t, err)
		// First try to query offer directly from connected peers.
		_, err = cmgr0.QueryOffer(ctx, pid, 0, cid3)
		assert.NotEmpty(t, err)
		// Does not found offer from connected peers. Using DHT.
		cOffers := cmgr0.SearchOffers(ctx, 0, cid3, 3)
		assert.Equal(t, 1, len(cOffers))
		cOffer := cOffers[0]
		// Found one cid offer from DHT, now find a correspoding pay offer to do payment over payment network.
		pOffers, err := pmgr0.SearchPayOffer(ctx, cOffer.CurrencyID(), cOffer.ToAddr(), big.NewInt(0).Mul(big.NewInt(cOffer.Size()), cOffer.PPB()))
		assert.Empty(t, err)
		assert.Equal(t, 1, len(pOffers))
		pOffer := pOffers[0]
		// Found one pay offer to use, now retrieve content.
		err = cmgr0.Retrieve(ctx, cOffer, &pOffer, testOut3)
		assert.Empty(t, err)
	}
	// Retrieval 2: node 1 retrieves cid4 from node 4
	routine2 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr1.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 1, len(outs[0]))
		pid, _, _, _, err := pmgr1.GetPeerInfo(0, outs[0][0])
		assert.Empty(t, err)
		// First try to query offer directly from connected peers.
		_, err = cmgr1.QueryOffer(ctx, pid, 0, cid4)
		assert.NotEmpty(t, err)
		// Does not found offer from connected peers. Using DHT.
		cOffers := cmgr1.SearchOffers(ctx, 0, cid4, 3)
		assert.Equal(t, 1, len(cOffers))
		cOffer := cOffers[0]
		// Found one cid offer from DHT, now find a correspoding pay offer to do payment over payment network.
		pOffers, err := pmgr1.SearchPayOffer(ctx, cOffer.CurrencyID(), cOffer.ToAddr(), big.NewInt(0).Mul(big.NewInt(cOffer.Size()), cOffer.PPB()))
		assert.Empty(t, err)
		assert.Equal(t, 1, len(pOffers))
		pOffer := pOffers[0]
		// Found one pay offer to use, now retrieve content.
		err = cmgr1.Retrieve(ctx, cOffer, &pOffer, testOut4)
		assert.Empty(t, err)
	}
	// Retrieval 3: node 1 retrieves cid1 from node 3
	routine3 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr1.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 1, len(outs[0]))
		pid, _, _, _, err := pmgr1.GetPeerInfo(0, outs[0][0])
		assert.Empty(t, err)
		// First try to query offer directly from connected peers.
		_, err = cmgr1.QueryOffer(ctx, pid, 0, cid1)
		assert.NotEmpty(t, err)
		// Does not found offer from connected peers. Using DHT.
		cOffers := cmgr1.SearchOffers(ctx, 0, cid1, 3)
		assert.Equal(t, 1, len(cOffers))
		cOffer := cOffers[0]
		// Found one cid offer from DHT, now find a correspoding pay offer to do payment over payment network.
		pOffers, err := pmgr1.SearchPayOffer(ctx, cOffer.CurrencyID(), cOffer.ToAddr(), big.NewInt(0).Mul(big.NewInt(cOffer.Size()), cOffer.PPB()))
		assert.Empty(t, err)
		assert.Equal(t, 1, len(pOffers))
		pOffer := pOffers[0]
		// Found one pay offer to use, now retrieve content.
		err = cmgr1.Retrieve(ctx, cOffer, &pOffer, testOut1)
		assert.Empty(t, err)
	}
	// Retrieval 4: node 2 retrieves cid2 from node 3
	routine4 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr2.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 1, len(outs[0]))
		pid, _, _, _, err := pmgr2.GetPeerInfo(0, outs[0][0])
		assert.Empty(t, err)
		// First try to query offer directly from connected peers.
		cOffer, err := cmgr2.QueryOffer(ctx, pid, 0, cid2)
		assert.Empty(t, err)
		// Found offer from connected peers, retrieve directly.
		err = cmgr2.Retrieve(ctx, *cOffer, nil, testOut2)
		assert.Empty(t, err)
	}

	// Start
	var wg sync.WaitGroup
	wg.Add(1)
	go routine0(&wg)
	wg.Add(1)
	go routine1(&wg)
	wg.Add(1)
	go routine2(&wg)
	wg.Add(1)
	go routine3(&wg)
	wg.Add(1)
	go routine4(&wg)
	// Wait for all routines to finish
	wg.Wait()

	// Verify Retrieval 0
	defer os.Remove(testOut0)
	assert.True(t, equalFile(testFile0, testOut0))
	// Verify Retrieval 1
	defer os.Remove(testOut3)
	assert.True(t, equalFile(testFile3, testOut3))
	// Verify Retrieval 2
	defer os.Remove(testOut4)
	assert.True(t, equalFile(testFile4, testOut4))
	// Verify Retrieval 3
	defer os.Remove(testOut1)
	assert.True(t, equalFile(testFile1, testOut1))
	// Verify Retrieval 4
	defer os.Remove(testOut2)
	assert.True(t, equalFile(testFile2, testOut2))

	cancel()
	time.Sleep(3 * time.Second)
}

func TestTenNodes(t *testing.T) {
	os.RemoveAll(testDB0)
	os.Mkdir(testDB0, os.ModePerm)
	defer os.RemoveAll(testDB0)
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
	os.RemoveAll(testDB6)
	os.Mkdir(testDB6, os.ModePerm)
	defer os.RemoveAll(testDB6)
	os.RemoveAll(testDB7)
	os.Mkdir(testDB7, os.ModePerm)
	defer os.RemoveAll(testDB7)
	os.RemoveAll(testDB8)
	os.Mkdir(testDB8, os.ModePerm)
	defer os.RemoveAll(testDB8)
	os.RemoveAll(testDB9)
	os.Mkdir(testDB9, os.ModePerm)
	defer os.RemoveAll(testDB9)
	// Context
	ctx, cancel := context.WithCancel(context.Background())
	// New mock chain
	mockChain, err := chainmgr.NewMockChainImplV1(time.Minute, "", 0)
	assert.Empty(t, err)
	servID := mockChain.GetAddr().ID.String()
	servAddrStrs := make([]string, 0)
	for _, addr := range mockChain.GetAddr().Addrs {
		servAddrStrs = append(servAddrStrs, addr.String())
	}
	servAddr := strings.Join(servAddrStrs, ";")
	// New managers
	cmgr0, pmgr0, _, pi0, err := makeManager(ctx, testDB0, testPrvKey0, servID, servAddr, "")
	assert.Empty(t, err)
	cmgr1, pmgr1, h1, pi1, err := makeManager(ctx, testDB1, testPrvKey1, servID, servAddr, "")
	assert.Empty(t, err)
	cmgr2, pmgr2, h2, pi2, err := makeManager(ctx, testDB2, testPrvKey2, servID, servAddr, "")
	assert.Empty(t, err)
	cmgr3, pmgr3, h3, pi3, err := makeManager(ctx, testDB3, testPrvKey3, servID, servAddr, "")
	assert.Empty(t, err)
	cmgr4, pmgr4, h4, pi4, err := makeManager(ctx, testDB4, testPrvKey4, servID, servAddr, "")
	assert.Empty(t, err)
	cmgr5, pmgr5, h5, pi5, err := makeManager(ctx, testDB5, testPrvKey5, servID, servAddr, "")
	assert.Empty(t, err)
	cmgr6, pmgr6, h6, pi6, err := makeManager(ctx, testDB6, testPrvKey6, servID, servAddr, "")
	assert.Empty(t, err)
	cmgr7, pmgr7, h7, pi7, err := makeManager(ctx, testDB7, testPrvKey7, servID, servAddr, "")
	assert.Empty(t, err)
	cmgr8, pmgr8, h8, pi8, err := makeManager(ctx, testDB8, testPrvKey8, servID, servAddr, "")
	assert.Empty(t, err)
	cmgr9, pmgr9, h9, pi9, err := makeManager(ctx, testDB9, testPrvKey9, servID, servAddr, "")
	assert.Empty(t, err)
	// Connect these peers
	err = h1.Connect(ctx, pi0)
	assert.Empty(t, err)
	err = h2.Connect(ctx, pi0)
	assert.Empty(t, err)
	err = h3.Connect(ctx, pi0)
	assert.Empty(t, err)
	err = h4.Connect(ctx, pi0)
	assert.Empty(t, err)
	err = h5.Connect(ctx, pi0)
	assert.Empty(t, err)
	err = h6.Connect(ctx, pi0)
	assert.Empty(t, err)
	err = h7.Connect(ctx, pi0)
	assert.Empty(t, err)
	err = h8.Connect(ctx, pi0)
	assert.Empty(t, err)
	err = h9.Connect(ctx, pi0)
	assert.Empty(t, err)
	// Get addrs
	addr0, err := pmgr0.GetRootAddress(0)
	assert.Empty(t, err)
	addr1, err := pmgr1.GetRootAddress(0)
	assert.Empty(t, err)
	addr2, err := pmgr2.GetRootAddress(0)
	assert.Empty(t, err)
	addr3, err := pmgr3.GetRootAddress(0)
	assert.Empty(t, err)
	addr4, err := pmgr4.GetRootAddress(0)
	assert.Empty(t, err)
	addr5, err := pmgr5.GetRootAddress(0)
	assert.Empty(t, err)
	addr6, err := pmgr6.GetRootAddress(0)
	assert.Empty(t, err)
	addr7, err := pmgr7.GetRootAddress(0)
	assert.Empty(t, err)
	addr8, err := pmgr8.GetRootAddress(0)
	assert.Empty(t, err)
	addr9, err := pmgr9.GetRootAddress(0)
	assert.Empty(t, err)
	// Generate files
	os.Remove(testFile0)
	err = generateRandomFile(testFile0)
	assert.Empty(t, err)
	defer os.Remove(testFile0)
	os.Remove(testFile1)
	err = generateRandomFile(testFile1)
	assert.Empty(t, err)
	defer os.Remove(testFile1)
	os.Remove(testFile2)
	err = generateRandomFile(testFile2)
	assert.Empty(t, err)
	defer os.Remove(testFile2)
	os.Remove(testFile3)
	err = generateRandomFile(testFile3)
	assert.Empty(t, err)
	defer os.Remove(testFile3)
	os.Remove(testFile4)
	err = generateRandomFile(testFile4)
	assert.Empty(t, err)
	defer os.Remove(testFile4)
	os.Remove(testFile5)
	err = generateRandomFile(testFile5)
	assert.Empty(t, err)
	defer os.Remove(testFile5)
	os.Remove(testFile6)
	err = generateRandomFile(testFile6)
	assert.Empty(t, err)
	defer os.Remove(testFile6)
	os.Remove(testFile7)
	err = generateRandomFile(testFile7)
	assert.Empty(t, err)
	defer os.Remove(testFile7)
	os.Remove(testFile8)
	err = generateRandomFile(testFile8)
	assert.Empty(t, err)
	defer os.Remove(testFile8)
	err = generateRandomFile(testFile9)
	assert.Empty(t, err)
	defer os.Remove(testFile9)
	// Import files
	cid0, err := cmgr0.Import(ctx, testFile0)
	assert.Empty(t, err)
	cid1, err := cmgr1.Import(ctx, testFile1)
	assert.Empty(t, err)
	cid2, err := cmgr2.Import(ctx, testFile2)
	assert.Empty(t, err)
	cid3, err := cmgr3.Import(ctx, testFile3)
	assert.Empty(t, err)
	cid4, err := cmgr4.Import(ctx, testFile4)
	assert.Empty(t, err)
	cid5, err := cmgr5.Import(ctx, testFile5)
	assert.Empty(t, err)
	cid6, err := cmgr6.Import(ctx, testFile6)
	assert.Empty(t, err)
	cid7, err := cmgr7.Import(ctx, testFile7)
	assert.Empty(t, err)
	cid8, err := cmgr8.Import(ctx, testFile8)
	assert.Empty(t, err)
	cid9, err := cmgr9.Import(ctx, testFile9)
	assert.Empty(t, err)
	t.Log("node 0 imported one file with cid:", cid0.String())
	t.Log("node 1 imported one file with cid:", cid1.String())
	t.Log("node 2 imported one file with cid:", cid2.String())
	t.Log("node 3 imported one file with cid:", cid3.String())
	t.Log("node 4 imported one file with cid:", cid4.String())
	t.Log("node 5 imported one file with cid:", cid5.String())
	t.Log("node 6 imported one file with cid:", cid6.String())
	t.Log("node 7 imported one file with cid:", cid7.String())
	t.Log("node 8 imported one file with cid:", cid8.String())
	t.Log("node 9 imported one file with cid:", cid9.String())

	// Creating payment channels.
	offer, err := pmgr0.QueryOutboundChOffer(ctx, 0, addr1, pi1)
	assert.Empty(t, err)
	err = pmgr0.CreateOutboundCh(ctx, offer, big.NewInt(1e13), pi1)
	assert.Empty(t, err)

	offer, err = pmgr1.QueryOutboundChOffer(ctx, 0, addr2, pi2)
	assert.Empty(t, err)
	err = pmgr1.CreateOutboundCh(ctx, offer, big.NewInt(1e13), pi2)
	assert.Empty(t, err)

	offer, err = pmgr1.QueryOutboundChOffer(ctx, 0, addr4, pi4)
	assert.Empty(t, err)
	err = pmgr1.CreateOutboundCh(ctx, offer, big.NewInt(1e13), pi4)
	assert.Empty(t, err)

	offer, err = pmgr1.QueryOutboundChOffer(ctx, 0, addr6, pi6)
	assert.Empty(t, err)
	err = pmgr1.CreateOutboundCh(ctx, offer, big.NewInt(1e13), pi6)
	assert.Empty(t, err)

	offer, err = pmgr2.QueryOutboundChOffer(ctx, 0, addr3, pi3)
	assert.Empty(t, err)
	err = pmgr2.CreateOutboundCh(ctx, offer, big.NewInt(1e13), pi3)
	assert.Empty(t, err)

	offer, err = pmgr3.QueryOutboundChOffer(ctx, 0, addr0, pi0)
	assert.Empty(t, err)
	err = pmgr3.CreateOutboundCh(ctx, offer, big.NewInt(1e13), pi0)
	assert.Empty(t, err)

	offer, err = pmgr4.QueryOutboundChOffer(ctx, 0, addr5, pi5)
	assert.Empty(t, err)
	err = pmgr4.CreateOutboundCh(ctx, offer, big.NewInt(1e13), pi5)
	assert.Empty(t, err)

	offer, err = pmgr4.QueryOutboundChOffer(ctx, 0, addr7, pi7)
	assert.Empty(t, err)
	err = pmgr4.CreateOutboundCh(ctx, offer, big.NewInt(1e13), pi7)
	assert.Empty(t, err)

	offer, err = pmgr6.QueryOutboundChOffer(ctx, 0, addr7, pi7)
	assert.Empty(t, err)
	err = pmgr6.CreateOutboundCh(ctx, offer, big.NewInt(1e13), pi7)
	assert.Empty(t, err)

	offer, err = pmgr7.QueryOutboundChOffer(ctx, 0, addr8, pi8)
	assert.Empty(t, err)
	err = pmgr7.CreateOutboundCh(ctx, offer, big.NewInt(1e13), pi8)
	assert.Empty(t, err)

	offer, err = pmgr8.QueryOutboundChOffer(ctx, 0, addr9, pi9)
	assert.Empty(t, err)
	err = pmgr8.CreateOutboundCh(ctx, offer, big.NewInt(1e13), pi9)
	assert.Empty(t, err)

	offer, err = pmgr8.QueryOutboundChOffer(ctx, 0, addr5, pi5)
	assert.Empty(t, err)
	err = pmgr8.CreateOutboundCh(ctx, offer, big.NewInt(1e13), pi5)
	assert.Empty(t, err)

	// Start serving.
	err = pmgr8.StartServing(0, addr9, big.NewInt(10), big.NewInt(200))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)
	err = pmgr8.StartServing(0, addr5, big.NewInt(10), big.NewInt(200))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)
	err = pmgr7.StartServing(0, addr8, big.NewInt(10), big.NewInt(200))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)
	err = pmgr4.StartServing(0, addr7, big.NewInt(5), big.NewInt(300))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)
	err = pmgr4.StartServing(0, addr5, big.NewInt(5), big.NewInt(300))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)
	err = pmgr6.StartServing(0, addr7, big.NewInt(25), big.NewInt(100))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)
	err = pmgr1.StartServing(0, addr6, big.NewInt(5), big.NewInt(200))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)
	err = pmgr1.StartServing(0, addr4, big.NewInt(5), big.NewInt(200))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)
	err = pmgr0.StartServing(0, addr1, big.NewInt(20), big.NewInt(300))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)
	err = pmgr3.StartServing(0, addr0, big.NewInt(13), big.NewInt(150))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)
	err = pmgr2.StartServing(0, addr3, big.NewInt(17), big.NewInt(150))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)
	err = pmgr1.StartServing(0, addr2, big.NewInt(19), big.NewInt(150))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)

	// Force publish two times to sync every node.
	for i := 0; i < 2; i++ {
		pmgr0.ForcePublishServings()
		time.Sleep(10 * time.Millisecond)
		pmgr1.ForcePublishServings()
		time.Sleep(10 * time.Millisecond)
		pmgr2.ForcePublishServings()
		time.Sleep(10 * time.Millisecond)
		pmgr3.ForcePublishServings()
		time.Sleep(10 * time.Millisecond)
		pmgr4.ForcePublishServings()
		time.Sleep(10 * time.Millisecond)
		pmgr5.ForcePublishServings()
		time.Sleep(10 * time.Millisecond)
		pmgr6.ForcePublishServings()
		time.Sleep(10 * time.Millisecond)
		pmgr7.ForcePublishServings()
		time.Sleep(10 * time.Millisecond)
		pmgr8.ForcePublishServings()
		time.Sleep(10 * time.Millisecond)
		pmgr9.ForcePublishServings()
		time.Sleep(10 * time.Millisecond)
	}

	// Start serving cids.
	err = cmgr0.StartServing(ctx, 0, cid0, big.NewInt(1))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)
	err = cmgr1.StartServing(ctx, 0, cid1, big.NewInt(2))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)
	err = cmgr2.StartServing(ctx, 0, cid2, big.NewInt(3))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)
	err = cmgr3.StartServing(ctx, 0, cid3, big.NewInt(4))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)
	err = cmgr4.StartServing(ctx, 0, cid4, big.NewInt(5))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)
	err = cmgr5.StartServing(ctx, 0, cid5, big.NewInt(6))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)
	err = cmgr6.StartServing(ctx, 0, cid6, big.NewInt(7))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)
	err = cmgr7.StartServing(ctx, 0, cid7, big.NewInt(8))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)
	err = cmgr8.StartServing(ctx, 0, cid8, big.NewInt(9))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)
	err = cmgr9.StartServing(ctx, 0, cid9, big.NewInt(10))
	assert.Empty(t, err)
	time.Sleep(10 * time.Millisecond)

	// There will be 20 concurrent retrievals.
	// Retrieval 0: node 0 retrieves cid7 from node 7
	routine0 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr0.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 1, len(outs[0]))
		pid, _, _, _, err := pmgr0.GetPeerInfo(0, outs[0][0])
		assert.Empty(t, err)
		// First try to query offer directly from connected peers.
		_, err = cmgr0.QueryOffer(ctx, pid, 0, cid7)
		assert.NotEmpty(t, err)
		// Does not found offer from connected peers. Using DHT.
		cOffers := cmgr0.SearchOffers(ctx, 0, cid7, 3)
		assert.Equal(t, 1, len(cOffers))
		cOffer := cOffers[0]
		// Found one cid offer from DHT, now find a correspoding pay offer to do payment over payment network.
		pOffers, err := pmgr0.SearchPayOffer(ctx, cOffer.CurrencyID(), cOffer.ToAddr(), big.NewInt(0).Mul(big.NewInt(cOffer.Size()), cOffer.PPB()))
		assert.Empty(t, err)
		assert.Equal(t, 2, len(pOffers))
		ratio0 := big.NewFloat(0).SetInt(pOffers[0].PPP())
		ratio0.Quo(ratio0, big.NewFloat(0).SetInt(pOffers[0].Period()))
		ratio1 := big.NewFloat(0).SetInt(pOffers[1].PPP())
		ratio1.Quo(ratio1, big.NewFloat(0).SetInt(pOffers[1].Period()))
		var pOffer *payoffer.PayOffer
		if ratio0.Cmp(ratio1) > 0 {
			pOffer = &pOffers[1]
		} else {
			pOffer = &pOffers[0]
		}
		// Found two pay offers to use, choose cheap one.
		err = cmgr0.Retrieve(ctx, cOffer, pOffer, testOut0)
		assert.Empty(t, err)
	}
	// Retrieval 1: node 0 retrieves cid9 from node 9
	routine1 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr0.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 1, len(outs[0]))
		pid, _, _, _, err := pmgr0.GetPeerInfo(0, outs[0][0])
		assert.Empty(t, err)
		// First try to query offer directly from connected peers.
		_, err = cmgr0.QueryOffer(ctx, pid, 0, cid9)
		assert.NotEmpty(t, err)
		// Does not found offer from connected peers. Using DHT.
		cOffers := cmgr0.SearchOffers(ctx, 0, cid9, 3)
		assert.Equal(t, 1, len(cOffers))
		cOffer := cOffers[0]
		// Found one cid offer from DHT, now find a correspoding pay offer to do payment over payment network.
		pOffers, err := pmgr0.SearchPayOffer(ctx, cOffer.CurrencyID(), cOffer.ToAddr(), big.NewInt(0).Mul(big.NewInt(cOffer.Size()), cOffer.PPB()))
		assert.Empty(t, err)
		assert.Equal(t, 2, len(pOffers))
		ratio0 := big.NewFloat(0).SetInt(pOffers[0].PPP())
		ratio0.Quo(ratio0, big.NewFloat(0).SetInt(pOffers[0].Period()))
		ratio1 := big.NewFloat(0).SetInt(pOffers[1].PPP())
		ratio1.Quo(ratio1, big.NewFloat(0).SetInt(pOffers[1].Period()))
		var pOffer *payoffer.PayOffer
		if ratio0.Cmp(ratio1) > 0 {
			pOffer = &pOffers[1]
		} else {
			pOffer = &pOffers[0]
		}
		// Found two pay offers to use, choose cheap one.
		err = cmgr0.Retrieve(ctx, cOffer, pOffer, testOut1)
		assert.Empty(t, err)
	}
	// Retrieval 2: node 0 retrieves cid5 from node 5
	routine2 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr0.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 1, len(outs[0]))
		pid, _, _, _, err := pmgr0.GetPeerInfo(0, outs[0][0])
		assert.Empty(t, err)
		// First try to query offer directly from connected peers.
		_, err = cmgr0.QueryOffer(ctx, pid, 0, cid5)
		assert.NotEmpty(t, err)
		// Does not found offer from connected peers. Using DHT.
		cOffers := cmgr0.SearchOffers(ctx, 0, cid5, 3)
		assert.Equal(t, 1, len(cOffers))
		cOffer := cOffers[0]
		// Found one cid offer from DHT, now find a correspoding pay offer to do payment over payment network.
		pOffers, err := pmgr0.SearchPayOffer(ctx, cOffer.CurrencyID(), cOffer.ToAddr(), big.NewInt(0).Mul(big.NewInt(cOffer.Size()), cOffer.PPB()))
		assert.Empty(t, err)
		assert.Equal(t, 2, len(pOffers))
		ratio0 := big.NewFloat(0).SetInt(pOffers[0].PPP())
		ratio0.Quo(ratio0, big.NewFloat(0).SetInt(pOffers[0].Period()))
		ratio1 := big.NewFloat(0).SetInt(pOffers[1].PPP())
		ratio1.Quo(ratio1, big.NewFloat(0).SetInt(pOffers[1].Period()))
		var pOffer *payoffer.PayOffer
		if ratio0.Cmp(ratio1) > 0 {
			pOffer = &pOffers[1]
		} else {
			pOffer = &pOffers[0]
		}
		// Found two pay offers to use, choose cheap one.
		err = cmgr0.Retrieve(ctx, cOffer, pOffer, testOut2)
		assert.Empty(t, err)
	}
	// Retrieval 3: node 0 retrieves cid1 from node 1
	routine3 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr0.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 1, len(outs[0]))
		pid, _, _, _, err := pmgr0.GetPeerInfo(0, outs[0][0])
		assert.Empty(t, err)
		// First try to query offer directly from connected peers.
		cOffer, err := cmgr0.QueryOffer(ctx, pid, 0, cid1)
		assert.Empty(t, err)
		// Found offer from connected peers, retrieve directly.
		err = cmgr0.Retrieve(ctx, *cOffer, nil, testOut3)
		assert.Empty(t, err)
	}
	// Retrieval 4: node 0 retrieves cid3 from node 3
	routine4 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr0.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 1, len(outs[0]))
		pid, _, _, _, err := pmgr0.GetPeerInfo(0, outs[0][0])
		assert.Empty(t, err)
		// First try to query offer directly from connected peers.
		_, err = cmgr0.QueryOffer(ctx, pid, 0, cid3)
		assert.NotEmpty(t, err)
		// Does not found offer from connected peers. Using DHT.
		cOffers := cmgr0.SearchOffers(ctx, 0, cid3, 3)
		assert.Equal(t, 1, len(cOffers))
		cOffer := cOffers[0]
		// Found one cid offer from DHT, now find a correspoding pay offer to do payment over payment network.
		pOffers, err := pmgr0.SearchPayOffer(ctx, cOffer.CurrencyID(), cOffer.ToAddr(), big.NewInt(0).Mul(big.NewInt(cOffer.Size()), cOffer.PPB()))
		assert.Empty(t, err)
		assert.Equal(t, 1, len(pOffers))
		pOffer := &pOffers[0]
		// Found one pay offer to use, choose it.
		err = cmgr0.Retrieve(ctx, cOffer, pOffer, testOut4)
		assert.Empty(t, err)
	}
	// Retrieval 5: node 3 retrieves cid0 from node 0
	routine5 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr3.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 1, len(outs[0]))
		pid, _, _, _, err := pmgr3.GetPeerInfo(0, outs[0][0])
		assert.Empty(t, err)
		// First try to query offer directly from connected peers.
		cOffer, err := cmgr3.QueryOffer(ctx, pid, 0, cid0)
		assert.Empty(t, err)
		// Found offer from connected peers, retrieve directly.
		err = cmgr3.Retrieve(ctx, *cOffer, nil, testOut5)
		assert.Empty(t, err)
	}
	// Retrieval 6: node 3 retrieves cid8 from node 8
	routine6 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr3.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 1, len(outs[0]))
		pid, _, _, _, err := pmgr3.GetPeerInfo(0, outs[0][0])
		assert.Empty(t, err)
		// First try to query offer directly from connected peers.
		_, err = cmgr3.QueryOffer(ctx, pid, 0, cid8)
		assert.NotEmpty(t, err)
		// Does not found offer from connected peers. Using DHT.
		cOffers := cmgr3.SearchOffers(ctx, 0, cid8, 3)
		assert.Equal(t, 1, len(cOffers))
		cOffer := cOffers[0]
		// Found one cid offer from DHT, now find a correspoding pay offer to do payment over payment network.
		pOffers, err := pmgr3.SearchPayOffer(ctx, cOffer.CurrencyID(), cOffer.ToAddr(), big.NewInt(0).Mul(big.NewInt(cOffer.Size()), cOffer.PPB()))
		assert.Empty(t, err)
		assert.Equal(t, 2, len(pOffers))
		ratio0 := big.NewFloat(0).SetInt(pOffers[0].PPP())
		ratio0.Quo(ratio0, big.NewFloat(0).SetInt(pOffers[0].Period()))
		ratio1 := big.NewFloat(0).SetInt(pOffers[1].PPP())
		ratio1.Quo(ratio1, big.NewFloat(0).SetInt(pOffers[1].Period()))
		var pOffer *payoffer.PayOffer
		if ratio0.Cmp(ratio1) > 0 {
			pOffer = &pOffers[1]
		} else {
			pOffer = &pOffers[0]
		}
		// Found two pay offers to use, choose cheap one.
		err = cmgr3.Retrieve(ctx, cOffer, pOffer, testOut6)
		assert.Empty(t, err)
	}
	// Retrieval 7: node 1 retrieves cid2 from node 2
	routine7 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr1.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 3, len(outs[0]))
		for _, out := range outs[0] {
			pid, _, _, _, err := pmgr1.GetPeerInfo(0, out)
			assert.Empty(t, err)
			// First try to query offer directly from connected peers.
			cOffer, err := cmgr1.QueryOffer(ctx, pid, 0, cid2)
			if err != nil {
				continue
			}
			// Found offer from connected peers, retrieve directly.
			err = cmgr1.Retrieve(ctx, *cOffer, nil, testOut7)
			assert.Empty(t, err)
			break
		}
	}
	// Retrieval 8: node 1 retrieves cid4 from node 4
	routine8 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr1.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 3, len(outs[0]))
		for _, out := range outs[0] {
			pid, _, _, _, err := pmgr1.GetPeerInfo(0, out)
			assert.Empty(t, err)
			// First try to query offer directly from connected peers.
			cOffer, err := cmgr1.QueryOffer(ctx, pid, 0, cid4)
			if err != nil {
				continue
			}
			// Found offer from connected peers, retrieve directly.
			err = cmgr1.Retrieve(ctx, *cOffer, nil, testOut8)
			assert.Empty(t, err)
			break
		}
	}
	// Retrieval 9: node 1 retrieves cid0 from node 0
	routine9 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr1.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 3, len(outs[0]))
		for _, out := range outs[0] {
			pid, _, _, _, err := pmgr1.GetPeerInfo(0, out)
			assert.Empty(t, err)
			// First try to query offer directly from connected peers.
			_, err = cmgr1.QueryOffer(ctx, pid, 0, cid0)
			assert.NotEmpty(t, err)
		}
		// Does not found offer from connected peers. Using DHT.
		cOffers := cmgr1.SearchOffers(ctx, 0, cid0, 3)
		assert.Equal(t, 1, len(cOffers))
		cOffer := cOffers[0]
		// Found one cid offer from DHT, now find a correspoding pay offer to do payment over payment network.
		pOffers, err := pmgr1.SearchPayOffer(ctx, cOffer.CurrencyID(), cOffer.ToAddr(), big.NewInt(0).Mul(big.NewInt(cOffer.Size()), cOffer.PPB()))
		assert.Empty(t, err)
		assert.Equal(t, 1, len(pOffers))
		pOffer := &pOffers[0]
		// Found one pay offer to use, choose it.
		err = cmgr1.Retrieve(ctx, cOffer, pOffer, testOut9)
		assert.Empty(t, err)
	}
	// Retrieval 10: node 1 retrieves cid6 from node 6
	routine10 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr1.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 3, len(outs[0]))
		for _, out := range outs[0] {
			pid, _, _, _, err := pmgr1.GetPeerInfo(0, out)
			assert.Empty(t, err)
			// First try to query offer directly from connected peers.
			cOffer, err := cmgr1.QueryOffer(ctx, pid, 0, cid6)
			if err != nil {
				continue
			}
			// Found offer from connected peers, retrieve directly.
			err = cmgr1.Retrieve(ctx, *cOffer, nil, testOut10)
			assert.Empty(t, err)
			break
		}
	}
	// Retrieval 11: node 2 retrieves cid3 from node 3
	routine11 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr2.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 1, len(outs[0]))
		pid, _, _, _, err := pmgr2.GetPeerInfo(0, outs[0][0])
		assert.Empty(t, err)
		// First try to query offer directly from connected peers.
		cOffer, err := cmgr2.QueryOffer(ctx, pid, 0, cid3)
		assert.Empty(t, err)
		// Found offer from connected peers, retrieve directly.
		err = cmgr2.Retrieve(ctx, *cOffer, nil, testOut11)
		assert.Empty(t, err)
	}
	// Retrieval 12: node 2 retrieves cid5 from node 5
	routine12 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr2.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 1, len(outs[0]))
		pid, _, _, _, err := pmgr2.GetPeerInfo(0, outs[0][0])
		assert.Empty(t, err)
		// First try to query offer directly from connected peers.
		_, err = cmgr2.QueryOffer(ctx, pid, 0, cid5)
		assert.NotEmpty(t, err)
		// Does not found offer from connected peers. Using DHT.
		cOffers := cmgr2.SearchOffers(ctx, 0, cid5, 3)
		assert.Equal(t, 1, len(cOffers))
		cOffer := cOffers[0]
		// Found one cid offer from DHT, now find a correspoding pay offer to do payment over payment network.
		pOffers, err := pmgr2.SearchPayOffer(ctx, cOffer.CurrencyID(), cOffer.ToAddr(), big.NewInt(0).Mul(big.NewInt(cOffer.Size()), cOffer.PPB()))
		assert.Empty(t, err)
		assert.Equal(t, 2, len(pOffers))
		ratio0 := big.NewFloat(0).SetInt(pOffers[0].PPP())
		ratio0.Quo(ratio0, big.NewFloat(0).SetInt(pOffers[0].Period()))
		ratio1 := big.NewFloat(0).SetInt(pOffers[1].PPP())
		ratio1.Quo(ratio1, big.NewFloat(0).SetInt(pOffers[1].Period()))
		var pOffer *payoffer.PayOffer
		if ratio0.Cmp(ratio1) > 0 {
			pOffer = &pOffers[1]
		} else {
			pOffer = &pOffers[0]
		}
		// Found two pay offers to use, choose cheap one.
		err = cmgr2.Retrieve(ctx, cOffer, pOffer, testOut12)
		assert.Empty(t, err)
	}
	// Retrieval 13: node 2 retrieves cid9 from node 9
	routine13 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr2.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 1, len(outs[0]))
		pid, _, _, _, err := pmgr2.GetPeerInfo(0, outs[0][0])
		assert.Empty(t, err)
		// First try to query offer directly from connected peers.
		_, err = cmgr2.QueryOffer(ctx, pid, 0, cid9)
		assert.NotEmpty(t, err)
		// Does not found offer from connected peers. Using DHT.
		cOffers := cmgr2.SearchOffers(ctx, 0, cid9, 3)
		assert.Equal(t, 1, len(cOffers))
		cOffer := cOffers[0]
		// Found one cid offer from DHT, now find a correspoding pay offer to do payment over payment network.
		pOffers, err := pmgr2.SearchPayOffer(ctx, cOffer.CurrencyID(), cOffer.ToAddr(), big.NewInt(0).Mul(big.NewInt(cOffer.Size()), cOffer.PPB()))
		assert.Empty(t, err)
		assert.Equal(t, 2, len(pOffers))
		ratio0 := big.NewFloat(0).SetInt(pOffers[0].PPP())
		ratio0.Quo(ratio0, big.NewFloat(0).SetInt(pOffers[0].Period()))
		ratio1 := big.NewFloat(0).SetInt(pOffers[1].PPP())
		ratio1.Quo(ratio1, big.NewFloat(0).SetInt(pOffers[1].Period()))
		var pOffer *payoffer.PayOffer
		if ratio0.Cmp(ratio1) > 0 {
			pOffer = &pOffers[1]
		} else {
			pOffer = &pOffers[0]
		}
		// Found two pay offers to use, choose cheap one.
		err = cmgr2.Retrieve(ctx, cOffer, pOffer, testOut13)
		assert.Empty(t, err)
	}
	// Retrieval 14: node 4 retrieves cid5 from node 5
	routine14 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr4.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 2, len(outs[0]))
		for _, out := range outs[0] {
			pid, _, _, _, err := pmgr4.GetPeerInfo(0, out)
			assert.Empty(t, err)
			// First try to query offer directly from connected peers.
			cOffer, err := cmgr4.QueryOffer(ctx, pid, 0, cid5)
			if err != nil {
				continue
			}
			// Found offer from connected peers, retrieve directly.
			err = cmgr4.Retrieve(ctx, *cOffer, nil, testOut14)
			assert.Empty(t, err)
			break
		}
	}
	// Retrieval 15: node 4 retrieves cid9 from node 9
	routine15 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr4.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 2, len(outs[0]))
		for _, out := range outs[0] {
			pid, _, _, _, err := pmgr4.GetPeerInfo(0, out)
			assert.Empty(t, err)
			// First try to query offer directly from connected peers.
			_, err = cmgr4.QueryOffer(ctx, pid, 0, cid9)
			assert.NotEmpty(t, err)
			break
		}
		// Does not found offer from connected peers. Using DHT.
		cOffers := cmgr4.SearchOffers(ctx, 0, cid9, 3)
		assert.Equal(t, 1, len(cOffers))
		cOffer := cOffers[0]
		// Found one cid offer from DHT, now find a correspoding pay offer to do payment over payment network.
		pOffers, err := pmgr4.SearchPayOffer(ctx, cOffer.CurrencyID(), cOffer.ToAddr(), big.NewInt(0).Mul(big.NewInt(cOffer.Size()), cOffer.PPB()))
		assert.Empty(t, err)
		assert.Equal(t, 1, len(pOffers))
		pOffer := &pOffers[0]
		// Found one pay offer to use, choose it.
		err = cmgr4.Retrieve(ctx, cOffer, pOffer, testOut15)
		assert.Empty(t, err)
	}
	// Retrieval 16: node 6 retrieves cid7 from node 7
	routine16 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr6.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 1, len(outs[0]))
		pid, _, _, _, err := pmgr6.GetPeerInfo(0, outs[0][0])
		assert.Empty(t, err)
		// First try to query offer directly from connected peers.
		cOffer, err := cmgr6.QueryOffer(ctx, pid, 0, cid7)
		assert.Empty(t, err)
		// Found offer from connected peers, retrieve directly.
		err = cmgr6.Retrieve(ctx, *cOffer, nil, testOut16)
		assert.Empty(t, err)
	}
	// Retrieval 17: node 6 retrieves cid8 from node 8
	routine17 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr6.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 1, len(outs[0]))
		pid, _, _, _, err := pmgr6.GetPeerInfo(0, outs[0][0])
		assert.Empty(t, err)
		// First try to query offer directly from connected peers.
		_, err = cmgr6.QueryOffer(ctx, pid, 0, cid8)
		assert.NotEmpty(t, err)
		// Does not found offer from connected peers. Using DHT.
		cOffers := cmgr6.SearchOffers(ctx, 0, cid8, 3)
		assert.Equal(t, 1, len(cOffers))
		cOffer := cOffers[0]
		// Found one cid offer from DHT, now find a correspoding pay offer to do payment over payment network.
		pOffers, err := pmgr6.SearchPayOffer(ctx, cOffer.CurrencyID(), cOffer.ToAddr(), big.NewInt(0).Mul(big.NewInt(cOffer.Size()), cOffer.PPB()))
		assert.Empty(t, err)
		assert.Equal(t, 1, len(pOffers))
		pOffer := &pOffers[0]
		// Found one pay offer to use, choose it.
		err = cmgr6.Retrieve(ctx, cOffer, pOffer, testOut17)
		assert.Empty(t, err)
	}
	// Retrieval 18: node 7 retrieves cid9 from node 9
	routine18 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr7.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 1, len(outs[0]))
		pid, _, _, _, err := pmgr7.GetPeerInfo(0, outs[0][0])
		assert.Empty(t, err)
		// First try to query offer directly from connected peers.
		_, err = cmgr7.QueryOffer(ctx, pid, 0, cid9)
		assert.NotEmpty(t, err)
		// Does not found offer from connected peers. Using DHT.
		cOffers := cmgr7.SearchOffers(ctx, 0, cid9, 3)
		assert.Equal(t, 1, len(cOffers))
		cOffer := cOffers[0]
		// Found one cid offer from DHT, now find a correspoding pay offer to do payment over payment network.
		pOffers, err := pmgr7.SearchPayOffer(ctx, cOffer.CurrencyID(), cOffer.ToAddr(), big.NewInt(0).Mul(big.NewInt(cOffer.Size()), cOffer.PPB()))
		assert.Empty(t, err)
		assert.Equal(t, 1, len(pOffers))
		pOffer := &pOffers[0]
		// Found one pay offer to use, choose it.
		err = cmgr7.Retrieve(ctx, cOffer, pOffer, testOut18)
		assert.Empty(t, err)
	}
	// Retrieval 19: node 8 retrieves cid9 from node 9
	routine19 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		var err error
		// List all connected peers.
		outs, err := pmgr8.ListActiveOutboundChs(ctx)
		assert.Empty(t, err)
		assert.Equal(t, 2, len(outs[0]))
		for _, out := range outs[0] {
			pid, _, _, _, err := pmgr8.GetPeerInfo(0, out)
			assert.Empty(t, err)
			// First try to query offer directly from connected peers.
			cOffer, err := cmgr8.QueryOffer(ctx, pid, 0, cid9)
			if err != nil {
				continue
			}
			// Found offer from connected peers, retrieve directly.
			err = cmgr8.Retrieve(ctx, *cOffer, nil, testOut19)
			assert.Empty(t, err)
			break
		}
	}

	// Start
	var wg sync.WaitGroup
	wg.Add(1)
	go routine0(&wg)
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
	wg.Add(1)
	go routine6(&wg)
	wg.Add(1)
	go routine7(&wg)
	wg.Add(1)
	go routine8(&wg)
	wg.Add(1)
	go routine9(&wg)
	wg.Add(1)
	go routine10(&wg)
	wg.Add(1)
	go routine11(&wg)
	wg.Add(1)
	go routine12(&wg)
	wg.Add(1)
	go routine13(&wg)
	wg.Add(1)
	go routine14(&wg)
	wg.Add(1)
	go routine15(&wg)
	wg.Add(1)
	go routine16(&wg)
	wg.Add(1)
	go routine17(&wg)
	wg.Add(1)
	go routine18(&wg)
	wg.Add(1)
	go routine19(&wg)
	// Wait for all routines to finish
	wg.Wait()

	// Verify Retrieval 0
	defer os.Remove(testOut0)
	assert.True(t, equalFile(testFile7, testOut0))
	// Verify Retrieval 1
	defer os.Remove(testOut1)
	assert.True(t, equalFile(testFile9, testOut1))
	// Verify Retrieval 2
	defer os.Remove(testOut2)
	assert.True(t, equalFile(testFile5, testOut2))
	// Verify Retrieval 3
	defer os.Remove(testOut3)
	assert.True(t, equalFile(testFile1, testOut3))
	// Verify Retrieval 4
	defer os.Remove(testOut4)
	assert.True(t, equalFile(testFile3, testOut4))
	// Verify Retrieval 5
	defer os.Remove(testOut5)
	assert.True(t, equalFile(testFile0, testOut5))
	// Verify Retrieval 6
	defer os.Remove(testOut6)
	assert.True(t, equalFile(testFile8, testOut6))
	// Verify Retrieval 7
	defer os.Remove(testOut7)
	assert.True(t, equalFile(testFile2, testOut7))
	// Verify Retrieval 8
	defer os.Remove(testOut8)
	assert.True(t, equalFile(testFile4, testOut8))
	// Verify Retrieval 9
	defer os.Remove(testOut9)
	assert.True(t, equalFile(testFile0, testOut9))
	// Verify Retrieval 10
	defer os.Remove(testOut10)
	assert.True(t, equalFile(testFile6, testOut10))
	// Verify Retrieval 11
	defer os.Remove(testOut11)
	assert.True(t, equalFile(testFile3, testOut11))
	// Verify Retrieval 12
	defer os.Remove(testOut12)
	assert.True(t, equalFile(testFile5, testOut12))
	// Verify Retrieval 13
	defer os.Remove(testOut13)
	assert.True(t, equalFile(testFile9, testOut13))
	// Verify Retrieval 14
	defer os.Remove(testOut14)
	assert.True(t, equalFile(testFile5, testOut14))
	// Verify Retrieval 15
	defer os.Remove(testOut15)
	assert.True(t, equalFile(testFile9, testOut15))
	// Verify Retrieval 16
	defer os.Remove(testOut16)
	assert.True(t, equalFile(testFile7, testOut16))
	// Verify Retrieval 17
	defer os.Remove(testOut17)
	assert.True(t, equalFile(testFile8, testOut17))
	// Verify Retrieval 18
	defer os.Remove(testOut18)
	assert.True(t, equalFile(testFile9, testOut18))
	// Verify Retrieval 19
	defer os.Remove(testOut19)
	assert.True(t, equalFile(testFile9, testOut19))

	cancel()
	time.Sleep(3 * time.Second)
}

func makeManager(ctx context.Context, testDB string, testKey string, servID string, servAddr string, testMinerKey string) (CIDNetworkManager, paynetmgr.PaymentNetworkManager, host.Host, peer.AddrInfo, error) {
	conf := config.Config{
		RootPath:     testDB,
		MinerKey:     testMinerKey,
		EnableCur0:   true,
		ServIDCur0:   servID,
		ServAddrCur0: servAddr,
		KeyCur0:      testKey,
		MaxHopCur0:   10,
	}
	var dualDHT *dual.DHT
	h, err := libp2p.New(
		ctx,
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			var err error
			dualDHT, err = dual.New(ctx, h, dual.DHTOption(dht.ProtocolPrefix("/filecoin-retrieval")))
			return dualDHT, err
		}))
	if err != nil {
		return nil, nil, nil, peer.AddrInfo{}, err
	}
	pi := peer.AddrInfo{ID: h.ID(), Addrs: h.Addrs()}
	paynetMgr, err := paynetmgr.NewPaymentNetworkManagerImplV1(ctx, h, pi, conf)
	if err != nil {
		return nil, nil, nil, peer.AddrInfo{}, err
	}
	cidnetMgr, err := NewCIDNetworkManagerImplV1(ctx, h, dualDHT, pi, conf, paynetMgr)
	return cidnetMgr, paynetMgr, h, pi, err
}

func generateRandomFile(out string) error {
	content := make([]byte, 4000000)
	rand.Read(content)
	return os.WriteFile(out, content, os.ModePerm)
}

func equalFile(file1 string, file2 string) bool {
	f1, err1 := ioutil.ReadFile(file1)
	if err1 != nil {
		return false
	}
	f2, err2 := ioutil.ReadFile(file2)
	if err2 != nil {
		return false
	}
	return bytes.Equal(f1, f2)
}
