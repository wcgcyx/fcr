package cli

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
	"encoding/base64"
	"fmt"
	"math/big"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	"github.com/wcgcyx/fcr/api"
	"github.com/wcgcyx/fcr/fcroffer"
)

// Retrieve command.
var RetrieveCMD = &cli.Command{
	Name:        "retrieve",
	Usage:       "retrieve a piece",
	Description: "Retrieve a piece based on given piece offer and paych reservation",
	ArgsUsage:   "[piece offer, reservation (0-0 if free), outpath, pay offer (optional)]",
	Action: func(c *cli.Context) error {
		offerData, err := base64.StdEncoding.DecodeString(c.Args().Get(0))
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse piece offer data"))
		}
		reservation := strings.Split(c.Args().Get(1), "-")
		if len(reservation) != 2 {
			return usageError(c, fmt.Errorf("fail to parse reservation, should be resCh-resId"))
		}
		resCh := reservation[0]
		resID, err := strconv.ParseUint(reservation[1], 10, 64)
		if err != nil {
			return err
		}
		outPath := c.Args().Get(2)
		if outPath == "" {
			return usageError(c, fmt.Errorf("empty outpath received"))
		}
		outPathAbs, err := filepath.Abs(outPath)
		if err != nil {
			return fmt.Errorf("error getting path: %v", err.Error())
		}
		pieceOffer := fcroffer.PieceOffer{}
		err = pieceOffer.Decode(offerData)
		if err != nil {
			return err
		}
		var payOffer *fcroffer.PayOffer
		if c.Args().Len() > 3 {
			offerData, err := base64.StdEncoding.DecodeString(c.Args().Get(3))
			if err != nil {
				return err
			}
			offer := fcroffer.PayOffer{}
			err = offer.Decode(offerData)
			if err != nil {
				return err
			}
			payOffer = &offer
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		fmt.Println("Start retrieval...")
		resChan := client.RetMgrRetrieve(c.Context, pieceOffer, payOffer, resCh, resID, outPathAbs)
		for res := range resChan {
			if res.Err != "" {
				return fmt.Errorf(res.Err)
			}
			fmt.Printf("\r%v\t\t", res.Progress)
		}
		fmt.Println("\nSucceed")
		return nil
	},
}

var RetrieveCacheCMD = &cli.Command{
	Name:        "cache-retrieve",
	Usage:       "retrieve a piece from cache",
	Description: "Retrieve a piece based on given cid from cache",
	ArgsUsage:   "[cid, outpath]",
	Action: func(c *cli.Context) error {
		id, err := cid.Parse(c.Args().Get(0))
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse cid: %v", err.Error()))
		}
		outPath := c.Args().Get(1)
		if outPath == "" {
			return usageError(c, fmt.Errorf("empty outpath received"))
		}
		outPathAbs, err := filepath.Abs(outPath)
		if err != nil {
			return fmt.Errorf("error getting path: %v", err.Error())
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		found, err := client.RetMgrRetrieveFromCache(c.Context, id, outPathAbs)
		if err != nil {
			return err
		}
		if found {
			fmt.Println("Succeed")
		} else {
			fmt.Println("Piece not found in cache")
		}
		return nil
	},
}

