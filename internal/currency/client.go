package currency

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
)

type cbrValCurs struct {
	Valute []cbrValute `xml:"Valute"`
}

type cbrValute struct {
	CharCode string `xml:"CharCode"`
	Nominal  int    `xml:"Nominal"`
	Value    string `xml:"Value"`
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
	body, err := c.get("https://www.cbr.ru/scripts/XML_daily.asp")
	if err != nil {
		return 0, err
	}

	// The feed is served as windows-1251 XML; decode it so the Cyrillic name
	// fields don't trip the decoder's UTF-8 validation.
	dec := xml.NewDecoder(bytes.NewReader(body))
	dec.CharsetReader = func(_ string, input io.Reader) (io.Reader, error) {
		return charmap.Windows1251.NewDecoder().Reader(input), nil
	}

	var resp cbrValCurs
	if err := dec.Decode(&resp); err != nil {
		return 0, fmt.Errorf("parse response: %w", err)
	}

	for _, v := range resp.Valute {
		if v.CharCode != "USD" {
			continue
		}
		// CBR uses a comma as the decimal separator ("76,4026").
		value, err := strconv.ParseFloat(strings.Replace(v.Value, ",", ".", 1), 64)
		if err != nil {
			return 0, fmt.Errorf("parse USD value %q: %w", v.Value, err)
		}
		if v.Nominal == 0 {
			v.Nominal = 1
		}
		return value / float64(v.Nominal), nil
	}
	return 0, fmt.Errorf("USD rate not found")
}

func (c *Client) get(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// cbr.ru returns 403 to the default Go user agent, so present a browser one.
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; telegrambot/1.0)")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}
