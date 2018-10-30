package exchange

import (
	"sync"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// IBotExchange enforces standard functions for all exchanges supported in
// GoCryptoTrader
type IBotExchange interface {
	Setup(exch config.ExchangeConfig)
	Start(wg *sync.WaitGroup)
	SetDefaults()
	GetName() string
	IsEnabled() bool
	SetEnabled(bool)
	FetchTicker(currency pair.CurrencyPair, assetType string) (ticker.Price, error)
	UpdateTicker(currency pair.CurrencyPair, assetType string) (ticker.Price, error)
	FetchOrderbook(currency pair.CurrencyPair, assetType string) (orderbook.Base, error)
	UpdateOrderbook(currency pair.CurrencyPair, assetType string) (orderbook.Base, error)
	GetEnabledCurrencies() []pair.CurrencyPair
	GetAvailableCurrencies() []pair.CurrencyPair
	GetAccountInfo() (AccountInfo, error)
	GetAuthenticatedAPISupport() bool
	SetCurrencies(pairs []pair.CurrencyPair, enabledPairs bool) error
	GetAssetTypes() []string
	GetExchangeHistory(pair.CurrencyPair, string) ([]TradeHistory, error)
	SupportsAutoPairUpdates() bool
	SupportsRESTTickerBatchUpdates() bool
	GetLastPairsUpdateTime() int64

	GetWithdrawPermissions() uint32
	FormatWithdrawPermissions() string
	SupportsWithdrawPermissions(permissions uint32) bool

	GetFundingHistory() ([]FundHistory, error)
	SubmitOrder(p pair.CurrencyPair, side OrderSide, orderType OrderType, amount, price float64, clientID string) (SubmitOrderResponse, error)
	ModifyOrder(action ModifyOrder) (string, error)
	CancelOrder(order OrderCancellation) error
	CancelAllOrders(orders OrderCancellation) (CancelAllOrdersResponse, error)
	GetOrderInfo(orderID int64) (OrderDetail, error)
	GetDepositAddress(cryptocurrency pair.CurrencyItem) (string, error)

	WithdrawCryptocurrencyFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error)
	WithdrawFiatFunds(currency pair.CurrencyItem, amount float64) (string, error)

	SetHTTPClientUserAgent(ua string)
	GetHTTPClientUserAgent() string
	SetClientProxyAddress(addr string) error

	SupportsWebsocket() bool
	SupportsREST() bool
	IsWebsocketEnabled() bool
	GetWebsocket() (*Websocket, error)
}
