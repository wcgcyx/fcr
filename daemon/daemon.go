package daemon

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
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/wcgcyx/fcr/api"
	"github.com/wcgcyx/fcr/config"
	"github.com/wcgcyx/fcr/node"
)

// Logger
var log = logging.Logger("daemon")

// Daemon starts a daemon from a config file.
//
// @input - context, config file.
//
// @output - error.
func Daemon(ctx context.Context, configFile string) error {
	logging.SetLogLevel("daemon", "INFO")
	// Load config
	log.Infof("Load configuration...")
	conf, err := config.NewConfig(configFile)
	if err != nil {
		return err
	}
	// Start node
	log.Infof("Start daemon...")
	node, err := node.NewNode(ctx, conf)
	if err != nil {
		return err
	}
	// Start API Server
	log.Infof("Start serving API...")
	apiServer, err := api.NewServer(node, int(conf.APIPort), conf.APIDevMode, conf.Path)
	if err != nil {
		defer node.Shutdown()
		return err
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	log.Infof("Daemon started.")
	// Do initial bootstrap.
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go bootstrap(subCtx, node)
	go publish(subCtx, node)
	for {
		// Loop forever, until exit
		<-c
		break
	}
	log.Infof("Graceful shutdown daemon...")
	cancel()
	apiServer.Shutdown()
	node.Shutdown()
	log.Infof("Daemon stopped.")
	return nil
}

// bootstrap performs an initial bootstrap.
//
// @input - context, node.
func bootstrap(ctx context.Context, node *node.Node) {
	// Wait for 1 second.
	after := time.After(1 * time.Second)
	select {
	case <-after:
		log.Infof("Start bootstrapping...")
		connected := 0
		connectedLock := sync.RWMutex{}
		wg := sync.WaitGroup{}
		curChan, errChan1 := node.PeerMgr.ListCurrencyIDs(ctx)
		for currencyID := range curChan {
			peerChan, errChan2 := node.PeerMgr.ListPeers(ctx, currencyID)
			for peer := range peerChan {
				pi, err := node.PeerMgr.PeerAddr(ctx, currencyID, peer)
				if err != nil {
					log.Warnf("Bootstrap - Fail to get peer addr for %v-%v: %v", currencyID, peer, err.Error())
					continue
				}
				wg.Add(1)
				go func() {
					defer wg.Done()
					subCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
					defer cancel()
					err = node.Host.Connect(subCtx, pi)
					if err != nil {
						log.Debugf("Bootstrap - Fail to connect to %v: %v", pi, err.Error())
						return
					}
					log.Debugf("Connected to %v", pi)
					connectedLock.Lock()
					defer connectedLock.Unlock()
					connected++
				}()
			}
			err := <-errChan2
			if err != nil {
				log.Warnf("Fail to list peers for %v: %v", currencyID, err.Error())
			}
		}
		err := <-errChan1
		if err != nil {
			log.Warnf("Fail to list currency ids: %v", err.Error())
		}
		wg.Wait()
		log.Infof("Initial bootstrapping done, %v connected.", connected)
	case <-ctx.Done():
		log.Warnf("Stopping bootstrapping: %v", ctx.Err().Error())
	}
}

func publish(ctx context.Context, node *node.Node) {
	// Wait for 5 seconds.
	after := time.After(5 * time.Second)
	select {
	case <-after:
		err := node.AddrProto.Publish(ctx)
		if err != nil {
			log.Warnf("Fail to publish address: %v", err.Error())
		}
		err = node.CofferProto.Publish(ctx)
		if err != nil {
			log.Warnf("Fail to publish cid servings: %v", err.Error())
		}
		err = node.RouteProto.StartPublish(ctx)
		if err != nil {
			log.Warnf("Fail to publish route: %v", err.Error())
		}
	case <-ctx.Done():
		log.Warnf("Stopping initial publish: %v", ctx.Err().Error())
	}
}
