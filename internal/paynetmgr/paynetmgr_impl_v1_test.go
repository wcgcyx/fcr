package paynetmgr

/*
	@wcgcyx - https://github.com/wcgcyx
*/

import (
	"context"
	"math/big"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/wcgcyx/fcr/internal/config"
	"github.com/wcgcyx/fcr/internal/paymgr/chainmgr"
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
	servAddr := strings.Join(servAddrStrs, sep)
	// New payment managers
	mgr0, pi0, err := makeManager(ctx, testDB0, testPrvKey0, servID, servAddr)
	assert.Empty(t, err)
	mgr1, pi1, err := makeManager(ctx, testDB1, testPrvKey1, servID, servAddr)
	assert.Empty(t, err)
	mgr2, pi2, err := makeManager(ctx, testDB2, testPrvKey2, servID, servAddr)
	assert.Empty(t, err)
	mgr3, pi3, err := makeManager(ctx, testDB3, testPrvKey3, servID, servAddr)
	assert.Empty(t, err)
	mgr4, pi4, err := makeManager(ctx, testDB4, testPrvKey4, servID, servAddr)
	assert.Empty(t, err)
	assert.NotEmpty(t, mgr0.KeyStore())
	assert.NotEmpty(t, mgr0.CurrencyIDs())
	// Get addrs
	addr0, err := mgr0.GetRootAddress(0)
	assert.Empty(t, err)
	addr1, err := mgr1.GetRootAddress(0)
	assert.Empty(t, err)
	addr2, err := mgr2.GetRootAddress(0)
	assert.Empty(t, err)
	addr3, err := mgr3.GetRootAddress(0)
	assert.Empty(t, err)
	addr4, err := mgr4.GetRootAddress(0)
	assert.Empty(t, err)
	t.Log("mgr0:", addr0, pi0.ID.String())
	t.Log("mgr1:", addr1, pi1.ID.String())
	t.Log("mgr2:", addr2, pi2.ID.String())
	t.Log("mgr3:", addr3, pi3.ID.String())
	t.Log("mgr4:", addr4, pi4.ID.String())

	// Creating payment channels.
	offer, err := mgr0.QueryOutboundChOffer(ctx, 0, addr2, pi2)
	assert.Empty(t, err)
	err = mgr0.CreateOutboundCh(ctx, offer, big.NewInt(1e18), pi2)
	assert.Empty(t, err)

	offer, err = mgr1.QueryOutboundChOffer(ctx, 0, addr2, pi2)
	assert.Empty(t, err)
	err = mgr1.CreateOutboundCh(ctx, offer, big.NewInt(1e18), pi2)
	assert.Empty(t, err)

	offer, err = mgr2.QueryOutboundChOffer(ctx, 0, addr3, pi3)
	assert.Empty(t, err)
	err = mgr2.CreateOutboundCh(ctx, offer, big.NewInt(1e18), pi3)
	assert.Empty(t, err)

	offer, err = mgr3.QueryOutboundChOffer(ctx, 0, addr4, pi4)
	assert.Empty(t, err)
	err = mgr3.CreateOutboundCh(ctx, offer, big.NewInt(1e18), pi4)
	assert.Empty(t, err)

	// Start serving.
	err = mgr3.StartServing(0, addr4, big.NewInt(10), big.NewInt(200))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)
	mgr2.StartServing(0, addr3, big.NewInt(5), big.NewInt(100))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)

	// There will be 5 concurrent routine.
	// Routine 0 -> 2 for 150 units.
	routine0 := func(wg *sync.WaitGroup) {
		// It will take in total 5 seconds.
		defer wg.Done()
		err = mgr0.Reserve(0, addr2, big.NewInt(150))
		assert.Empty(t, err)
		for i := 0; i < 5; i++ {
			err = mgr0.Pay(ctx, 0, addr2, big.NewInt(30))
			assert.Empty(t, err)
			time.Sleep(1 * time.Second)
		}
	}
	// Routine 0 -> 2 -> 3 -> 4 for 1130 units.
	routine1 := func(wg *sync.WaitGroup) {
		// It will take in total 3 seconds.
		defer wg.Done()
		err = mgr0.Reserve(0, addr4, big.NewInt(1130))
		assert.NotEmpty(t, err)
		offers, err := mgr0.SearchPayOffer(ctx, 0, addr4, big.NewInt(1130))
		assert.Empty(t, err)
		assert.Equal(t, 1, len(offers))
		offer := &offers[0]
		err = mgr0.ReserveWithPayOffer(offer)
		assert.Empty(t, err)
		for i := 0; i < 6; i++ {
			if i == 5 {
				err = mgr0.PayWithOffer(ctx, offer, big.NewInt(30))
			} else {
				err = mgr0.PayWithOffer(ctx, offer, big.NewInt(220))
			}
			assert.Empty(t, err)
			time.Sleep(500 * time.Millisecond)
		}
	}
	// Routine 1 -> 2 -> 3 -> 4 for 1030 units.
	routine2 := func(wg *sync.WaitGroup) {
		// It will take in total 4 seconds.
		defer wg.Done()
		err = mgr1.Reserve(0, addr4, big.NewInt(1030))
		assert.NotEmpty(t, err)
		offers, err := mgr1.SearchPayOffer(ctx, 0, addr4, big.NewInt(1030))
		assert.Empty(t, err)
		assert.Equal(t, 1, len(offers))
		offer := &offers[0]
		err = mgr1.ReserveWithPayOffer(offer)
		assert.Empty(t, err)
		for i := 0; i < 206; i++ {
			err = mgr1.PayWithOffer(ctx, offer, big.NewInt(5))
			assert.Empty(t, err)
			time.Sleep(20 * time.Millisecond)
		}
	}
	// Routine 1 -> 2 -> 3 for 520 units.
	routine3 := func(wg *sync.WaitGroup) {
		// It will take in total 2 seconds.
		defer wg.Done()
		err = mgr1.Reserve(0, addr3, big.NewInt(520))
		assert.NotEmpty(t, err)
		offers, err := mgr1.SearchPayOffer(ctx, 0, addr3, big.NewInt(520))
		assert.Empty(t, err)
		assert.Equal(t, 1, len(offers))
		offer := &offers[0]
		err = mgr1.ReserveWithPayOffer(offer)
		for i := 0; i < 20; i++ {
			err = mgr1.PayWithOffer(ctx, offer, big.NewInt(26))
			assert.Empty(t, err)
			time.Sleep(100 * time.Millisecond)
		}
	}
	// Routine 2 -> 3 for 3400 units.
	routine4 := func(wg *sync.WaitGroup) {
		// It will take in total 2 seconds.
		defer wg.Done()
		err = mgr2.Reserve(0, addr3, big.NewInt(3400))
		assert.Empty(t, err)
		for i := 0; i < 100; i++ {
			err = mgr2.Pay(ctx, 0, addr3, big.NewInt(34))
			assert.Empty(t, err)
			time.Sleep(20 * time.Millisecond)
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
	// Wait for all routines to finish
	wg.Wait()
	// Verify 0 -> 2 for 150 units
	amt, err := mgr2.Receive(0, addr0)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(150), amt)
	// Verify 0 -> 4 for 1130 units
	amt, err = mgr4.Receive(0, addr0)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1130), amt)
	// Verify 1 -> 4 for 1030 units
	amt, err = mgr4.Receive(0, addr1)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1030), amt)
	// Verify 1 -> 3 for 520 units
	amt, err = mgr3.Receive(0, addr1)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(520), amt)
	// Verify 2 -> 3 for 3400 units
	amt, err = mgr3.Receive(0, addr2)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(3400), amt)

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
	servAddr := strings.Join(servAddrStrs, sep)
	// New payment managers
	mgr0, pi0, err := makeManager(ctx, testDB0, testPrvKey0, servID, servAddr)
	assert.Empty(t, err)
	mgr1, pi1, err := makeManager(ctx, testDB1, testPrvKey1, servID, servAddr)
	assert.Empty(t, err)
	mgr2, pi2, err := makeManager(ctx, testDB2, testPrvKey2, servID, servAddr)
	assert.Empty(t, err)
	mgr3, pi3, err := makeManager(ctx, testDB3, testPrvKey3, servID, servAddr)
	assert.Empty(t, err)
	mgr4, pi4, err := makeManager(ctx, testDB4, testPrvKey4, servID, servAddr)
	assert.Empty(t, err)
	mgr5, pi5, err := makeManager(ctx, testDB5, testPrvKey5, servID, servAddr)
	assert.Empty(t, err)
	mgr6, pi6, err := makeManager(ctx, testDB6, testPrvKey6, servID, servAddr)
	assert.Empty(t, err)
	mgr7, pi7, err := makeManager(ctx, testDB7, testPrvKey7, servID, servAddr)
	assert.Empty(t, err)
	mgr8, pi8, err := makeManager(ctx, testDB8, testPrvKey8, servID, servAddr)
	assert.Empty(t, err)
	mgr9, pi9, err := makeManager(ctx, testDB9, testPrvKey9, servID, servAddr)
	assert.Empty(t, err)
	// Get addrs
	addr0, err := mgr0.GetRootAddress(0)
	assert.Empty(t, err)
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
	addr6, err := mgr6.GetRootAddress(0)
	assert.Empty(t, err)
	addr7, err := mgr7.GetRootAddress(0)
	assert.Empty(t, err)
	addr8, err := mgr8.GetRootAddress(0)
	assert.Empty(t, err)
	addr9, err := mgr9.GetRootAddress(0)
	assert.Empty(t, err)
	t.Log("mgr0:", addr0, pi0.ID.String())
	t.Log("mgr1:", addr1, pi1.ID.String())
	t.Log("mgr2:", addr2, pi2.ID.String())
	t.Log("mgr3:", addr3, pi3.ID.String())
	t.Log("mgr4:", addr4, pi4.ID.String())
	t.Log("mgr5:", addr5, pi5.ID.String())
	t.Log("mgr6:", addr6, pi6.ID.String())
	t.Log("mgr7:", addr7, pi7.ID.String())
	t.Log("mgr8:", addr8, pi8.ID.String())
	t.Log("mgr9:", addr9, pi9.ID.String())

	// Creating payment channels.
	offer, err := mgr0.QueryOutboundChOffer(ctx, 0, addr1, pi1)
	assert.Empty(t, err)
	err = mgr0.CreateOutboundCh(ctx, offer, big.NewInt(1e18), pi1)
	assert.Empty(t, err)

	offer, err = mgr1.QueryOutboundChOffer(ctx, 0, addr2, pi2)
	assert.Empty(t, err)
	err = mgr1.CreateOutboundCh(ctx, offer, big.NewInt(1e18), pi2)
	assert.Empty(t, err)

	offer, err = mgr1.QueryOutboundChOffer(ctx, 0, addr4, pi4)
	assert.Empty(t, err)
	err = mgr1.CreateOutboundCh(ctx, offer, big.NewInt(1e18), pi4)
	assert.Empty(t, err)

	offer, err = mgr1.QueryOutboundChOffer(ctx, 0, addr6, pi6)
	assert.Empty(t, err)
	err = mgr1.CreateOutboundCh(ctx, offer, big.NewInt(1e18), pi6)
	assert.Empty(t, err)

	offer, err = mgr2.QueryOutboundChOffer(ctx, 0, addr3, pi3)
	assert.Empty(t, err)
	err = mgr2.CreateOutboundCh(ctx, offer, big.NewInt(1e18), pi3)
	assert.Empty(t, err)

	offer, err = mgr3.QueryOutboundChOffer(ctx, 0, addr0, pi0)
	assert.Empty(t, err)
	err = mgr3.CreateOutboundCh(ctx, offer, big.NewInt(1e18), pi0)
	assert.Empty(t, err)

	offer, err = mgr4.QueryOutboundChOffer(ctx, 0, addr5, pi5)
	assert.Empty(t, err)
	err = mgr4.CreateOutboundCh(ctx, offer, big.NewInt(1e18), pi5)
	assert.Empty(t, err)

	offer, err = mgr4.QueryOutboundChOffer(ctx, 0, addr7, pi7)
	assert.Empty(t, err)
	err = mgr4.CreateOutboundCh(ctx, offer, big.NewInt(1e18), pi7)
	assert.Empty(t, err)

	offer, err = mgr6.QueryOutboundChOffer(ctx, 0, addr7, pi7)
	assert.Empty(t, err)
	err = mgr6.CreateOutboundCh(ctx, offer, big.NewInt(1e18), pi7)
	assert.Empty(t, err)

	offer, err = mgr7.QueryOutboundChOffer(ctx, 0, addr8, pi8)
	assert.Empty(t, err)
	err = mgr7.CreateOutboundCh(ctx, offer, big.NewInt(1e18), pi8)
	assert.Empty(t, err)

	offer, err = mgr8.QueryOutboundChOffer(ctx, 0, addr9, pi9)
	assert.Empty(t, err)
	err = mgr8.CreateOutboundCh(ctx, offer, big.NewInt(1e18), pi9)
	assert.Empty(t, err)

	offer, err = mgr8.QueryOutboundChOffer(ctx, 0, addr5, pi5)
	assert.Empty(t, err)
	err = mgr8.CreateOutboundCh(ctx, offer, big.NewInt(1e18), pi5)
	assert.Empty(t, err)

	// Start serving.
	err = mgr8.StartServing(0, addr9, big.NewInt(10), big.NewInt(200))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)
	err = mgr8.StartServing(0, addr5, big.NewInt(10), big.NewInt(200))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)
	err = mgr7.StartServing(0, addr8, big.NewInt(10), big.NewInt(200))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)
	err = mgr4.StartServing(0, addr7, big.NewInt(5), big.NewInt(300))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)
	err = mgr4.StartServing(0, addr5, big.NewInt(5), big.NewInt(300))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)
	err = mgr6.StartServing(0, addr7, big.NewInt(25), big.NewInt(100))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)
	err = mgr1.StartServing(0, addr6, big.NewInt(5), big.NewInt(200))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)
	err = mgr1.StartServing(0, addr4, big.NewInt(5), big.NewInt(200))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)
	err = mgr0.StartServing(0, addr1, big.NewInt(20), big.NewInt(300))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)
	err = mgr3.StartServing(0, addr0, big.NewInt(13), big.NewInt(150))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)
	err = mgr2.StartServing(0, addr3, big.NewInt(17), big.NewInt(150))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)
	err = mgr1.StartServing(0, addr2, big.NewInt(19), big.NewInt(150))
	assert.Empty(t, err)
	time.Sleep(100 * time.Millisecond)

	// Force publish two times to sync every node.
	for i := 0; i < 2; i++ {
		mgr0.ForcePublishServings()
		time.Sleep(10 * time.Millisecond)
		mgr1.ForcePublishServings()
		time.Sleep(10 * time.Millisecond)
		mgr2.ForcePublishServings()
		time.Sleep(10 * time.Millisecond)
		mgr3.ForcePublishServings()
		time.Sleep(10 * time.Millisecond)
		mgr4.ForcePublishServings()
		time.Sleep(10 * time.Millisecond)
		mgr5.ForcePublishServings()
		time.Sleep(10 * time.Millisecond)
		mgr6.ForcePublishServings()
		time.Sleep(10 * time.Millisecond)
		mgr7.ForcePublishServings()
		time.Sleep(10 * time.Millisecond)
		mgr8.ForcePublishServings()
		time.Sleep(10 * time.Millisecond)
		mgr9.ForcePublishServings()
		time.Sleep(10 * time.Millisecond)
	}

	// There will be 20 concurrent payment routines.
	// Routine 0: 0 -> 7 for 1001 units of payment.
	routine0 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = mgr0.Reserve(0, addr7, big.NewInt(1001))
		assert.NotEmpty(t, err)
		offers, err := mgr0.SearchPayOffer(ctx, 0, addr7, big.NewInt(1001))
		assert.Empty(t, err)
		// There will be 2 offers, one offer to pay 500, one offer to pay 501
		assert.Equal(t, 2, len(offers))
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			if len(offers) > 0 {
				offer := &offers[0]
				err = mgr0.ReserveWithPayOffer(offer)
				assert.Empty(t, err)
				for i := 0; i < 5; i++ {
					err = mgr0.PayWithOffer(ctx, offer, big.NewInt(100))
					assert.Empty(t, err)
					// 1 second in total
					time.Sleep(200 * time.Millisecond)
				}
			}
		}(wg)
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			if len(offers) > 1 {
				offer := &offers[1]
				err = mgr0.ReserveWithPayOffer(offer)
				assert.Empty(t, err)
				for i := 0; i < 100; i++ {
					if i == 99 {
						err = mgr0.PayWithOffer(ctx, offer, big.NewInt(6))
					} else {
						err = mgr0.PayWithOffer(ctx, offer, big.NewInt(5))
					}
					assert.Empty(t, err)
					// 2 second in total
					time.Sleep(20 * time.Millisecond)
				}
			}
		}(wg)
	}
	// Routine 1: 0 -> 9 for 1002 units of payment.
	routine1 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = mgr0.Reserve(0, addr9, big.NewInt(1002))
		assert.NotEmpty(t, err)
		offers, err := mgr0.SearchPayOffer(ctx, 0, addr9, big.NewInt(1002))
		assert.Empty(t, err)
		// There will be 2 offers, one offer to pay 500, one offer to pay 502
		assert.Equal(t, 2, len(offers))
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			if len(offers) > 0 {
				offer := &offers[0]
				err = mgr0.ReserveWithPayOffer(offer)
				assert.Empty(t, err)
				for i := 0; i < 10; i++ {
					err = mgr0.PayWithOffer(ctx, offer, big.NewInt(50))
					assert.Empty(t, err)
					// 1.5 second in total
					time.Sleep(150 * time.Millisecond)
				}
			}
		}(wg)
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			if len(offers) > 1 {
				offer := &offers[1]
				err = mgr0.ReserveWithPayOffer(offer)
				assert.Empty(t, err)
				for i := 0; i < 50; i++ {
					if i == 49 {
						err = mgr0.PayWithOffer(ctx, offer, big.NewInt(12))
					} else {
						err = mgr0.PayWithOffer(ctx, offer, big.NewInt(10))
					}
					assert.Empty(t, err)
					// 1 second in total
					time.Sleep(20 * time.Millisecond)
				}
			}
		}(wg)
	}
	// Routine 2: 0 -> 5 for 1003 units of payment.
	routine2 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = mgr0.Reserve(0, addr5, big.NewInt(1003))
		assert.NotEmpty(t, err)
		offers, err := mgr0.SearchPayOffer(ctx, 0, addr5, big.NewInt(1003))
		assert.Empty(t, err)
		// There will be 2 offers, one offer to pay 500, one offer to pay 503
		assert.Equal(t, 2, len(offers))
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			if len(offers) > 0 {
				offer := &offers[0]
				err = mgr0.ReserveWithPayOffer(offer)
				assert.Empty(t, err)
				for i := 0; i < 250; i++ {
					err = mgr0.PayWithOffer(ctx, offer, big.NewInt(2))
					assert.Empty(t, err)
					// 2.5 second in total
					time.Sleep(10 * time.Millisecond)
				}
			}
		}(wg)
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			if len(offers) > 1 {
				offer := &offers[1]
				err = mgr0.ReserveWithPayOffer(offer)
				assert.Empty(t, err)
				for i := 0; i < 3; i++ {
					if i == 2 {
						err = mgr0.PayWithOffer(ctx, offer, big.NewInt(103))
					} else {
						err = mgr0.PayWithOffer(ctx, offer, big.NewInt(200))
					}
					assert.Empty(t, err)
					// 1.5 second in total
					time.Sleep(500 * time.Millisecond)
				}
			}
		}(wg)
	}
	// Routine 3: 0 -> 1 for 1004 units of payment.
	routine3 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = mgr0.Reserve(0, addr1, big.NewInt(1004))
		assert.Empty(t, err)
		for i := 0; i < 25; i++ {
			if i == 24 {
				err = mgr0.Pay(ctx, 0, addr1, big.NewInt(44))
			} else {
				err = mgr0.Pay(ctx, 0, addr1, big.NewInt(40))
			}
			assert.Empty(t, err)
			// 0.25 second in total
			time.Sleep(10 * time.Millisecond)
		}
	}
	// Routine 4: 0 -> 3 for 1005 units of payment.
	routine4 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = mgr0.Reserve(0, addr3, big.NewInt(1005))
		assert.NotEmpty(t, err)
		offers, err := mgr0.SearchPayOffer(ctx, 0, addr3, big.NewInt(1005))
		assert.Empty(t, err)
		// There will be 1 offer.
		assert.Equal(t, 1, len(offers))
		if len(offers) > 0 {
			offer := &offers[0]
			err = mgr0.ReserveWithPayOffer(offer)
			assert.Empty(t, err)
			for i := 0; i < 201; i++ {
				err = mgr0.PayWithOffer(ctx, offer, big.NewInt(5))
				assert.Empty(t, err)
				// 2 seconds in total
				time.Sleep(10 * time.Millisecond)
			}
		}
	}
	// Routine 5: 1 -> 2 for 1006 units of payment.
	routine5 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = mgr1.Reserve(0, addr2, big.NewInt(1006))
		assert.Empty(t, err)
		for i := 0; i < 100; i++ {
			if i == 99 {
				err = mgr1.Pay(ctx, 0, addr2, big.NewInt(16))
			} else {
				err = mgr1.Pay(ctx, 0, addr2, big.NewInt(10))
			}
			assert.Empty(t, err)
			// 1 second in total
			time.Sleep(10 * time.Millisecond)
		}
	}
	// Routine 6: 1 -> 4 for 1007 units of payment.
	routine6 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = mgr1.Reserve(0, addr4, big.NewInt(1007))
		assert.Empty(t, err)
		for i := 0; i < 200; i++ {
			if i == 199 {
				err = mgr1.Pay(ctx, 0, addr4, big.NewInt(12))
			} else {
				err = mgr1.Pay(ctx, 0, addr4, big.NewInt(5))
			}
			assert.Empty(t, err)
			// 2 seconds in total
			time.Sleep(10 * time.Millisecond)
		}
	}
	// Routine 7: 1 -> 0 for 1008 units of payment.
	routine7 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = mgr1.Reserve(0, addr0, big.NewInt(1008))
		assert.NotEmpty(t, err)
		offers, err := mgr1.SearchPayOffer(ctx, 0, addr0, big.NewInt(1008))
		assert.Empty(t, err)
		// There will be 1 offer.
		assert.Equal(t, 1, len(offers))
		if len(offers) > 0 {
			offer := &offers[0]
			err = mgr1.ReserveWithPayOffer(offer)
			assert.Empty(t, err)
			for i := 0; i < 33; i++ {
				if i == 32 {
					err = mgr1.PayWithOffer(ctx, offer, big.NewInt(48))
				} else {
					err = mgr1.PayWithOffer(ctx, offer, big.NewInt(30))
				}
				assert.Empty(t, err)
				// 0.6 second in total
				time.Sleep(20 * time.Millisecond)
			}
		}
	}
	// Routine 8: 1 -> 9 for 1009 units of payment.
	routine8 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = mgr1.Reserve(0, addr9, big.NewInt(1009))
		assert.NotEmpty(t, err)
		offers, err := mgr1.SearchPayOffer(ctx, 0, addr9, big.NewInt(1009))
		assert.Empty(t, err)
		// There will be 2 offers, one offer to pay 500, one offer to pay 509
		assert.Equal(t, 2, len(offers))
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			if len(offers) > 0 {
				offer := &offers[0]
				err = mgr1.ReserveWithPayOffer(offer)
				assert.Empty(t, err)
				for i := 0; i < 250; i++ {
					err = mgr1.PayWithOffer(ctx, offer, big.NewInt(2))
					assert.Empty(t, err)
					// 1 second in total
					time.Sleep(5 * time.Millisecond)
				}
			}
		}(wg)
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			if len(offers) > 1 {
				offer := &offers[1]
				err = mgr1.ReserveWithPayOffer(offer)
				assert.Empty(t, err)
				for i := 0; i < 250; i++ {
					if i == 249 {
						err = mgr1.PayWithOffer(ctx, offer, big.NewInt(11))
					} else {
						err = mgr1.PayWithOffer(ctx, offer, big.NewInt(2))
					}
					assert.Empty(t, err)
					// 1 second in total
					time.Sleep(5 * time.Millisecond)
				}
			}
		}(wg)
	}
	// Routine 9: 2 -> 3 for 1010 units of payment.
	routine9 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = mgr2.Reserve(0, addr3, big.NewInt(1010))
		assert.Empty(t, err)
		for i := 0; i < 1010; i++ {
			err = mgr2.Pay(ctx, 0, addr3, big.NewInt(1))
			assert.Empty(t, err)
			// 1 second in total
			time.Sleep(1 * time.Millisecond)
		}
	}
	// Routine 10: 2 -> 5 for 1011 units of payment.
	routine10 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = mgr2.Reserve(0, addr5, big.NewInt(1011))
		assert.NotEmpty(t, err)
		offers, err := mgr2.SearchPayOffer(ctx, 0, addr5, big.NewInt(1011))
		assert.Empty(t, err)
		// There will be 2 offers, one offer to pay 500, one offer to pay 511
		assert.Equal(t, 2, len(offers))
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			if len(offers) > 0 {
				offer := &offers[0]
				err = mgr2.ReserveWithPayOffer(offer)
				assert.Empty(t, err)
				for i := 0; i < 20; i++ {
					err = mgr2.PayWithOffer(ctx, offer, big.NewInt(25))
					assert.Empty(t, err)
					// 2 seconds in total
					time.Sleep(100 * time.Millisecond)
				}
			}
		}(wg)
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			if len(offers) > 1 {
				offer := &offers[1]
				err = mgr2.ReserveWithPayOffer(offer)
				assert.Empty(t, err)
				for i := 0; i < 100; i++ {
					if i == 99 {
						err = mgr2.PayWithOffer(ctx, offer, big.NewInt(16))
					} else {
						err = mgr2.PayWithOffer(ctx, offer, big.NewInt(5))
					}
					assert.Empty(t, err)
					// 1 second in total
					time.Sleep(10 * time.Millisecond)
				}
			}
		}(wg)
	}
	// Routine 11: 2 -> 9 for 1012 units of payment.
	routine11 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = mgr2.Reserve(0, addr9, big.NewInt(1012))
		assert.NotEmpty(t, err)
		offers, err := mgr2.SearchPayOffer(ctx, 0, addr9, big.NewInt(1012))
		assert.Empty(t, err)
		// There will be 2 offers, one offer to pay 500, one offer to pay 512
		assert.Equal(t, 2, len(offers))
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			if len(offers) > 0 {
				offer := &offers[0]
				err = mgr2.ReserveWithPayOffer(offer)
				assert.Empty(t, err)
				for i := 0; i < 25; i++ {
					err = mgr2.PayWithOffer(ctx, offer, big.NewInt(20))
					assert.Empty(t, err)
					// 2.5 seconds in total
					time.Sleep(100 * time.Millisecond)
				}
			}
		}(wg)
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			if len(offers) > 1 {
				offer := &offers[1]
				err = mgr2.ReserveWithPayOffer(offer)
				assert.Empty(t, err)
				for i := 0; i < 100; i++ {
					if i == 99 {
						err = mgr2.PayWithOffer(ctx, offer, big.NewInt(17))
					} else {
						err = mgr2.PayWithOffer(ctx, offer, big.NewInt(5))
					}
					assert.Empty(t, err)
					// 1 second in total
					time.Sleep(10 * time.Millisecond)
				}
			}
		}(wg)
	}
	// Routine 12: 3 -> 0 for 1013 units of payment.
	routine12 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = mgr3.Reserve(0, addr0, big.NewInt(1013))
		assert.Empty(t, err)
		for i := 0; i < 100; i++ {
			if i == 99 {
				err = mgr3.Pay(ctx, 0, addr0, big.NewInt(23))
			} else {
				err = mgr3.Pay(ctx, 0, addr0, big.NewInt(10))
			}
			assert.Empty(t, err)
			// 2 seconds in total
			time.Sleep(20 * time.Millisecond)
		}
	}
	// Routine 13: 3 -> 8 for 1014 units of payment.
	routine13 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = mgr3.Reserve(0, addr8, big.NewInt(1014))
		assert.NotEmpty(t, err)
		offers, err := mgr3.SearchPayOffer(ctx, 0, addr8, big.NewInt(1014))
		assert.Empty(t, err)
		// There will be 2 offers, one offer to pay 500, one offer to pay 514
		assert.Equal(t, 2, len(offers))
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			if len(offers) > 0 {
				offer := &offers[0]
				err = mgr3.ReserveWithPayOffer(offer)
				assert.Empty(t, err)
				for i := 0; i < 50; i++ {
					err = mgr3.PayWithOffer(ctx, offer, big.NewInt(10))
					assert.Empty(t, err)
					// 2.5 seconds in total
					time.Sleep(50 * time.Millisecond)
				}
			}
		}(wg)
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			if len(offers) > 1 {
				offer := &offers[1]
				err = mgr3.ReserveWithPayOffer(offer)
				assert.Empty(t, err)
				for i := 0; i < 20; i++ {
					if i == 19 {
						err = mgr3.PayWithOffer(ctx, offer, big.NewInt(39))
					} else {
						err = mgr3.PayWithOffer(ctx, offer, big.NewInt(25))
					}
					assert.Empty(t, err)
					// 2 seconds in total
					time.Sleep(100 * time.Millisecond)
				}
			}
		}(wg)
	}
	// Routine 14: 4 -> 5 for 1015 units of payment.
	routine14 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = mgr4.Reserve(0, addr5, big.NewInt(1015))
		assert.Empty(t, err)
		for i := 0; i < 203; i++ {
			err = mgr4.Pay(ctx, 0, addr5, big.NewInt(5))
			assert.Empty(t, err)
			// 2 seconds in total
			time.Sleep(10 * time.Millisecond)
		}
	}
	// Routine 15: 4 -> 9 for 1016 units of payment.
	routine15 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = mgr4.Reserve(0, addr9, big.NewInt(1016))
		assert.NotEmpty(t, err)
		offers, err := mgr4.SearchPayOffer(ctx, 0, addr9, big.NewInt(1016))
		assert.Empty(t, err)
		// There will be 1 offer.
		assert.Equal(t, 1, len(offers))
		if len(offers) > 0 {
			offer := &offers[0]
			err = mgr4.ReserveWithPayOffer(offer)
			assert.Empty(t, err)
			for i := 0; i < 60; i++ {
				if i == 59 {
					err = mgr4.PayWithOffer(ctx, offer, big.NewInt(72))
				} else {
					err = mgr4.PayWithOffer(ctx, offer, big.NewInt(16))
				}
				assert.Empty(t, err)
				// 2 seconds in total
				time.Sleep(30 * time.Millisecond)
			}
		}
	}
	// Routine 16: 6 -> 7 for 1017 units of payment.
	routine16 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = mgr6.Reserve(0, addr7, big.NewInt(1017))
		assert.Empty(t, err)
		for i := 0; i < 10; i++ {
			if i == 9 {
				err = mgr6.Pay(ctx, 0, addr7, big.NewInt(117))
			} else {
				err = mgr6.Pay(ctx, 0, addr7, big.NewInt(100))
			}
			assert.Empty(t, err)
			// 1 second in total
			time.Sleep(100 * time.Millisecond)
		}
	}
	// Routine 17: 6 -> 8 for 1018 units of payment.
	routine17 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = mgr6.Reserve(0, addr8, big.NewInt(1018))
		assert.NotEmpty(t, err)
		offers, err := mgr6.SearchPayOffer(ctx, 0, addr8, big.NewInt(1018))
		assert.Empty(t, err)
		// There will be 1 offer.
		assert.Equal(t, 1, len(offers))
		if len(offers) > 0 {
			offer := &offers[0]
			err = mgr6.ReserveWithPayOffer(offer)
			assert.Empty(t, err)
			for i := 0; i < 50; i++ {
				if i == 49 {
					err = mgr6.PayWithOffer(ctx, offer, big.NewInt(38))
				} else {
					err = mgr6.PayWithOffer(ctx, offer, big.NewInt(20))
				}
				assert.Empty(t, err)
				// 1.5 second in total
				time.Sleep(30 * time.Millisecond)
			}
		}
	}
	// Routine 18: 7 -> 9 for 1019 units of payment.
	routine18 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = mgr7.Reserve(0, addr9, big.NewInt(1019))
		assert.NotEmpty(t, err)
		offers, err := mgr7.SearchPayOffer(ctx, 0, addr9, big.NewInt(1019))
		assert.Empty(t, err)
		// There will be 1 offer.
		assert.Equal(t, 1, len(offers))
		if len(offers) > 0 {
			offer := &offers[0]
			err = mgr7.ReserveWithPayOffer(offer)
			assert.Empty(t, err)
			for i := 0; i < 500; i++ {
				if i == 499 {
					err = mgr7.PayWithOffer(ctx, offer, big.NewInt(21))
				} else {
					err = mgr7.PayWithOffer(ctx, offer, big.NewInt(2))
				}
				assert.Empty(t, err)
				// 2.5 seconds in total
				time.Sleep(5 * time.Millisecond)
			}
		}
	}
	// Routine 19: 8 -> 9 for 1020 units of payment.
	routine19 := func(wg *sync.WaitGroup) {
		defer wg.Done()
		err = mgr8.Reserve(0, addr9, big.NewInt(1020))
		assert.Empty(t, err)
		for i := 0; i < 102; i++ {
			err = mgr8.Pay(ctx, 0, addr9, big.NewInt(10))
			assert.Empty(t, err)
			// 3 seconds in total
			time.Sleep(30 * time.Millisecond)
		}
	}

	// Start routines
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

	// Verify Routine 0: 0 -> 7 for 1001 units of payment.
	amt, err := mgr7.Receive(0, addr0)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1001), amt)
	// Verify Routine 1: 0 -> 9 for 1002 units of payment.
	amt, err = mgr9.Receive(0, addr0)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1002), amt)
	// Verify Routine 2: 0 -> 5 for 1003 units of payment.
	amt, err = mgr5.Receive(0, addr0)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1003), amt)
	// Verify Routine 3: 0 -> 1 for 1004 units of payment.
	amt, err = mgr1.Receive(0, addr0)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1004), amt)
	// Verify Routine 4: 0 -> 3 for 1005 units of payment.
	amt, err = mgr3.Receive(0, addr0)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1005), amt)
	// Verify Routine 5: 1 -> 2 for 1006 units of payment.
	amt, err = mgr2.Receive(0, addr1)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1006), amt)
	// Verify Routine 6: 1 -> 4 for 1007 units of payment.
	amt, err = mgr4.Receive(0, addr1)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1007), amt)
	// Verify Routine 7: 1 -> 0 for 1008 units of payment.
	amt, err = mgr0.Receive(0, addr1)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1008), amt)
	// Verify Routine 8: 1 -> 9 for 1009 units of payment.
	amt, err = mgr9.Receive(0, addr1)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1009), amt)
	// Verify Routine 9: 2 -> 3 for 1010 units of payment.
	amt, err = mgr3.Receive(0, addr2)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1010), amt)
	// Verify Routine 10: 2 -> 5 for 1011 units of payment.
	amt, err = mgr5.Receive(0, addr2)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1011), amt)
	// Verify Routine 11: 2 -> 9 for 1012 units of payment.
	amt, err = mgr9.Receive(0, addr2)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1012), amt)
	// Verify Routine 12: 3 -> 0 for 1013 units of payment.
	amt, err = mgr0.Receive(0, addr3)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1013), amt)
	// Verify Routine 13: 3 -> 8 for 1014 units of payment.
	amt, err = mgr8.Receive(0, addr3)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1014), amt)
	// Verify Routine 14: 4 -> 5 for 1015 units of payment.
	amt, err = mgr5.Receive(0, addr4)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1015), amt)
	// Verify Routine 15: 4 -> 9 for 1016 units of payment.
	amt, err = mgr9.Receive(0, addr4)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1016), amt)
	// Verify Routine 16: 6 -> 7 for 1017 units of payment.
	amt, err = mgr7.Receive(0, addr6)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1017), amt)
	// Verify Routine 17: 6 -> 8 for 1018 units of payment.
	amt, err = mgr8.Receive(0, addr6)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1018), amt)
	// Verify Routine 18: 7 -> 9 for 1019 units of payment.
	amt, err = mgr9.Receive(0, addr7)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1019), amt)
	// Verify Routine 19: 8 -> 9 for 1020 units of payment.
	amt, err = mgr9.Receive(0, addr8)
	assert.Empty(t, err)
	assert.Equal(t, big.NewInt(1020), amt)

	cancel()
	time.Sleep(3 * time.Second)
}

func makeManager(ctx context.Context, testDB string, testKey string, servID string, servAddr string) (PaymentNetworkManager, peer.AddrInfo, error) {
	conf := config.Config{
		RootPath:     testDB,
		EnableCur0:   true,
		ServIDCur0:   servID,
		ServAddrCur0: servAddr,
		KeyCur0:      testKey,
		MaxHopCur0:   10,
	}
	h, err := libp2p.New(ctx)
	if err != nil {
		return nil, peer.AddrInfo{}, err
	}
	pi := peer.AddrInfo{ID: h.ID(), Addrs: h.Addrs()}
	mgr, err := NewPaymentNetworkManagerImplV1(ctx, h, pi, conf)
	return mgr, pi, err
}
