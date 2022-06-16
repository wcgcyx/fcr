package paychmon

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
	"time"
)

// trackRoutine is used to track a payment channel.
//
// @input - whether it is outbound channel, currency id, peer address, channel address, settlement, renew signal chan, retire signal chan.
func (m *PaychMonitorImpl) trackRoutine(outbound bool, currencyID byte, peerAddr string, chAddr string, settlement time.Time, renew chan time.Time, retire chan bool) {
	check := time.NewTicker(1 * time.Second)
	first := true
	defer check.Stop()
	expire := time.NewTimer(settlement.Sub(time.Now()))
	for {
		inactive := false
		select {
		case <-check.C:
			if first {
				first = false
				check.Reset(m.checkFreq)
			}
			// Check if settling.
			log.Infof("Track routine - Start checking channel state for %v-%v", currencyID, chAddr)
			settling, _, _, _, _, _, _, err := m.transactor.Check(m.routineCtx, currencyID, chAddr)
			if err != nil {
				log.Warnf("Fail to check payment channel %v-%v: %v", currencyID, chAddr, err.Error())
				continue
			}
			if settling != 0 {
				expire.Stop()
				inactive = true
				log.Infof("Track routine - Channel %v-%v is now inactive", currencyID, chAddr)
			}
			log.Infof("Track routine - Finished checking")
		case newSettlement := <-renew:
			expire.Stop()
			expire = time.NewTimer(newSettlement.Sub(time.Now()))
		case <-expire.C:
			expire.Stop()
			inactive = true
		case <-retire:
			expire.Stop()
			inactive = true
		case <-m.routineCtx.Done():
			expire.Stop()
			log.Infof("Shutdown track routine")
			return
		}
		if inactive {
			break
		}
	}
	// Remove from ds.
	log.Infof("Track routine - Remove inactive payment channel %v-%v", currencyID, chAddr)
	m.addToQueue(func() {
		log.Debugf("Track routine - Obtain write lock for %v-%v", outbound, currencyID)
		release, err := m.store.Lock(m.routineCtx, outbound, currencyID)
		if err != nil {
			log.Warnf("Track routine - Fail to obtain write lock for %v-%v to remove %v: %v", outbound, currencyID, chAddr, err.Error())
			return
		}
		log.Debugf("Track routine - Write lock obtained for %v-%v", outbound, currencyID)
		temp := release
		release = func() {
			temp()
			log.Debugf("Track routine - Write lock released for %v-%v", outbound, currencyID)
		}
		defer release()

		txn, err := m.store.NewTransaction(m.routineCtx, false)
		if err != nil {
			log.Warnf("Track routine - Fail to start new transaction for %v-%v to remove %v: %v: %v", outbound, currencyID, chAddr, err.Error())
			return
		}
		defer txn.Discard(context.Background())

		err = txn.Delete(m.routineCtx, outbound, currencyID, chAddr)
		if err != nil {
			log.Warnf("Track routine - Fail to remove %v-%v-%v: %v", outbound, currencyID, chAddr, err.Error())
			return
		}
		err = txn.Commit(m.routineCtx)
		if err != nil {
			log.Warnf("Track routine - Fail to commit transaction to remove %v-%v-%v: %v", outbound, currencyID, chAddr, err.Error())
			return
		}
	})
	// Retire channel.
	var err error
	if outbound {
		err = m.pam.RetireOutboundChannel(m.routineCtx, currencyID, peerAddr, chAddr)
	} else {
		err = m.pam.RetireInboundChannel(m.routineCtx, currencyID, peerAddr, chAddr)
	}
	if err != nil {
		log.Errorf("Track routine - Fail to retire channel %v-%v-%v-%v from payment manager: %v", outbound, currencyID, peerAddr, chAddr, err.Error())
	}

	// Remove from cache.
	log.Debugf("Track routine - Obtain write lock for %v-%v", outbound, currencyID)
	release, err := m.cacheLock.Lock(m.routineCtx, outbound, currencyID)
	if err != nil {
		log.Warnf("Track routine - Fail to obtain write lock for %v-%v to remove %v from cache: %v", outbound, currencyID, chAddr, err.Error())
		return
	}
	log.Debugf("Track routine - Write lock obtained for %v-%v", outbound, currencyID)
	temp := release
	release = func() {
		temp()
		log.Debugf("Track routine - Write lock released for %v-%v", outbound, currencyID)
	}
	defer release()
	delete(m.cache[outbound][currencyID], chAddr)
	m.cacheLock.Remove(outbound, currencyID, chAddr)
}
