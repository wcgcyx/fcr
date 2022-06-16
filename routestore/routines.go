package routestore

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
	"sync"
	"time"
)

// cleanRoutine is the routine for clean expired routes.
func (s *RouteStoreImpl) cleanRoutine() {
	for {
		after := time.After(s.cleanFreq)
		select {
		case <-after:
			// Start clean.
			log.Infof("Clean routine - Start cleaning routes")
			func() {
				wg0 := sync.WaitGroup{}
				for currencyID := range s.maxHops {
					wg0.Add(1)
					go func(currencyID byte) {
						defer wg0.Done()
						log.Debugf("Clean routine - Obtain read lock for %v", currencyID)
						release1, err := s.ds.RLock(s.routineCtx, currencyID)
						if err != nil {
							log.Debugf("Clean routine - Fail to obtain read lock for %v: %v", currencyID, err.Error())
							return
						}
						log.Debugf("Clean routine - Read lock obtained for %v", currencyID)
						temp1 := release1
						release1 = func() {
							temp1()
							log.Debugf("Clean routine - Read lock released for %v", currencyID)
						}
						defer release1()

						txn1, err := s.ds.NewTransaction(s.routineCtx, true)
						if err != nil {
							log.Warnf("Clean routine - Fail to start new transaction: %v", err.Error())
							return
						}
						defer txn1.Discard(context.Background())

						children, err := txn1.GetChildren(s.routineCtx, currencyID)
						if err != nil {
							log.Warnf("Clean routine - Fail to get children for %v: %v", currencyID, err.Error())
							return
						}
						wg1 := sync.WaitGroup{}
						for child := range children {
							wg1.Add(1)
							go func(srcAddr string) {
								defer wg1.Done()
								subCtx, cancel := context.WithTimeout(s.routineCtx, s.cleanTimeout)
								defer cancel()
								log.Debugf("Clean routine - Obtain write lock for %v-%v", currencyID, srcAddr)
								release2, err := s.ds.Lock(subCtx, currencyID, srcAddr)
								if err != nil {
									log.Debugf("Clean routine - Fail to obtain wrote lock for %v-%v: %v", currencyID, srcAddr, err.Error())
									return
								}
								temp2 := release2
								release2 = func() {
									temp2()
									log.Debugf("Clean routine - Read lock released for %v-%v", currencyID, srcAddr)
								}
								defer release2()

								txn2, err := s.ds.NewTransaction(s.routineCtx, false)
								if err != nil {
									log.Warnf("Clean routine - Fail to start new transaction: %v", err.Error())
									return
								}
								defer txn2.Discard(context.Background())

								val, err := txn2.Get(s.routineCtx, currencyID, srcAddr)
								if err != nil {
									log.Warnf("Clean routine - Fail to read ds value for %v-%v: %v", currencyID, srcAddr, err.Error())
									return
								}
								routesMap, err := decDSVal(val)
								if err != nil {
									log.Errorf("Clean routine - Fail to decode ds value %v, this should never happen: %v", val, err.Error())
									return
								}
								routesMap = cleanup(routesMap)
								val, err = encDSVal(routesMap)
								if err != nil {
									log.Errorf("Clean routine - Fail to encode ds value this should never happen: %v", err.Error())
									return
								}
								err = txn2.Put(s.routineCtx, val, currencyID, srcAddr)
								if err != nil {
									log.Warnf("Clean routine - Fail to put ds value %v for %v-%v: %v", val, currencyID, srcAddr, err.Error())
									return
								}
								err = txn2.Commit(s.routineCtx)
								if err != nil {
									log.Warnf("Clean routine - Fail to commit transaction: %v", err.Error())
								}
							}(child)
						}
						wg1.Wait()
					}(currencyID)
				}
				wg0.Wait()
				log.Infof("Clean routine - Finished cleaning routes")
			}()
		case <-s.routineCtx.Done():
			// Shutdown.
			log.Infof("Shutdown cleaning routes routine")
			return
		}
	}
}
