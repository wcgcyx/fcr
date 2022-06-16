package offerproto

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

import "time"

// publishRoutine is the routine for publish provider record.
func (p *OfferProtocol) publishRoutine() {
	after := time.NewTicker(p.publishFreq)
	for {
		select {
		case <-after.C:
			log.Infof("Publish routine - Start publish")
			pieceChan, errChan := p.cservMgr.ListPieceIDs(p.routineCtx)
			for piece := range pieceChan {
				// TODO: Maybe also publish to indexing service.
				err := p.dht.Provide(p.routineCtx, piece, true)
				if err != nil {
					log.Warnf("Publish routine - Fail to provide %v over DHT: %v", piece, err.Error())
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
