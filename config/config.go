package config

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
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

const (
	defaultPath    = ".fcr"
	defaultP2PPort = 19955
	defaultAPIPort = 9424
)

// Configuration for FCR node.
type Config struct {
	// Path
	Path string `mapstructure:"FCR_PATH"` // FCR datastore path.

	// Port
	P2PPort uint64 `mapstructure:"FCR_P2P_PORT"` // FCR P2P port.
	APIPort uint64 `mapstructure:"FCR_API_PORT"` // FCR API port.

	// API Server settings
	APIServerLoggingLevel string `mapstructure:"APISEVER_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.
	APIDevMode            bool   `mapstructure:"APISEVER_DEV_MODE"`      // Server DEV API enabled: True, False.

	// Signer settings
	SignerLoggingLevel string `mapstructure:"SIGNER_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.

	// Peer manager settings
	PeerMgrLoggingLevel string `mapstructure:"PEERMGR_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.

	// Transactor settings
	TransactorLoggingLevel       string `mapstructure:"TRANSACTOR_LOGGING_LEVEL"`       // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.
	TransactorFilecoinEnabled    bool   `mapstructure:"TRANSACTOR_FILECOIN_ENABLED"`    // Filecoin enabled: True, False.
	TransactorFilecoinAPI        string `mapstructure:"TRANSACTOR_FILECOIN_API"`        // Filecoin api address (Non empty if Filecoin is enabled).
	TransactorFilecoinAuthToken  string `mapstructure:"TRANSACTOR_FILECOIN_AUTH_TOKEN"` // Filecoin auth token (Can be empty if remote access is used).
	TransactorFilecoinConfidence uint64 `mapstructure:"TRANSACTOR_FILECOIN_CONFIDENCE"` // Filecoin confidence: 0-10.

	// Active out paych store settings
	ActiveOutLoggingLevel string        `mapstructure:"ACTIVEOUT_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.
	ActiveOutDSTimeout    time.Duration `mapstructure:"ACTIVEOUT_DS_TIMEOUT"`    // ActiveOut datastore timeout.
	ActiveOutDSRetry      uint64        `mapstructure:"ACTIVEOUT_DS_RETRY"`      // ActiveOut datastore retry limit.

	// Inactive out paych store settings
	InactiveOutLoggingLevel string        `mapstructure:"INACTIVEOUT_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.
	InactiveOutDSTimeout    time.Duration `mapstructure:"INACTIVEOUT_DS_TIMEOUT"`    // InactiveOut datastore timeout.
	InactiveOutDSRetry      uint64        `mapstructure:"INACTIVEOUT_DS_RETRY"`      // InactiveOut datastore retry limit.

	// Active in paych store settings
	ActiveInLoggingLevel string        `mapstructure:"ACTIVEIN_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.
	ActiveInDSTimeout    time.Duration `mapstructure:"ACTIVEIN_DS_TIMEOUT"`    // ActiveIn datastore timeout.
	ActiveInDSRetry      uint64        `mapstructure:"ACTIVEIN_DS_RETRY"`      // ActiveIn datastore retry limit.

	// Inactive in paych store settings
	InactiveInLoggingLevel string        `mapstructure:"INACTIVEIN_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.
	InactiveInDSTimeout    time.Duration `mapstructure:"INACTIVEIN_DS_TIMEOUT"`    // InactiveIn datastore timeout.
	InactiveInDSRetry      uint64        `mapstructure:"INACTIVEIN_DS_RETRY"`      // InactiveIn datastore retry limit.

	// Paych serving manager settings
	PServMgrLoggingLevel string `mapstructure:"PSERVMGR_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.

	// Route store settings
	RouteStoreLoggingLevel string        `mapstructure:"ROUTESTORE_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.
	RouteStoreCleanFreq    time.Duration `mapstructure:"ROUTESTORE_CLEAN_FREQ"`    // RouteStore clean frequency.
	RouteStoreCleanTimeout time.Duration `mapstructure:"ROUTESTORE_CLEAN_TIMEOUT"` // RouteStore clean timeout.
	RouteStoreMaxHopFIL    uint64        `mapstructure:"ROUTESTORE_MAX_HOP_FIL"`   // RouteStore max hop for FIL: 3-10.

	// Subscriber store settings
	SubStoreLoggingLevel string `mapstructure:"SUBSTORE_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.

	// Payment manager settings
	PayMgrLoggingLevel     string        `mapstructure:"PAYMGR_LOGGING_LEVEL"`      // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.
	PayMgrCacheSyncFreq    time.Duration `mapstructure:"PAYMGR_CACHE_SYNC_FREQ"`    // Payment manager cache sync frequency.
	PayMgrResCleanFreq     time.Duration `mapstructure:"PAYMGR_RES_CLEAN_FREQ"`     // Payment manager reservation clean frequency.
	PayMgrResCleanTimeout  time.Duration `mapstructure:"PAYMGR_RES_CLEAN_TIMEOUT"`  // Payment manager reservation clean timeout.
	PayMgrPeerCleanFreq    time.Duration `mapstructure:"PAYMGR_PEER_CLEAN_FREQ"`    // Payment manager peer clean frequency.
	PayMgrPeerCleanTimeout time.Duration `mapstructure:"PAYMGR_PEER_CLEAN_TIMEOUT"` // Payment manager peer clean timeout.

	// Settlement manager settings
	SettleMgrLoggingLevel string `mapstructure:"SETTLEMGR_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.

	// Renew manager settings
	RenewMgrLoggingLevel string `mapstructure:"RENEWMGR_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.

	// Reservation manager settings
	ReservMgrLoggingLevel string `mapstructure:"RESERVMGR_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.

	// Offer manager settings
	OfferMgrLoggingLevel string `mapstructure:"OFFERMGR_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.

	// Paych monitor settings
	PaychMonitorLoggingLevel string        `mapstructure:"PAYCHMONITOR_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.
	PaychMonitorCheckFreq    time.Duration `mapstructure:"PAYCHMONITOR_CHECK_FREQ"`    // Paych monitor check frequency.

	// Piece manager settings
	PieceMgrLoggingLevel string `mapstructure:"PIECEMGR_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.

	// Piece serving manager settings
	CServMgrLoggingLevel string `mapstructure:"CSERVMGR_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.

	// Miner proof store settings
	MinerProofStoreLoggingLevel string `mapstructure:"MINERPROOFSTORE_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.

	// Addr protocol settings
	AddrProtoLoggingLevel string        `mapstructure:"ADDRPROTO_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.
	AddrProtoIOTimeout    time.Duration `mapstructure:"ADDRPROTO_IO_TIMEOUT"`    // Addr protocol IO timeout.
	AddrProtoOPTimeout    time.Duration `mapstructure:"ADDRPROTO_OP_TIMEOUT"`    // Addr protocol OP timeout.
	AddrProtoPublishFreq  time.Duration `mapstructure:"ADDRPROTO_PUBLISH_FREQ"`  // Addr protocol publish frequency.

	// Paych protocol settings
	PaychProtoLoggingLevel string        `mapstructure:"PAYCHPROTO_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.
	PaychProtoIOTimeout    time.Duration `mapstructure:"PAYCHPROTO_IO_TIMEOUT"`    // Paych protocol IO timeout.
	PaychProtoOPTimeout    time.Duration `mapstructure:"PAYCHPROTO_OP_TIMEOUT"`    // Paych protocol OP timeout.
	PaychProtoOfferExpiry  time.Duration `mapstructure:"PAYCHPROTO_OFFER_EXPIRY"`  // Paych protocol offer expiry.
	PaychProtoRenewWindow  uint64        `mapstructure:"PAYCHPROTO_RENEW_WINDOW"`  // Paych protocol renew window: 5-95.

	// Route protocol settings
	RouteProtoLoggingLevel string        `mapstructure:"ROUTEPROTO_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.
	RouteProtoIOTimeout    time.Duration `mapstructure:"ROUTEPROTO_IO_TIMEOUT"`    // Route protocol IO timeout.
	RouteProtoOPTimeout    time.Duration `mapstructure:"ROUTEPROTO_OP_TIMEOUT"`    // Route protocol OP timeout.
	RouteProtoPublishFreq  time.Duration `mapstructure:"ROUTEPROTO_PUBLISH_FREQ"`  // Route protocol publish frequency.
	RouteProtoRouteExpiry  time.Duration `mapstructure:"ROUTEPROTO_ROUTE_EXPIRY"`  // Route protocol route expiry.
	RouteProtoPublishWait  time.Duration `mapstructure:"ROUTEPROTO_PUBLISH_WAIT"`  // Route protocol initial wait time for publish.

	// Pay offer protocol settings
	POfferProtoLoggingLevel    string        `mapstructure:"POFFERPROTO_LOGGING_LEVEL"`    // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.
	POfferProtoIOTimeout       time.Duration `mapstructure:"POFFERPROTO_IO_TIMEOUT"`       // Pay offer protocol IO timeout.
	POfferProtoOPTimeout       time.Duration `mapstructure:"POFFERPROTO_OP_TIMEOUT"`       // Pay offer protocol OP timeout.
	POfferProtoOfferExpiry     time.Duration `mapstructure:"POFFERPROTO_OFFER_EXPIRY"`     // Pay offer protocol offer expiry.
	POfferProtoOfferInactivity time.Duration `mapstructure:"POFFERPROTO_OFFER_INACTIVITY"` // Pay offer protocol offer inactivity.

	// Piece offer protocol settings
	COfferProtoLoggingLevel    string        `mapstructure:"COFFERPROTO_LOGGING_LEVEL"`    // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.
	COfferProtoIOTimeout       time.Duration `mapstructure:"COFFERPROTO_IO_TIMEOUT"`       // Piece offer protocol IO timeout.
	COfferProtoOPTimeout       time.Duration `mapstructure:"COFFERPROTO_OP_TIMEOUT"`       // Piece offer protocol OP timeout.
	COfferProtoOfferExpiry     time.Duration `mapstructure:"COFFERPROTO_OFFER_EXPIRY"`     // Piece offer protocol offer expiry.
	COfferProtoOfferInactivity time.Duration `mapstructure:"COFFERPROTO_OFFER_INACTIVITY"` // Piece offer protocol offer inactivity.
	COfferPublishFreq          time.Duration `mapstructure:"COFFERPROTO_PUBLISH_FREQ"`     // Piece offer protocol publish frequency.

	// Pay protocol settings
	PayProtoLoggingLevel string        `mapstructure:"PAYPROTO_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.
	PayProtoIOTimeout    time.Duration `mapstructure:"PAYPROTO_IO_TIMEOUT"`    // Pay protocol IO timeout.
	PayProtoOPTimeout    time.Duration `mapstructure:"PAYPROTO_OP_TIMEOUT"`    // Pay protocol OP timeout.
	PayProtoCleanFreq    time.Duration `mapstructure:"PAYPROTO_CLEAN_FREQ"`    // Pay protocol clean frequency.
	PayProtoCleanTimeout time.Duration `mapstructure:"PAYPROTO_CLEAN_TIMEOUT"` // Pay protocol clean timeout.

	// Retrieval manager settings
	RetMgrLoggingLevel string        `mapstructure:"RETMGR_LOGGING_LEVEL"` // Log Level: FATAL, PANIC, ERROR, WARN, INFO, DEBUG.
	RetMgrIOTimeout    time.Duration `mapstructure:"RETMGR_IO_TIMEOUT"`    // Retrieval manager IO timeout.
	RetMgrOPTimeout    time.Duration `mapstructure:"RETMGR_OP_TIMEOUT"`    // Retrieval manager OP timeout.
}

