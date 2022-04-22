package loadtest

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/giansalex/cw-load-test/internal/logging"
	"github.com/gorilla/websocket"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	rpctypes "github.com/tendermint/tendermint/rpc/jsonrpc/types"
	"github.com/tendermint/tendermint/types"
)

const (
	connSendTimeout = 10 * time.Second
	// see https://github.com/tendermint/tendermint/blob/master/rpc/lib/server/handlers.go
	connPingPeriod = (30 * 9 / 10) * time.Second

	jsonRPCID = rpctypes.JSONRPCStringID("tm-load-test")

	defaultProgressCallbackInterval = 5 * time.Second
)

// Transactor represents a single wire-level connection to a Tendermint RPC
// endpoint, and this is responsible for sending transactions to that endpoint.
type Transactor struct {
	remoteAddr string  // The full URL of the remote WebSockets endpoint.
	config     *Config // The configuration for the load test.

	client            Client
	logger            logging.Logger
	conn              *websocket.Conn
	rpc               *rpchttp.HTTP
	lcd               *LcdClient
	broadcastTxMethod string
	wg                sync.WaitGroup

	// Rudimentary statistics
	statsMtx  sync.RWMutex
	startTime time.Time // When did the transaction sending start?
	txCount   int       // How many transactions have been sent.
	txBytes   int64     // How many transaction bytes have been sent, cumulatively.
	txRate    float64   // The number of transactions sent, per second.

	progressCallbackMtx      sync.RWMutex
	progressCallbackID       int                                      // A unique identifier for this transactor when calling the progress callback.
	progressCallbackInterval time.Duration                            // How frequently to call the progress update callback.
	progressCallback         func(id int, txCount int, txBytes int64) // Called with the total number of transactions executed so far.

	stopMtx     sync.RWMutex
	stop        bool
	listenBlock bool
	sequence    uint64
	stopErr     error // Did an error occur that triggered the stop?
	cancel      chan int
	accAddress  string
}

// NewTransactor initiates a WebSockets connection to the given host address.
// Must be a valid WebSockets URL, e.g. "ws://host:port/websocket"
func NewTransactor(remoteAddr string, config *Config) (*Transactor, error) {
	u, err := url.Parse(remoteAddr)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "ws" && u.Scheme != "wss" {
		return nil, fmt.Errorf("unsupported protocol: %s (only ws:// and wss:// are supported)", u.Scheme)
	}
	clientFactory, exists := clientFactories[config.ClientFactory]
	if !exists {
		return nil, fmt.Errorf("unrecognized client factory: %s", config.ClientFactory)
	}
	client, err := clientFactory.NewClient(*config)
	if err != nil {
		return nil, err
	}
	conn, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}
	rpc, err := rpchttp.New("tcp://"+u.Host, "/websocket")
	if err != nil {
		return nil, err
	}
	lcd := NewLcdClient(&http.Client{}, config.LcdEndpoint)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to connect to remote WebSockets endpoint %s: %s (status code %d)", remoteAddr, resp.Status, resp.StatusCode)
	}

	account, _ := client.GetAccount()
	address := account.GetAddress().String()

	logger := logging.NewLogrusLogger("transactor")
	logger.Info("Connected to remote Tendermint WebSockets RPC")
	return &Transactor{
		remoteAddr:               u.String(),
		config:                   config,
		client:                   client,
		logger:                   logger,
		conn:                     conn,
		rpc:                      rpc,
		lcd:                      lcd,
		broadcastTxMethod:        "broadcast_tx_" + config.BroadcastTxMethod,
		progressCallbackInterval: defaultProgressCallbackInterval,
		cancel:                   make(chan int),
		accAddress:               address,
	}, nil
}

func (t *Transactor) SetProgressCallback(id int, interval time.Duration, callback func(int, int, int64)) {
	t.progressCallbackMtx.Lock()
	t.progressCallbackID = id
	t.progressCallbackInterval = interval
	t.progressCallback = callback
	t.progressCallbackMtx.Unlock()
}

// Start kicks off the transactor's operations in separate goroutines (one for
// reading from the WebSockets endpoint, and one for writing to it).
func (t *Transactor) Start() {
	t.logger.Debug("Starting transactor")
	t.wg.Add(2)
	go t.receiveLoop()
	go t.sendLoop()
}

