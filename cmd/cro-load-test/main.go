package main

import (
	"log"
	"os"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/giansalex/cw-load-test/pkg/loadtest"
)

const appLongDesc = `Load testing application for Tendermint with optional master/slave mode.
Generates large quantities of arbitrary transactions and submits those 
transactions to one or more Tendermint endpoints. By default, it assumes that
you are running the Crossfire Crypto.com chain on your Tendermint network.

To run the application in a similar fashion to cro-bench (STANDALONE mode):
    cw-load-test -c 1 -T 10 -r 1000 -s 250 \
        --broadcast-tx-method async \
        --endpoints ws://tm-endpoint1.somewhere.com:26657/websocket,ws://tm-endpoint2.somewhere.com:26657/websocket

To run the application in MASTER mode:
    cw-load-test \
        master \
        --expect-slaves 2 \
        --bind localhost:26670 \
        --shutdown-wait 60 \
        -c 1 -T 10 -r 1000 -s 250 \
        --broadcast-tx-method async \
        --endpoints ws://tm-endpoint1.somewhere.com:26657/websocket,ws://tm-endpoint2.somewhere.com:26657/websocket

To run the application in SLAVE mode:
    cw-load-test slave --master localhost:26680

NOTES:
* MASTER mode exposes a "/metrics" endpoint in Prometheus plain text format
  which shows total number of transactions and the status for the master and
  all connected slaves.
* The "--shutdown-wait" flag in MASTER mode is specifically to allow your 
  monitoring system some time to obtain the final Prometheus metrics from the
  metrics endpoint.
* In SLAVE mode, all load testing-related flags are ignored. The slave always 
  takes instructions from the master node it's connected to.
`

func main() {

	wallet := os.Getenv("WALLET")
	chainID := os.Getenv("CHAINID")
	if wallet == "" || chainID == "" {
		log.Fatal("Required WALLET, CHAINID environment")
	}

	configCro()

	appFactory := loadtest.NewABCIAppClientFactory(wallet, chainID)

	if err := loadtest.RegisterClientFactory("cro-crossfire", appFactory); err != nil {
		panic(err)
	}

	loadtest.Run(&loadtest.CLIConfig{
		AppName:              "cw-load-test",
		AppShortDesc:         "Load testing application for Crypto.com Chain",
		AppLongDesc:          appLongDesc,
		DefaultClientFactory: "cro-crossfire",
	})
}

func configCro() {
	config := cosmostypes.GetConfig()
	config.SetBech32PrefixForAccount(Bech32PrefixAccAddr, Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(Bech32PrefixValAddr, Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(Bech32PrefixConsAddr, Bech32PrefixConsPub)
	config.SetCoinType(CointType)

	config.Seal()
}
