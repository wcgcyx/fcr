package paymgr

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
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/wcgcyx/fcr/paychstate"
)

// syncRoutine is the routine for sync operation.
func (mgr *PaymentManagerImpl) syncRoutine() {
	for {
		after := time.After(mgr.cacheSyncFreq)
		select {
		case <-after:
			// Start sync.
			log.Debugf("Sync routine - Start syncing")
			wg0 := sync.WaitGroup{}
			wg0.Add(1)
			go func() {
				defer wg0.Done()
				release0, err := mgr.activeLock.RLock(mgr.routineCtx, false)
				if err != nil {
					log.Warnf("Sync routine - Fail to obtain read lock for %v: %v", false, err.Error())
					return
				}
				defer release0()
				wg1 := sync.WaitGroup{}
				for currencyID := range mgr.active[false] {
					wg1.Add(1)
					go func(currencyID byte) {
						defer wg1.Done()
						release1, err := mgr.activeLock.RLock(mgr.routineCtx, false, currencyID)
						if err != nil {
							log.Warnf("Sync routine - Fail to obtain read lock for %v-%v: %v", false, currencyID, err.Error())
							return
						}
						defer release1()
						wg2 := sync.WaitGroup{}
						for fromAddr := range mgr.active[false][currencyID] {
							wg2.Add(1)
							go func(fromAddr string) {
								defer wg2.Done()
								release2, err := mgr.activeLock.RLock(mgr.routineCtx, false, currencyID, fromAddr)
								if err != nil {
									log.Warnf("Sync routine - Fail to obtain read lock for %v-%v-%v: %v", false, currencyID, fromAddr, err.Error())
									return
								}
								defer release2()
								wg3 := sync.WaitGroup{}
								for chAddr := range mgr.active[false][currencyID][fromAddr] {
									wg3.Add(1)
									go func(chAddr string) {
										defer wg3.Done()
										release3, err := mgr.activeLock.RLock(mgr.routineCtx, false, currencyID, fromAddr, chAddr)
										if err != nil {
											log.Warnf("Sync routine - Fail to obtain read lock for %v-%v-%v-%v: %v", false, currencyID, fromAddr, chAddr, err.Error())
											return
										}
										defer release3()
										// Get state now.
										cachedCS := mgr.active[false][currencyID][fromAddr][chAddr].state
										storedCS, err := mgr.activeInStore.Read(mgr.routineCtx, currencyID, fromAddr, chAddr)
										if err != nil {
											log.Warnf("Sync routine - Fail to read state for inbound %v-%v-%v", currencyID, fromAddr, chAddr)
											// Just in case, push the state.
										} else {
											if cachedCS.StateNonce == storedCS.StateNonce {
												// No need to update.
												return
											}
										}
										mgr.activeInStore.Upsert(paychstate.State{
											CurrencyID:         cachedCS.CurrencyID,
											FromAddr:           cachedCS.FromAddr,
											ToAddr:             cachedCS.ToAddr,
											ChAddr:             cachedCS.ChAddr,
											Balance:            big.NewInt(0).Set(cachedCS.Balance),
											Redeemed:           big.NewInt(0).Set(cachedCS.Redeemed),
											Nonce:              cachedCS.Nonce,
											Voucher:            cachedCS.Voucher,
											NetworkLossVoucher: cachedCS.NetworkLossVoucher,
											StateNonce:         cachedCS.StateNonce,
										})
									}(chAddr)
								}
								wg3.Wait()
							}(fromAddr)
						}
						wg2.Wait()
					}(currencyID)
				}
				wg1.Wait()
				log.Debugf("Sync routine - Finished syncing active inbound channels")
			}()
			wg0.Add(1)
			go func() {
				defer wg0.Done()
				release0, err := mgr.activeLock.RLock(mgr.routineCtx, true)
				if err != nil {
					log.Warnf("Sync routine - Fail to obtain read lock for %v: %v", true, err.Error())
					return
				}
				defer release0()
				wg1 := sync.WaitGroup{}
				for currencyID := range mgr.active[true] {
					wg1.Add(1)
					go func(currencyID byte) {
						defer wg1.Done()
						release1, err := mgr.activeLock.RLock(mgr.routineCtx, true, currencyID)
						if err != nil {
							log.Warnf("Sync routine - Fail to obtain read lock for %v-%v: %v", true, currencyID, err.Error())
							return
						}
						defer release1()
						wg2 := sync.WaitGroup{}
						for toAddr := range mgr.active[true][currencyID] {
							wg2.Add(1)
							go func(toAddr string) {
								defer wg2.Done()
								release2, err := mgr.activeLock.RLock(mgr.routineCtx, true, currencyID, toAddr)
								if err != nil {
									log.Warnf("Sync routine - Fail to obtain read lock for %v-%v-%v: %v", true, currencyID, toAddr, err.Error())
									return
								}
								defer release2()
								wg3 := sync.WaitGroup{}
								for chAddr := range mgr.active[true][currencyID][toAddr] {
									wg3.Add(1)
									go func(chAddr string) {
										defer wg3.Done()
										release3, err := mgr.activeLock.RLock(mgr.routineCtx, true, currencyID, toAddr, chAddr)
										if err != nil {
											log.Warnf("Sync routine - Fail to obtain read lock for %v-%v-%v-%v: %v", true, currencyID, toAddr, chAddr, err.Error())
											return
										}
										defer release3()
										// Get state now.
										cachedCS := mgr.active[true][currencyID][toAddr][chAddr].state
										storedCS, err := mgr.activeOutStore.Read(mgr.routineCtx, currencyID, toAddr, chAddr)
										if err != nil {
											log.Warnf("Sync routine - Fail to read state for outbound %v-%v-%v", currencyID, toAddr, chAddr)
											// Just in case, push the state.
										} else {
											if cachedCS.StateNonce == storedCS.StateNonce {
												// No need to update.
												return
											}
										}
										mgr.activeOutStore.Upsert(paychstate.State{
											CurrencyID:         cachedCS.CurrencyID,
											FromAddr:           cachedCS.FromAddr,
											ToAddr:             cachedCS.ToAddr,
											ChAddr:             cachedCS.ChAddr,
											Balance:            big.NewInt(0).Set(cachedCS.Balance),
											Redeemed:           big.NewInt(0).Set(cachedCS.Redeemed),
											Nonce:              cachedCS.Nonce,
											Voucher:            cachedCS.Voucher,
											NetworkLossVoucher: cachedCS.NetworkLossVoucher,
											StateNonce:         cachedCS.StateNonce,
										})
									}(chAddr)
								}
								wg3.Wait()
							}(toAddr)
						}
						wg2.Wait()
					}(currencyID)
				}
				wg1.Wait()
				log.Debugf("Sync routine - Finished syncing active outbound channels")
			}()
			wg0.Wait()
		case <-mgr.routineCtx.Done():
			// Shutdown.
			log.Infof("Shutdown sync routine")
			return
		}
	}
}