// Cancel will indicate to the transactor that it must stop, but does not wait
// until it has completely stopped. To wait, call the Transactor.Wait() method.
func (t *Transactor) Cancel() {
	t.setStop(fmt.Errorf("transactor operations cancelled"))
	t.cancel <- 1
}

// Wait will block until the transactor terminates.
func (t *Transactor) Wait() error {
	t.wg.Wait()
	t.stopMtx.RLock()
	defer t.stopMtx.RUnlock()
	return t.stopErr
}

// GetTxCount returns the total number of transactions sent thus far by this
// transactor.
func (t *Transactor) GetTxCount() int {
	t.statsMtx.RLock()
	defer t.statsMtx.RUnlock()
	return t.txCount
}

// GetTxBytes returns the cumulative total number of bytes (as transactions)
// sent thus far by this transactor.
func (t *Transactor) GetTxBytes() int64 {
	t.statsMtx.RLock()
	defer t.statsMtx.RUnlock()
	return t.txBytes
}

// GetTxRate returns the average number of transactions per second sent by
// this transactor over the duration of its operation.
func (t *Transactor) GetTxRate() float64 {
	t.statsMtx.RLock()
	defer t.statsMtx.RUnlock()
	return t.txRate
}

func (t *Transactor) receiveLoop() {
	defer t.wg.Done()
	for {
		// right now we don't care about what we read back from the RPC endpoint
		_, _, err := t.conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				t.logger.Error("Failed to read response on connection", "err", err)
				return
			}
		}
		// fmt.Println("MsgType", r)
		// fmt.Println("Data: ", string(d))
		if t.mustStop() {
			return
		}
	}
}

func (t *Transactor) sendLoop() {
	defer t.wg.Done()
	t.conn.SetPingHandler(func(message string) error {
		err := t.conn.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(connSendTimeout))
		if err == websocket.ErrCloseSent {
			return nil
		} else if e, ok := err.(net.Error); ok && e.Temporary() {
			return nil
		}
		return err
	})

	err := t.rpc.Start()
	if err != nil {
		t.logger.Error("Cannot connect rpc", "err", err)
		t.setStop(err)
		return
	}
	defer t.rpc.Stop()
	block, err := t.listenBlocks()
	if err != nil {
		t.logger.Error("Cannot subscribe to listen blocks", "err", err)
		t.setStop(err)
		return
	}

	minRate := uint64(float32(t.config.Rate) * t.config.RatePercent)
	for {

		if t.mustStop() {
			t.close()
			return
		}

		currentSeq, err := t.getLatestSequence()
		if err != nil {
			t.logger.Error("Cannot get account from LCD", "err", err)
			t.setStop(err)
			continue
		}

		if err := t.sendTransactions(); err != nil {
			t.logger.Error("Failed to send transactions", "err", err)
			t.setStop(err)
			continue
		}

		minSeqRequired := currentSeq + minRate
		t.logger.Info(fmt.Sprintf("Target Sequence: %d", minSeqRequired))
		t.setSequenceRequired(minSeqRequired)

		t.setListenBlock(true)
		select {
		case <-block:
		case <-t.cancel:
			t.logger.Info("Cancel Listened")
		}
		t.setListenBlock(false)
	}
}

func (t *Transactor) writeTx(tx []byte) error {
	txBase64 := base64.StdEncoding.EncodeToString(tx)
	paramsJSON, err := json.Marshal(map[string]interface{}{"tx": txBase64})
	if err != nil {
		return err
	}
	_ = t.conn.SetWriteDeadline(time.Now().Add(connSendTimeout))
	return t.conn.WriteJSON(rpctypes.RPCRequest{
		JSONRPC: "2.0",
		ID:      jsonRPCID,
		Method:  t.broadcastTxMethod,
		Params:  json.RawMessage(paramsJSON),
	})
}

func (t *Transactor) mustStop() bool {
	t.stopMtx.RLock()
	defer t.stopMtx.RUnlock()
	return t.stop
}

func (t *Transactor) setStop(err error) {
	t.stopMtx.Lock()
	t.stop = true
	if err != nil {
		t.stopErr = err
	}
	t.stopMtx.Unlock()
}