// NewConfig creates a new configuration.
//
// @output - configuration, error.
func NewConfig(configFile string) (Config, error) {
	// Try to load config file from $HOME/.fcr
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.fcr")
	if configFile != "" {
		viper.SetConfigFile(configFile)
	}
	err := viper.ReadInConfig()
	if err != nil {
		return Config{}, err
	}
	// Parse path
	path := viper.GetString("FCR_PATH")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Config{}, err
		}
		path = filepath.Join(home, ".fcr")
	}

	// Parse p2p port
	p2pPort := viper.GetInt("FCR_P2P_PORT")
	if p2pPort == 0 {
		p2pPort = defaultP2PPort
	}

	// Parse api port
	apiPort := viper.GetInt("FCR_API_PORT")
	if apiPort == 0 {
		apiPort = defaultAPIPort
	}

	// Parse confidence
	confidence := viper.GetInt("TRANSACTOR_FILECOIN_CONFIDENCE")
	if confidence < 0 || confidence > 10 {
		// Have the confidence to be default 4.
		confidence = 4
	}

	// Parse max hops
	maxHopFIL := viper.GetInt("ROUTESTORE_MAX_HOP_FIL")
	if maxHopFIL < 3 || maxHopFIL > 10 {
		maxHopFIL = 0
	}

	// Parse the rest options and return.
	return Config{
		Path:                         path,
		P2PPort:                      uint64(p2pPort),
		APIPort:                      uint64(apiPort),
		APIServerLoggingLevel:        viper.GetString("APISEVER_LOGGING_LEVEL"),
		APIDevMode:                   viper.GetBool("APISEVER_DEV_MODE"),
		SignerLoggingLevel:           viper.GetString("SIGNER_LOGGING_LEVEL"),
		PeerMgrLoggingLevel:          viper.GetString("PEERMGR_LOGGING_LEVEL"),
		TransactorLoggingLevel:       viper.GetString("TRANSACTOR_LOGGING_LEVEL"),
		TransactorFilecoinEnabled:    viper.GetBool("TRANSACTOR_FILECOIN_ENABLED"),
		TransactorFilecoinAPI:        viper.GetString("TRANSACTOR_FILECOIN_API"),
		TransactorFilecoinAuthToken:  viper.GetString("TRANSACTOR_FILECOIN_AUTH_TOKEN"),
		TransactorFilecoinConfidence: uint64(confidence),
		ActiveOutLoggingLevel:        viper.GetString("ACTIVEOUT_LOGGING_LEVEL"),
		ActiveOutDSTimeout:           viper.GetDuration("ACTIVEOUT_DS_TIMEOUT"),
		ActiveOutDSRetry:             uint64(viper.GetInt("ACTIVEOUT_DS_RETRY")),
		InactiveOutLoggingLevel:      viper.GetString("INACTIVEOUT_LOGGING_LEVEL"),
		InactiveOutDSTimeout:         viper.GetDuration("INACTIVEOUT_DS_TIMEOUT"),
		InactiveOutDSRetry:           uint64(viper.GetInt("INACTIVEOUT_DS_RETRY")),
		ActiveInLoggingLevel:         viper.GetString("ACTIVEIN_LOGGING_LEVEL"),
		ActiveInDSTimeout:            viper.GetDuration("ACTIVEIN_DS_TIMEOUT"),
		ActiveInDSRetry:              uint64(viper.GetInt("ACTIVEIN_DS_RETRY")),
		InactiveInLoggingLevel:       viper.GetString("INACTIVEIN_LOGGING_LEVEL"),
		InactiveInDSTimeout:          viper.GetDuration("INACTIVEIN_DS_TIMEOUT"),
		InactiveInDSRetry:            uint64(viper.GetInt("INACTIVEIN_DS_RETRY")),
		PServMgrLoggingLevel:         viper.GetString("PSERVMGR_LOGGING_LEVEL"),
		RouteStoreLoggingLevel:       viper.GetString("ROUTESTORE_LOGGING_LEVEL"),
		RouteStoreCleanFreq:          viper.GetDuration("ROUTESTORE_CLEAN_FREQ"),
		RouteStoreCleanTimeout:       viper.GetDuration("ROUTESTORE_CLEAN_TIMEOUT"),
		RouteStoreMaxHopFIL:          uint64(maxHopFIL),
		SubStoreLoggingLevel:         viper.GetString("SUBSTORE_LOGGING_LEVEL"),
		PayMgrLoggingLevel:           viper.GetString("PAYMGR_LOGGING_LEVEL"),
		PayMgrCacheSyncFreq:          viper.GetDuration("PAYMGR_CACHE_SYNC_FREQ"),
		PayMgrResCleanFreq:           viper.GetDuration("PAYMGR_RES_CLEAN_FREQ"),
		PayMgrResCleanTimeout:        viper.GetDuration("PAYMGR_RES_CLEAN_TIMEOUT"),
		PayMgrPeerCleanFreq:          viper.GetDuration("PAYMGR_PEER_CLEAN_FREQ"),
		PayMgrPeerCleanTimeout:       viper.GetDuration("PAYMGR_PEER_CLEAN_TIMEOUT"),
		SettleMgrLoggingLevel:        viper.GetString("SETTLEMGR_LOGGING_LEVEL"),
		RenewMgrLoggingLevel:         viper.GetString("RENEWMGR_LOGGING_LEVEL"),
		ReservMgrLoggingLevel:        viper.GetString("RESERVMGR_LOGGING_LEVEL"),
		OfferMgrLoggingLevel:         viper.GetString("OFFERMGR_LOGGING_LEVEL"),
		PaychMonitorLoggingLevel:     viper.GetString("PAYCHMONITOR_LOGGING_LEVEL"),
		PaychMonitorCheckFreq:        viper.GetDuration("PAYCHMONITOR_CHECK_FREQ"),
		PieceMgrLoggingLevel:         viper.GetString("PIECEMGR_LOGGING_LEVEL"),
		CServMgrLoggingLevel:         viper.GetString("CSERVMGR_LOGGING_LEVEL"),
		MinerProofStoreLoggingLevel:  viper.GetString("MINERPROOFSTORE_LOGGING_LEVEL"),
		AddrProtoLoggingLevel:        viper.GetString("ADDRPROTO_LOGGING_LEVEL"),
		AddrProtoIOTimeout:           viper.GetDuration("ADDRPROTO_IO_TIMEOUT"),
		AddrProtoOPTimeout:           viper.GetDuration("ADDRPROTO_OP_TIMEOUT"),
		AddrProtoPublishFreq:         viper.GetDuration("ADDRPROTO_PUBLISH_FREQ"),
		PaychProtoLoggingLevel:       viper.GetString("PAYCHPROTO_LOGGING_LEVEL"),
		PaychProtoIOTimeout:          viper.GetDuration("PAYCHPROTO_IO_TIMEOUT"),
		PaychProtoOPTimeout:          viper.GetDuration("PAYCHPROTO_OP_TIMEOUT"),
		PaychProtoOfferExpiry:        viper.GetDuration("PAYCHPROTO_OFFER_EXPIRY"),
		PaychProtoRenewWindow:        uint64(viper.GetInt("PAYCHPROTO_RENEW_WINDOW")),
		RouteProtoLoggingLevel:       viper.GetString("ROUTEPROTO_LOGGING_LEVEL"),
		RouteProtoIOTimeout:          viper.GetDuration("ROUTEPROTO_IO_TIMEOUT"),
		RouteProtoOPTimeout:          viper.GetDuration("ROUTEPROTO_OP_TIMEOUT"),
		RouteProtoPublishFreq:        viper.GetDuration("ROUTEPROTO_PUBLISH_FREQ"),
		RouteProtoRouteExpiry:        viper.GetDuration("ROUTEPROTO_ROUTE_EXPIRY"),
		RouteProtoPublishWait:        viper.GetDuration("ROUTEPROTO_PUBLISH_WAIT"),
		POfferProtoLoggingLevel:      viper.GetString("POFFERPROTO_LOGGING_LEVEL"),
		POfferProtoIOTimeout:         viper.GetDuration("POFFERPROTO_IO_TIMEOUT"),
		POfferProtoOPTimeout:         viper.GetDuration("POFFERPROTO_OP_TIMEOUT"),
		POfferProtoOfferExpiry:       viper.GetDuration("POFFERPROTO_OFFER_EXPIRY"),
		POfferProtoOfferInactivity:   viper.GetDuration("POFFERPROTO_OFFER_INACTIVITY"),
		COfferProtoLoggingLevel:      viper.GetString("COFFERPROTO_LOGGING_LEVEL"),
		COfferProtoIOTimeout:         viper.GetDuration("COFFERPROTO_IO_TIMEOUT"),
		COfferProtoOPTimeout:         viper.GetDuration("COFFERPROTO_OP_TIMEOUT"),
		COfferProtoOfferExpiry:       viper.GetDuration("COFFERPROTO_OFFER_EXPIRY"),
		COfferProtoOfferInactivity:   viper.GetDuration("COFFERPROTO_OFFER_INACTIVITY"),
		COfferPublishFreq:            viper.GetDuration("COFFERPROTO_PUBLISH_FREQ"),
		PayProtoLoggingLevel:         viper.GetString("PAYPROTO_LOGGING_LEVEL"),
		PayProtoIOTimeout:            viper.GetDuration("PAYPROTO_IO_TIMEOUT"),
		PayProtoOPTimeout:            viper.GetDuration("PAYPROTO_OP_TIMEOUT"),
		PayProtoCleanFreq:            viper.GetDuration("PAYPROTO_CLEAN_FREQ"),
		PayProtoCleanTimeout:         viper.GetDuration("PAYPROTO_CLEAN_TIMEOUT"),
		RetMgrLoggingLevel:           viper.GetString("RETMGR_LOGGING_LEVEL"),
		RetMgrIOTimeout:              viper.GetDuration("RETMGR_IO_TIMEOUT"),
		RetMgrOPTimeout:              viper.GetDuration("RETMGR_OP_TIMEOUT"),
	}, nil
}
