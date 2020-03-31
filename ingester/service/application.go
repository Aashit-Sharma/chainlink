package service

import (
	"database/sql"
	// "errors"
	"fmt"

	"chainlink/ingester/client"
	"chainlink/ingester/logger"

	// "github.com/ethereum/go-ethereum"
	// "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	_ "github.com/jinzhu/gorm/dialects/postgres" // http://doc.gorm.io/database.html#connecting-to-a-database
)

// Application is an instance of the aggregator monitor application containing
// all clients and services
type Application struct {
	Feeds  FeedsTracker
	Config *Config

	ETHClient client.ETH
}

// InterruptHandler is a function that is called after application startup
// designed to wait based on a specified interrupt
type InterruptHandler func()

// NewApplication returns an instance of the Application with
// all clients connected and services instantiated
func NewApplication(config *Config) (*Application, error) {
	logger.SetLogger(logger.CreateTestLogger(-1))

	logger.Infow(
		"Starting the Chainlink Ingester",
		"eth-url", config.EthereumURL,
		"eth-chain-id", config.NetworkID,
		"db-host", config.DatabaseHost,
		"db-name", config.DatabaseName,
		"db-port", config.DatabasePort,
		"db-username", config.DatabaseUsername,
	)

	psqlInfo := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.DatabaseHost,
		config.DatabasePort,
		config.DatabaseUsername,
		config.DatabasePassword,
		config.DatabaseName,
	)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	ec, err := client.NewClient(config.EthereumURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to ETH client: %+v", err)
	}

	// e := a.abi.Events[NewRoundEventName]
	// q := ethereum.FilterQuery{
	// 	Addresses: []common.Address{a.address},
	// 	Topics:    [][]common.Hash{{e.ID()}},
	// }

	// f := client.NewFeedsUI(config.FeedsUIURL)
	// feeds, err := f.Feeds()
	// for f := range feeds {
	// logger.Infow("-------- FEED:", "name", f.Name)
	// }

	// q := ethereum.FilterQuery{}
	// logChan := make(chan types.Log)
	// _, err = ec.SubscribeToLogs(logChan, q)
	// if err != nil {
	// 	return nil, err
	// }

	// go func() {
	// 	logger.Debug("Listening for logs")
	// 	for log := range logChan {
	// 		address := make([]byte, 20)
	// 		copy(address, log.Address[:])

	// 		topics := make([]byte, len(log.Topics)*len(common.Hash{}))
	// 		for index, topic := range log.Topics {
	// 			copy(topics[index*len(common.Hash{}):], topic.Bytes())
	// 		}

	// 		_, err := db.Exec(`INSERT INTO "ethereum_log" ("address", "topics", "data", "blockNumber", "txHash", "txIndex", "blockHash", "index", "removed") VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);`,
	// 			address,
	// 			topics,
	// 			log.Data,
	// 			log.BlockNumber,
	// 			log.TxHash.Bytes(),
	// 			log.TxIndex,
	// 			log.BlockHash.Bytes(),
	// 			log.Index,
	// 			log.Removed)
	// 		if err != nil {
	// 			logger.Errorw("Insert failed", "error", err)
	// 		}

	// 		// logger.Debugw("Oberved new log", "blockHash", log.BlockHash, "index", log.Index, "removed", log.Removed)
	// 		// logger.Debugw("Oberved new log", "topics", topics)

	// // if (len(log.Topics) == 3) {
	// //   logger.Debugw("-------- NewRoundEvent:", "topics byte[]", topics, "log topics", log.Topics, "data", log.Data)
	// //   nr, _ := UnmarshalNewRoundEvent(log)
	// //   // if (err) {
	// //   //   logger.Errorw("could not unmarshal NewRoundEvent", "topics", log.Topics)
	// //   // }
	// //   logger.Debugw("-------- NewRoundEvent unmarshaled", "Round ID", nr.RoundID, "Started By", nr.StartedBy)
	// // } else if (len(log.Topics) == 4) {
	// //   // logger.Debugw("******** ResponseReceived:", "topics byte[]", topics, "log topics", log.Topics)
	// //   // logger.Debugw("******** ResponseReceived")
	// // } else {
	// //   // logger.Debugw("******** Other event:", "len", len(log.Topics), "topics byte[]", topics, "log topics", log.Topics)
	// //   // logger.Debugw("******** Other event")
	// // }
	// 	}
	// }()

	headChan := make(chan types.Header)
	_, err = ec.SubscribeToNewHeads(headChan)
	if err != nil {
		return nil, err
	}

	go func() {
		logger.Debug("Listening for heads")
		for head := range headChan {
			nonce := make([]byte, 8)
			copy(nonce, head.Nonce[:])

			logger.Debugw("Observed new head", "blockHeight", head.Number, "blockHash", head.Hash())
			_, err := db.Exec(`INSERT INTO "ethereum_head" ("blockHash", "parentHash", "uncleHash", "coinbase", "root", "txHash", "receiptHash", "bloom", "difficulty", "number", "gasLimit", "gasUsed", "time", "extra", "mixDigest", "nonce") VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16);`,
				head.Hash().Bytes(),
				head.ParentHash,
				head.UncleHash,
				head.Coinbase,
				head.Root,
				head.TxHash,
				head.ReceiptHash,
				head.Bloom.Bytes(),
				head.Difficulty.String(),
				head.Number.String(),
				head.GasLimit,
				head.GasUsed,
				head.Time,
				head.Extra,
				head.MixDigest,
				nonce)
			if err != nil {
				logger.Errorw("Insert failed", "error", err)
			}
		}
	}()

	ft := NewFeedsTracker(ec, client.NewFeedsUI(config.FeedsUIURL), config.NetworkID)

	return &Application{
		ETHClient: ec,
		Config:    config,
		Feeds:     ft,
	}, nil
}

// Start will start all the services within the application and call the interrupt handler
func (a *Application) Start(ih InterruptHandler) {
	a.Feeds.Start()

	ih()
}

// Stop will call each services that requires a clean shutdown to stop
func (a *Application) Stop() {
	logger.Info("Shutting down")
}
