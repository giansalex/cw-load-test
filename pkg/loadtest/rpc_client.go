package loadtest

import (
	"net/http"
)

type RpcClient struct {
	client  *http.Client
	baseUrl string
}

func NewRpcClient(client *http.Client, baseUrl string) *RpcClient {
	return &RpcClient{client, baseUrl}
}

func (lcd *RpcClient) UnsafeFlushMempool() error {
	_, err := lcd.client.Get(lcd.baseUrl + "/unsafe_flush_mempool")
	if err != nil {
		return err
	}

	return nil
}
