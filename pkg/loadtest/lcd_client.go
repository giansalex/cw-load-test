package loadtest

import (
	"encoding/json"
	"net/http"
)

type LcdClient struct {
	client  *http.Client
	baseUrl string
}

func NewLcdClient(client *http.Client, baseUrl string) *LcdClient {
	return &LcdClient{client, baseUrl}
}

func (lcd *LcdClient) Account(address string) (*AccountResponse, error) {
	resp, err := lcd.client.Get(lcd.baseUrl + "/cosmos/auth/v1beta1/accounts/" + address)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var data AccountResponse
	err = decoder.Decode(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (lcd *LcdClient) Balances(address string) (*BalancesResponse, error) {
	resp, err := lcd.client.Get(lcd.baseUrl + "/cosmos/bank/v1beta1/balances/" + address)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var data BalancesResponse
	err = decoder.Decode(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

type AccountResponse struct {
	Account struct {
		Type    string `json:"@type"`
		Address string `json:"address"`
		PubKey  struct {
			Type string `json:"@type"`
			Key  string `json:"key"`
		} `json:"pub_key"`
		AccountNumber string `json:"account_number"`
		Sequence      string `json:"sequence"`
	} `json:"account"`
}

type BalancesResponse struct {
	Balances []struct {
		Denom  string `json:"denom"`
		Amount string `json:"amount"`
	} `json:"balances"`
}