// reservationCleanRoutine is the routine for reservation cleaning.
func (mgr *PaymentManagerImpl) reservationCleanRoutine() {
	for {
		after := time.After(mgr.resCleanFreq)
		select {
		case <-after:
			// Start clean
			log.Infof("Res clean routine - Start clean")
			func() {
				wg0 := sync.WaitGroup{}
				wg0.Add(1)
				go func() {
					defer wg0.Done()
					release0, err := mgr.activeLock.RLock(mgr.routineCtx, true)
					if err != nil {
						log.Warnf("Res clean routine - Fail to obtain read lock for %v: %v", true, err.Error())
						return
					}
					defer release0()
					wg1 := sync.WaitGroup{}
					for currencyID := range mgr.active[true] {
						wg1.Add(1)
						go func(currencyID byte) {
							defer wg1.Done()
							release1, err := mgr.activeLock.RLock(mgr.routineCtx, true, currencyID)
							if err != nil {
								log.Warnf("Res clean routine - Fail to obtain read lock for %v-%v: %v", true, currencyID, err.Error())
								return
							}
							defer release1()
							wg2 := sync.WaitGroup{}
							for toAddr := range mgr.active[true][currencyID] {
								wg2.Add(1)
								go func(toAddr string) {
									defer wg2.Done()
									release2, err := mgr.activeLock.RLock(mgr.routineCtx, true, currencyID, toAddr)
									if err != nil {
										log.Warnf("Res clean routine - Fail to obtain read lock for %v-%v-%v: %v", true, currencyID, toAddr, err.Error())
										return
									}
									defer release2()
									wg3 := sync.WaitGroup{}
									for chAddr := range mgr.active[true][currencyID][toAddr] {
										wg3.Add(1)
										go func(chAddr string) {
											defer wg3.Done()
											release3, err := mgr.activeLock.RLock(mgr.routineCtx, true, currencyID, toAddr, chAddr)
											if err != nil {
												log.Warnf("Res clean routine - Fail to obtain read lock for %v-%v-%v-%v: %v", true, currencyID, toAddr, chAddr, err.Error())
												return
											}
											if getExpiredAmount(mgr.active[true][currencyID][toAddr][chAddr]).Cmp(big.NewInt(0)) > 0 {
												// Need to clean.
												release3()
												ctx, cancel := context.WithTimeout(mgr.routineCtx, mgr.resCleanTimeout)
												defer cancel()
												release3, err = mgr.activeLock.Lock(ctx, true, currencyID, toAddr, chAddr)
												if err != nil {
													log.Warnf("Res clean routine - Fail to obtain write lock for %v-%v-%v-%v: %v", true, currencyID, toAddr, chAddr, err.Error())
													return
												}
												defer release3()
												releaseExpiredAmount(mgr.active[true][currencyID][toAddr][chAddr])
											} else {
												release3()
											}
										}(chAddr)
									}
									wg3.Wait()
								}(toAddr)
							}
							wg2.Wait()
						}(currencyID)
					}
					wg1.Wait()
					log.Infof("Res clean routine - Finished clean reservations")
				}()
				wg0.Wait()
			}()
		case <-mgr.routineCtx.Done():
			// Shutdown.
			log.Infof("Shutdown res clean routine")
			return
		}
	}
}

