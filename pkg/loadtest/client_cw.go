package loadtest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	wasmUtils "github.com/CosmWasm/wasmd/x/wasm/client/utils"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/bytecodealliance/wasmtime-go"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/giansalex/cw-load-test/internal/logging"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
)

// MyABCIAppClientFactory creates instances of MyABCIAppClient
type MyABCIAppClientFactory struct {
	paraphrase string
	chainID    string
}

// MyABCIAppClientFactory implements loadtest.ClientFactory
var _ ClientFactory = (*MyABCIAppClientFactory)(nil)

// MyABCIAppClient is responsible for generating transactions. Only one client
// will be created per connection to the remote Tendermint RPC endpoint, and
// each client will be responsible for maintaining its own state in a
// thread-safe manner.
type MyABCIAppClient struct {
	txs      map[uint64][]byte
	rpc      *rpchttp.HTTP
	lcd      *LcdClient
	signer   *Signature
	keyInfo  keyring.Info
	txB      cosmosclient.TxBuilder
	logger   logging.Logger
	count    uint64
	max      uint64
	seq      uint64
	bloclP   uint64
	watFile  []byte
	contract string
	execMsg  string
}

// MyABCIAppClient implements loadtest.Client
var _ Client = (*MyABCIAppClient)(nil)

func NewABCIAppClientFactory(paraphrase, chainID string) *MyABCIAppClientFactory {
	return &MyABCIAppClientFactory{paraphrase, chainID}
}

func (f *MyABCIAppClientFactory) ValidateConfig(cfg Config) error {
	// Do any checks here that you need to ensure that the load test
	// configuration is compatible with your client.
	if cfg.BlockPeriod < 1 {
		return fmt.Errorf("block period shoul be greater than 0 (got %d)", cfg.BlockPeriod)
	}

	if cfg.GasPrices == "" {
		return errors.New("gas prices cannot empty (got %d)")
	}

	if cfg.Contract != "" && cfg.WatPath != "" {
		return errors.New("Only wat-file or contract can be specified")
	}

	if cfg.Contract == "" && cfg.WatPath == "" {
		return errors.New("wat-file or contract needs to be specified")
	}

	return nil
}

func (f *MyABCIAppClientFactory) NewClient(cfg Config) (Client, error) {
	signer := NewSignature(f.chainID).RegisterInterfaces(RegisterDefaultInterfaces)
	info, err := signer.Recover(f.paraphrase)
	if err != nil {
		return nil, err
	}
	address := info.GetAddress().String()
	logger := logging.NewLogrusLogger("cw-client")

	logger.Info("Wallet Address: " + address)

	client := &http.Client{}
	lcd := NewLcdClient(client, cfg.LcdEndpoint)
	u, _ := url.Parse(cfg.Endpoints[0])
	rpc, err := rpchttp.New("tcp://"+u.Host, "/websocket")
	if err != nil {
		return nil, err
	}

	txBuilder := signer.GetTxConfig().NewTxBuilder()
	fees, err := f.parseFees(cfg.GasPrices, cfg.Gas)
	if err != nil {
		return nil, err
	}

	txBuilder.SetFeeAmount(fees)
	txBuilder.SetGasLimit(cfg.Gas)

	var wat []byte
	if cfg.Contract == "" {
		wat, err = ioutil.ReadFile(cfg.WatPath)
		if err != nil {
			return nil, err
		}
	}

	abciClient := &MyABCIAppClient{
		signer:   signer,
		keyInfo:  info,
		txB:      txBuilder,
		logger:   logger,
		lcd:      lcd,
		rpc:      rpc,
		max:      uint64(cfg.Rate),
		bloclP:   uint64(cfg.BlockPeriod),
		count:    0,
		watFile:  wat,
		contract: cfg.Contract,
		execMsg:  cfg.ExecMsg,
	}
	return abciClient, nil
}

