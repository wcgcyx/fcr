package routeproto

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
	"sync"
	"time"
)

// publishRoutine is the routine for publish operation.
func (proto *RouteProtocol) publishRoutine() {
	after := time.NewTicker(proto.publishFreq)
	for {
		select {
		case <-after.C:
			log.Infof("Publish routine - Start publish")
			// Start publish
			curOut, errOut0 := proto.sbs.ListCurrencyIDs(proto.routineCtx)
			wg0 := sync.WaitGroup{}
			for currencyID := range curOut {
				wg0.Add(1)
				go func(currencyID byte) {
					defer wg0.Done()
					subOut, errOut1 := proto.sbs.ListSubscribers(proto.routineCtx, currencyID)
					wg1 := sync.WaitGroup{}
					for sub := range subOut {
						wg1.Add(1)
						go func(sub string) {
							defer wg1.Done()
							err2 := proto.Publish(proto.routineCtx, currencyID, sub)
							if err2 != nil {
								log.Errorf("error publish routes to %v-%v: %v", currencyID, sub, err2.Error())
							}
						}(sub)
					}
					wg1.Wait()
					err1 := <-errOut1
					if err1 != nil {
						log.Errorf("error listing subscribers for %v: %v", currencyID, err1.Error())
					}
				}(currencyID)
			}
			wg0.Wait()
			err0 := <-errOut0
			if err0 != nil {
				log.Errorf("error listing currency ids: %v", err0.Error())
			}
			log.Infof("Publish routine - Finished publish")
		case <-proto.routineCtx.Done():
			// Shutdown.
			log.Infof("Shutdown route publish routine")
			return
		}
	}
}