// peerCleanRoutine is the routine for peer cleaning
func (mgr *PaymentManagerImpl) peerCleanRoutine() {
	for {
		after := time.After(mgr.peerCleanFreq)
		select {
		case <-after:
			// Start clean
			log.Infof("Peer clean routine - Start clean")
			func() {
				wg0 := sync.WaitGroup{}
				wg0.Add(1)
				go func() {
					defer wg0.Done()
					release0, err := mgr.activeLock.RLock(mgr.routineCtx, false)
					if err != nil {
						log.Warnf("Peer clean routine - Fail to obtain read lock for %v: %v", false, err.Error())
						return
					}
					defer release0()
					wg1 := sync.WaitGroup{}
					for currencyID := range mgr.active[false] {
						wg1.Add(1)
						go func(currencyID byte) {
							defer wg1.Done()
							release1, err := mgr.activeLock.RLock(mgr.routineCtx, false, currencyID)
							if err != nil {
								log.Warnf("Peer clean routine - Fail to obtain read lock for %v-%v: %v", false, currencyID, err.Error())
								return
							}
							wg2 := sync.WaitGroup{}
							toRemove := make([]string, 0)
							toRemoveLock := sync.RWMutex{}
							for fromAddr := range mgr.active[false][currencyID] {
								wg2.Add(1)
								go func(fromAddr string) {
									defer wg2.Done()
									release2, err := mgr.activeLock.RLock(mgr.routineCtx, false, currencyID, fromAddr)
									if err != nil {
										log.Warnf("Peer clean routine - Fail to obtain read lock for %v-%v-%v: %v", false, currencyID, fromAddr, err.Error())
										return
									}
									defer release2()
									if len(mgr.active[false][currencyID][fromAddr]) == 0 {
										// Need to remove.
										toRemoveLock.Lock()
										toRemove = append(toRemove, fromAddr)
										toRemoveLock.Unlock()
									}
								}(fromAddr)
							}
							wg2.Wait()
							if len(toRemove) > 0 {
								release1()
								ctx, cancel := context.WithTimeout(mgr.routineCtx, mgr.peerCleanTimeout)
								defer cancel()
								release1, err = mgr.activeLock.Lock(ctx, false, currencyID)
								if err != nil {
									log.Warnf("Peer clean routine - Fail to obtain write lock for inbound %v: %v", currencyID, err.Error())
									return
								}
								defer release1()
								for _, fromAddr := range toRemove {
									_, ok := mgr.active[false][currencyID][fromAddr]
									if ok && len(mgr.active[false][currencyID][fromAddr]) == 0 {
										delete(mgr.active[false][currencyID], fromAddr)
									}
								}
							} else {
								release1()
							}
						}(currencyID)
					}
					wg1.Wait()
					log.Infof("Peer clean routine - Finished clean empty inbound peers")
				}()
				wg0.Add(1)
				go func() {
					defer wg0.Done()
					release0, err := mgr.activeLock.RLock(mgr.routineCtx, true)
					if err != nil {
						log.Warnf("Peer clean routine - Fail to obtain read lock for %v: %v", true, err.Error())
						return
					}
					defer release0()
					wg1 := sync.WaitGroup{}
					for currencyID := range mgr.active[true] {
						wg1.Add(1)
						go func(currencyID byte) {
							defer wg1.Done()
							release1, err := mgr.activeLock.RLock(mgr.routineCtx, true, currencyID)
							if err != nil {
								log.Warnf("Peer clean routine - Fail to obtain read lock for %v-%v: %v", true, currencyID, err.Error())
								return
							}
							wg2 := sync.WaitGroup{}
							toRemove := make([]string, 0)
							toRemoveLock := sync.RWMutex{}
							for fromAddr := range mgr.active[true][currencyID] {
								wg2.Add(1)
								go func(fromAddr string) {
									defer wg2.Done()
									release2, err := mgr.activeLock.RLock(mgr.routineCtx, true, currencyID, fromAddr)
									if err != nil {
										log.Warnf("Peer clean routine - Fail to obtain read lock for %v-%v-%v: %v", true, currencyID, fromAddr, err.Error())
										return
									}
									defer release2()
									if len(mgr.active[true][currencyID][fromAddr]) == 0 {
										// Need to remove.
										toRemoveLock.Lock()
										toRemove = append(toRemove, fromAddr)
										toRemoveLock.Unlock()
									}
								}(fromAddr)
							}
							wg2.Wait()
							if len(toRemove) > 0 {
								release1()
								ctx, cancel := context.WithTimeout(mgr.routineCtx, mgr.peerCleanTimeout)
								defer cancel()
								release1, err = mgr.activeLock.Lock(ctx, true, currencyID)
								if err != nil {
									log.Warnf("Peer clean routine - Fail to obtain write lock for outbound %v: %v", currencyID, err.Error())
									return
								}
								defer release1()
								for _, fromAddr := range toRemove {
									_, ok := mgr.active[true][currencyID][fromAddr]
									if ok && len(mgr.active[true][currencyID][fromAddr]) == 0 {
										delete(mgr.active[true][currencyID], fromAddr)
									}
								}
							} else {
								release1()
							}
						}(currencyID)
					}
					wg1.Wait()
					log.Infof("Peer clean routine - Finished clean empty outbound peers")
				}()
				wg0.Wait()
			}()
		case <-mgr.routineCtx.Done():
			// Shutdown.
			log.Infof("Shutdown peer clean routine")
			return
		}
	}
}