func (c *MyABCIAppClient) createWasmStoreMsg(address string, value uint64) cosmostypes.Msg {
	search := []byte("1555268")
	replace := []byte(strconv.FormatUint(value, 10))

	newWat := bytes.Replace(c.watFile, search, replace, 1)
	wasm, _ := wasmtime.Wat2Wasm(string(newWat))

	wasm, _ = wasmUtils.GzipIt(wasm)
	wasmMsg := &wasmtypes.MsgStoreCode{
		Sender:                address,
		WASMByteCode:          wasm,
		InstantiatePermission: nil,
	}

	return wasmMsg
}

func (c *MyABCIAppClient) createExecuteMsg(address string) cosmostypes.Msg {
	wasmMsg := &wasmtypes.MsgExecuteContract{
		Sender:   address,
		Contract: c.contract,
		Msg:      []byte(c.execMsg), // {"argon2":{"mem_cost":1000,"time_cost":10}}
		Funds:    cosmostypes.Coins{},
	}

	return wasmMsg
}

func (f *MyABCIAppClientFactory) parseFees(gasPrices string, gas uint64) (cosmostypes.Coins, error) {

	parsedGasPrices, err := cosmostypes.ParseDecCoins(gasPrices)
	if err != nil {
		return nil, err
	}

	glDec := cosmostypes.NewDec(int64(gas))

	// Derive the fees based on the provided gas prices, where
	// fee = ceil(gasPrice * gasLimit).
	fees := make(cosmostypes.Coins, len(parsedGasPrices))

	for i, gp := range parsedGasPrices {
		fee := gp.Amount.Mul(glDec)
		fees[i] = cosmostypes.NewCoin(gp.Denom, fee.Ceil().RoundInt())
	}

	return fees, nil
}

// GetAccount must return current account
func (c *MyABCIAppClient) GetAccount() (keyring.Info, error) {
	return c.keyInfo, nil
}

// GenerateTx must return the raw bytes that make up the transaction for your
// ABCI app. The conversion to base64 will automatically be handled by the
// loadtest package, so don't worry about that. Only return an error here if you
// want to completely fail the entire load test operation.
func (c *MyABCIAppClient) GenerateTx() ([]byte, error) {

	if c.count >= c.max {
		c.count = 0
	}

	if c.count == 0 {
		err := c.makeTxs()
		if err != nil {
			return nil, err
		}
	}
	seq := c.seq + c.count
	c.count++

	return c.txs[seq], nil
}

func (c *MyABCIAppClient) makeTxs() error {
	account, err := c.lcd.Account(c.keyInfo.GetAddress().String())
	if err != nil {
		return err
	}

	totalTxs := c.max
	accountNro, _ := strconv.ParseUint(account.Account.AccountNumber, 10, 64)
	sequence, _ := strconv.ParseUint(account.Account.Sequence, 10, 64)
	d, err := c.rpc.ABCIInfo(context.Background())
	if err != nil {
		return err
	}
	height := uint64(d.Response.LastBlockHeight)
	expireHeight := height + c.bloclP

	c.logger.Info(fmt.Sprintf("Sending %d txs (Block Expire: %d)", totalTxs, expireHeight))
	c.txB.SetTimeoutHeight(expireHeight)

	c.txs = nil
	txs := make(map[uint64][]byte, totalTxs)
	var i uint64
	for i = 0; i < c.max; i++ {
		seq := sequence + i

		var msg cosmostypes.Msg
		if c.contract != "" {
			msg = c.createExecuteMsg(c.keyInfo.GetAddress().String())
		} else {
			msg = c.createWasmStoreMsg(account.Account.Address, seq)
		}
		err = c.txB.SetMsgs(msg)
		if err != nil {
			return err
		}

		data, err := c.signer.Sign(accountNro, seq, c.txB)
		if err != nil {
			return err
		}

		txs[seq] = data
	}

	c.txs = txs
	c.seq = sequence

	return nil
}
