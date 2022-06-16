package payproto

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
)

// cleanRoutine is the routine for clean operation.
func (p *PayProtocol) cleanRoutine() {
	for {
		after := time.After(p.cleanFreq)
		select {
		case <-after:
			// Start clean
			log.Infof("Clean routine - Start clean")
			func() {
				release0, err := p.receivedLock.RLock(p.routineCtx)
				if err != nil {
					log.Warnf("Clean routine - Fail to obtain read lock for received: %v", err.Error())
					return
				}
				defer release0()
				wg1 := sync.WaitGroup{}
				for currencyID := range p.received {
					wg1.Add(1)
					go func(currencyID byte) {
						defer wg1.Done()
						release1, err := p.receivedLock.RLock(p.routineCtx, currencyID)
						if err != nil {
							log.Warnf("Clean routine - Fail to obtain read lock for %v: %v", currencyID, err.Error())
							return
						}
						wg2 := sync.WaitGroup{}
						toRemove := make([]string, 0)
						toRemoveLock := sync.RWMutex{}
						for fromAddr := range p.received[currencyID] {
							wg2.Add(1)
							go func(fromAddr string) {
								defer wg2.Done()
								release2, err := p.receivedLock.RLock(p.routineCtx, currencyID, fromAddr)
								if err != nil {
									log.Warnf("Clean routine - Fail to obtain read lock for %v-%v: %v", currencyID, fromAddr, err.Error())
									return
								}
								defer release2()
								if p.received[currencyID][fromAddr].Cmp(big.NewInt(0)) == 0 {
									// Need to clean
									toRemoveLock.Lock()
									toRemove = append(toRemove, fromAddr)
									toRemoveLock.Unlock()
								}
							}(fromAddr)
						}
						wg2.Wait()
						if len(toRemove) > 0 {
							release1()
							ctx, cancel := context.WithTimeout(p.routineCtx, p.cleanTimeout)
							defer cancel()
							release1, err = p.receivedLock.Lock(ctx, currencyID)
							if err != nil {
								log.Errorf("Clean routine - Fail to obtain write lock for %v: %v", currencyID, err.Error())
								return
							}
							defer release1()
							for _, fromAddr := range toRemove {
								_, ok := p.received[currencyID][fromAddr]
								if ok && p.received[currencyID][fromAddr].Cmp(big.NewInt(0)) == 0 {
									delete(p.received[currencyID], fromAddr)
								}
							}
						} else {
							release1()
						}
					}(currencyID)
				}
				wg1.Wait()
				log.Infof("Clean routine - Finished clean received map")
			}()
		case <-p.routineCtx.Done():
			// Shutdown.
			log.Infof("Shutdown clean routine")
			return
		}
	}
}