func (t *Transactor) mustListenBlock() bool {
	t.stopMtx.RLock()
	defer t.stopMtx.RUnlock()
	return t.listenBlock
}

func (t *Transactor) setListenBlock(stop bool) {
	t.stopMtx.Lock()
	t.listenBlock = stop
	t.stopMtx.Unlock()
}

func (t *Transactor) getSequenceRequired() uint64 {
	t.stopMtx.RLock()
	defer t.stopMtx.RUnlock()
	return t.sequence
}

func (t *Transactor) setSequenceRequired(seq uint64) {
	t.stopMtx.Lock()
	t.sequence = seq
	t.stopMtx.Unlock()
}

func (t *Transactor) sendTransactions() error {
	toSend := t.config.Rate

	var sent int
	for ; sent < toSend; sent++ {
		tx, err := t.client.GenerateTx()
		if err != nil {
			return err
		}
		if err := t.writeTx(tx); err != nil {
			return err
		}
	}
	return nil
}

func (t *Transactor) listenBlocks() (<-chan int64, error) {
	query := "tm.event = 'NewBlock'"
	txs, err := t.rpc.Subscribe(context.Background(), "cw-load-test", query)
	if err != nil {
		return nil, err
	}

	out := make(chan int64)

	go func() {
		count := 0
		for e := range txs {
			switch data := e.Data.(type) {
			case types.EventDataNewBlock:
				if t.mustStop() {
					out <- 0
					return
				}

				if !t.mustListenBlock() {
					count = 0
					continue
				}
				count++

				currentSeq, err := t.getLatestSequence()
				if err != nil {
					t.logger.Error("Can't get account from lcd endpoint", "err", err)
					continue
				}

				seq := t.getSequenceRequired()

				if currentSeq >= seq || count >= t.config.BlockPeriod {
					count = 0
					eff := 1.0 - float64(seq-currentSeq)/float64(t.config.Rate)
					t.logger.Info(fmt.Sprintf("Sequence %d, block: %d", currentSeq, data.Block.Height))
					t.logger.Info(fmt.Sprintf("E: %.2f %%", eff*100.0))
					out <- data.Block.Height
				}
			}
		}
	}()

	return out, nil
}

func (t *Transactor) getLatestSequence() (uint64, error) {
	resp, err := t.lcd.Account(t.accAddress)
	if err != nil {
		return 0, err
	}

	return strconv.ParseUint(resp.Account.Sequence, 10, 64)
}
func (t *Transactor) trackStartTime() {
	t.statsMtx.Lock()
	t.startTime = time.Now()
	t.txRate = 0.0
	t.statsMtx.Unlock()
}

func (t *Transactor) trackSentTxs(count int, byteCount int64) {
	t.statsMtx.Lock()
	defer t.statsMtx.Unlock()

	t.txCount += count
	t.txBytes += byteCount
	elapsed := time.Since(t.startTime).Seconds()
	if elapsed > 0 {
		t.txRate = float64(t.txCount) / elapsed
	} else {
		t.txRate = 0
	}
}

func (t *Transactor) sendPing() error {
	_ = t.conn.SetWriteDeadline(time.Now().Add(connSendTimeout))
	return t.conn.WriteMessage(websocket.PingMessage, []byte{})
}

func (t *Transactor) reportProgress() {
	txCount := t.GetTxCount()
	txRate := t.GetTxRate()
	txBytes := t.GetTxBytes()
	t.logger.Debug("Statistics", "txCount", txCount, "txRate", fmt.Sprintf("%.3f txs/sec", txRate))

	t.progressCallbackMtx.RLock()
	defer t.progressCallbackMtx.RUnlock()
	if t.progressCallback != nil {
		t.progressCallback(t.progressCallbackID, txCount, txBytes)
	}
}

func (t *Transactor) getProgressCallbackInterval() time.Duration {
	t.progressCallbackMtx.RLock()
	defer t.progressCallbackMtx.RUnlock()
	return t.progressCallbackInterval
}

func (t *Transactor) close() {
	// try to cleanly shut down the connection
	_ = t.conn.SetWriteDeadline(time.Now().Add(connSendTimeout))
	err := t.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		t.logger.Error("Failed to write close message", "err", err)
	} else {
		t.logger.Debug("Wrote close message to remote endpoint")
	}
}
