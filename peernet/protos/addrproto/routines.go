package addrproto

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
	"time"
)

// publishRoutine is the routine for publish provider record.
func (p *AddrProtocol) publishRoutine() {
	after := time.NewTicker(p.publishFreq)
	for {
		select {
		case <-after.C:
			log.Infof("Publish routine - Start publish")
			curChan, errChan := p.signer.ListCurrencyIDs(p.routineCtx)
			for currencyID := range curChan {
				_, toAddr, err := p.signer.GetAddr(p.routineCtx, currencyID)
				if err != nil {
					log.Warnf("Publish routine - Fail to addr for %v from signer: %v", currencyID, err.Error())
				} else {
					// Publish addr record to DHT. TODO: Maybe also publish to indexing service.
					rec, err := getAddrRecord(toAddr, currencyID)
					if err != nil {
						log.Errorf("Publish routine - Fail to generate address record for %v-%v: %v", currencyID, toAddr, err.Error())
					} else {
						err := p.dht.Provide(p.routineCtx, rec, true)
						if err != nil {
							log.Warnf("Publish routine  - Fail to provide provide %v over DHT: %v", rec, err.Error())
						}
					}
				}
			}
			err := <-errChan
			if err != nil {
				log.Warnf("Publish routine - Fail to list currency ids from signer: %v", err.Error())
			}
			log.Infof("Publish routine - Finished publish")
		case <-p.routineCtx.Done():
			log.Infof("Shutdown publish routine")
			return
		}
	}
}