var FastRetrieveCMD = &cli.Command{
	Name:        "fast-retrieve",
	Usage:       "fast retrieve a piece",
	Description: "Fast retrieve a piece from network using pre-set strategy",
	ArgsUsage:   "[cid, outpath, currency id, max amount]",
	Action: func(c *cli.Context) error {
		id, err := cid.Parse(c.Args().Get(0))
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse cid: %v", err.Error()))
		}
		outPath := c.Args().Get(1)
		if outPath == "" {
			return usageError(c, fmt.Errorf("empty outpath received"))
		}
		outPathAbs, err := filepath.Abs(outPath)
		if err != nil {
			return fmt.Errorf("error getting path: %v", err.Error())
		}
		currencyID, err := strconv.ParseUint(c.Args().Get(2), 10, 8)
		if err != nil {
			return usageError(c, fmt.Errorf("fail to parse currency id: %v", err.Error()))
		}
		max, ok := big.NewInt(0).SetString(c.Args().Get(3), 10)
		if !ok {
			return usageError(c, fmt.Errorf("fail to parse max"))
		}
		if max.Cmp(big.NewInt(0)) <= 0 {
			return usageError(c, fmt.Errorf("max amount must be positive, got %v", max))
		}
		client, closer, err := api.NewClient(c.Context, c.Int("port"), c.Path("auth"))
		if err != nil {
			return err
		}
		defer closer()
		// First try to retrieve from cache
		fmt.Println("Try to retrieve from cache...")
		found, err := client.RetMgrRetrieveFromCache(c.Context, id, outPathAbs)
		if err != nil {
			if !strings.Contains(err.Error(), "failed to fetch all nodes") {
				return err
			}
		}
		if found {
			fmt.Println("Retrieved from cache")
			return nil
		}
		// Second try to query offers from peers we are having payment channel with.
		fmt.Println("Try to retrieve using direct payment...")
		peerChan := client.ActiveOutListPeers(c.Context, byte(currencyID))
		peers := make(map[string]bool)
		for peer := range peerChan {
			peers[peer] = true
			offer, err := client.COfferProtoQueryOffer(c.Context, byte(currencyID), peer, id)
			if err != nil {
				fmt.Printf("Fail to query piece offer from %v-%v: %v\n", currencyID, peer, err.Error())
				continue
			}
			// TODO: Filter offer with too short expiration & inactivity.
			offerData, err := offer.Encode()
			if err != nil {
				fmt.Printf("Fail to encode offer: %v\n", err.Error())
				continue
			}
			required := big.NewInt(0).Mul(offer.PPB, big.NewInt(int64(offer.Size)))
			if required.Cmp(max) > 0 {
				continue
			}
			if required.Cmp(big.NewInt(0)) > 0 {
				// Reserve and retrieve.
				res, err := client.PayMgrReserveForSelf(c.Context, offer.CurrencyID, offer.RecipientAddr, required, offer.Expiration, offer.Inactivity)
				if err != nil {
					fmt.Printf("Fail to reserve for %v-%v: %v\n", currencyID, peer, err.Error())
					continue
				}
				fmt.Printf("Start direct retrieval using piece offer %v and reservation %v-%v\n", base64.StdEncoding.EncodeToString(offerData), res.ResCh, res.ResID)
				resChan := client.RetMgrRetrieve(c.Context, offer, nil, res.ResCh, res.ResID, outPathAbs)
				for res := range resChan {
					if res.Err != "" {
						err = fmt.Errorf(res.Err)
						break
					}
					fmt.Printf("\r%v\t\t", res.Progress)
				}
				if err == nil {
					fmt.Printf("\nRetrieved from %v-%v with direct payment %v\n", currencyID, peer, required)
					return nil
				}
			} else {
				// Free retrieval, retrieve directly.
				fmt.Printf("Start direct free retrieval using piece offer of %v\n", base64.StdEncoding.EncodeToString(offerData))
				resChan := client.RetMgrRetrieve(c.Context, offer, nil, "0", 0, outPathAbs)
				for res := range resChan {
					if res.Err != "" {
						err = fmt.Errorf(res.Err)
						break
					}
					fmt.Printf("\r%v\t\t", res.Progress)
				}
				if err == nil {
					fmt.Printf("\nRetrieved from %v-%v with free payment\n", currencyID, offer.RecipientAddr)
					return nil
				}
			}
		}
		// Third try to query offer from the network.
		fmt.Println("Try to retrieve using proxy payment...")
		offerChan := client.COfferProtoFindOffersAsync(c.Context, byte(currencyID), id, 0)
		for offer := range offerChan {
			// Skip offer from direct payment peers.
			if peers[offer.RecipientAddr] {
				continue
			}
			offerData, err := offer.Encode()
			if err != nil {
				fmt.Printf("Fail to encode offer: %v\n", err.Error())
				continue
			}
			if offer.PPB.Cmp(big.NewInt(0)) > 0 {
				// Retreival not free, query pay offer.
				required := big.NewInt(0).Mul(offer.PPB, big.NewInt(int64(offer.Size)))
				if required.Cmp(max) > 0 {
					continue
				}
				// Try to find pay offer
				routeChan := client.RouteStoreListRoutesTo(c.Context, offer.CurrencyID, offer.RecipientAddr)
				for route := range routeChan {
					payOffer, err := client.POfferProtoQueryOffer(c.Context, offer.CurrencyID, route, required)
					if err != nil {
						fmt.Printf("Fail to query pay offer: %v\n", err.Error())
						continue
					}
					payOfferData, err := payOffer.Encode()
					if err != nil {
						fmt.Printf("Fail to encode pay offer: %v\n", err.Error())
						continue
					}
					if payOffer.PPP.Cmp(big.NewInt(0)) > 0 {
						// Calculate total cost of this pay offer
						temp := big.NewInt(0).Sub(required, big.NewInt(1))
						temp = big.NewInt(0).Div(temp, payOffer.Period)
						temp = big.NewInt(0).Add(temp, big.NewInt(1))
						temp = big.NewInt(0).Mul(temp, payOffer.PPP)
						total := big.NewInt(0).Add(temp, required)
						if total.Cmp(max) > 0 {
							continue
						}
					}
					// Either free retrieval or less than max
					// Reserve
					res, err := client.PayMgrReserveForSelfWithOffer(c.Context, payOffer)
					if err != nil {
						fmt.Printf("Fail to reserve: %v", err.Error())
						continue
					}
					fmt.Printf("Start retrieval using piece offer %v, reservation %v-%v and payment offer %v\n", base64.StdEncoding.EncodeToString(offerData), res.ResCh, res.ResID, base64.StdEncoding.EncodeToString(payOfferData))
					resChan := client.RetMgrRetrieve(c.Context, offer, &payOffer, res.ResCh, res.ResID, outPathAbs)
					for res := range resChan {
						if res.Err != "" {
							err = fmt.Errorf(res.Err)
							break
						}
						fmt.Printf("\r%v\t\t", res.Progress)
					}
					if err == nil {
						fmt.Printf("\nRetrieved from %v-%v with proxy payment %v from %v\n", currencyID, offer.RecipientAddr, required, payOffer.SrcAddr)
						return nil
					}
				}
			} else {
				// Free retrieval, retrieve directly.
				fmt.Printf("Start free retrieval using piece offer of %v\n", base64.StdEncoding.EncodeToString(offerData))
				resChan := client.RetMgrRetrieve(c.Context, offer, nil, "0", 0, outPathAbs)
				for res := range resChan {
					if res.Err != "" {
						err = fmt.Errorf(res.Err)
						break
					}
					fmt.Printf("\r%v\t\t", res.Progress)
				}
				if err == nil {
					fmt.Printf("\nRetrieved from %v-%v with free payment\n", currencyID, offer.RecipientAddr)
					return nil
				}
			}
		}
		return fmt.Errorf("fail to fast retrieve %v with max %v-%v, recommend using old offer to retry instead of using another fast-retrieve", id, currencyID, max)
	},
}
