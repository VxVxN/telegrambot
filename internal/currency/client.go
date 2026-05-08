package currency

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type cbrResponse struct {
	Valute map[string]cbrCurrency `json:"Valute"`
}

type cbrCurrency struct {
	Nominal int     `json:"Nominal"`
	Value   float64 `json:"Value"`
}

type coinGeckoResponse map[string]struct {
	USD float64 `json:"usd"`
}

var Symbols = map[string]string{
	"ethereum": "ETH",
	"bitcoin":  "BTC",
	"ripple":   "XRP",
}

type Client struct {
	http *http.Client
}

func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) FetchCryptoPrice(cryptoID string) (float64, error) {
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd", cryptoID)

	body, err := c.get(url)
	if err != nil {
		return 0, err
	}

	var result coinGeckoResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("parse response: %w", err)
	}

	if data, ok := result[cryptoID]; ok {
		return data.USD, nil
	}
	return 0, fmt.Errorf("price not found for %s", cryptoID)
}

func (c *Client) FetchUSDRate() (float64, error) {
	body, err := c.get("https://www.cbr-xml-daily.ru/daily_json.js")
	if err != nil {
		return 0, err
	}

	var resp cbrResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("parse response: %w", err)
	}

	usd, ok := resp.Valute["USD"]
	if !ok {
		return 0, fmt.Errorf("USD rate not found")
	}
	return usd.Value / float64(usd.Nominal), nil
}

func (c *Client) get(url string) ([]byte, error) {
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}
