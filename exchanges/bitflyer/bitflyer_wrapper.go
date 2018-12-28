package bitflyer

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// GetDefaultConfig returns a default exchange config
func (b *Bitflyer) GetDefaultConfig() (*config.ExchangeConfig, error) {
	b.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = b.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = common.JoinStrings(b.BaseCurrencies, ",")

	err := b.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if b.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = b.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Bitflyer
func (b *Bitflyer) SetDefaults() {
	b.Name = "Bitflyer"
	b.Enabled = true
	b.Verbose = true
	b.APIWithdrawPermissions = exchange.WithdrawCryptoViaWebsiteOnly | exchange.AutoWithdrawFiat
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true

	b.CurrencyPairs = exchange.CurrencyPairs{
		AssetTypes: assets.AssetTypes{
			assets.AssetTypeSpot,
			assets.AssetTypeFutures,
		},

		UseGlobalPairFormat: true,

		RequestFormat: config.CurrencyPairFormatConfig{
			Delimiter: "_",
			Uppercase: true,
		},
		ConfigFormat: config.CurrencyPairFormatConfig{
			Delimiter: "_",
			Uppercase: true,
		},
	}

	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: false,

			Trading: exchange.TradingSupported{
				Spot:    true,
				Futures: true,
			},

			RESTCapabilities: exchange.ProtocolFeatures{
				AutoPairUpdates: true,
				TickerBatching:  false,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	b.Requester = request.New(b.Name,
		request.NewRateLimit(time.Minute, bitflyerAuthRate),
		request.NewRateLimit(time.Minute, bitflyerUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	b.API.Endpoints.URLDefault = japanURL
	b.API.Endpoints.URL = b.API.Endpoints.URLDefault
	b.API.Endpoints.URLSecondaryDefault = chainAnalysis
	b.API.Endpoints.URLSecondary = b.API.Endpoints.URLSecondaryDefault
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *Bitflyer) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}

	return b.SetupDefaults(exch)
}

// Start starts the Bitflyer go routine
func (b *Bitflyer) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the Bitflyer wrapper
func (b *Bitflyer) Run() {
	if b.Verbose {
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.CurrencyPairs.Spot.Enabled), b.CurrencyPairs.Spot.Enabled)
	}

	if !b.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := b.UpdateTradablePairs(false)
	if err != nil {
		log.Printf("%s failed to update tradable pairs. Err: %s", b.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *Bitflyer) FetchTradablePairs(assetType assets.AssetType) ([]string, error) {
	pairs, err := b.GetMarkets()
	if err != nil {
		return nil, err
	}

	var products []string
	for _, info := range pairs {
		if info.Alias != "" && assetType == assets.AssetTypeFutures {
			products = append(products, info.Alias)
		} else if info.Alias == "" && assetType == assets.AssetTypeSpot && common.StringContains(info.ProductCode, "_") {
			products = append(products, info.ProductCode)
		}
	}
	return products, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *Bitflyer) UpdateTradablePairs(forceUpdate bool) error {
	for x := range b.CurrencyPairs.AssetTypes {
		a := b.CurrencyPairs.AssetTypes[x]
		pairs, err := b.FetchTradablePairs(a)
		if err != nil {
			return err
		}

		err = b.UpdatePairs(pairs, a, false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bitflyer) UpdateTicker(p pair.CurrencyPair, assetType assets.AssetType) (ticker.Price, error) {
	var tickerPrice ticker.Price

	p = b.CheckFXString(p)

	tickerNew, err := b.GetTicker(p.Pair().String())
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Pair = p
	tickerPrice.Ask = tickerNew.BestAsk
	tickerPrice.Bid = tickerNew.BestBid
	// tickerPrice.Low
	tickerPrice.Last = tickerNew.Last
	tickerPrice.Volume = tickerNew.Volume
	// tickerPrice.High
	ticker.ProcessTicker(b.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(b.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (b *Bitflyer) FetchTicker(p pair.CurrencyPair, assetType assets.AssetType) (ticker.Price, error) {
	tick, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// CheckFXString upgrades currency pair if needed
func (b *Bitflyer) CheckFXString(p pair.CurrencyPair) pair.CurrencyPair {
	if common.StringContains(p.FirstCurrency.String(), "FX") {
		p.FirstCurrency = "FX_BTC"
		return p
	}
	return p
}

// FetchOrderbook returns the orderbook for a currency pair
func (b *Bitflyer) FetchOrderbook(p pair.CurrencyPair, assetType assets.AssetType) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitflyer) UpdateOrderbook(p pair.CurrencyPair, assetType assets.AssetType) (orderbook.Base, error) {
	var orderBook orderbook.Base

	p = b.CheckFXString(p)

	orderbookNew, err := b.GetOrderBook(p.Pair().String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Price: orderbookNew.Asks[x].Price, Amount: orderbookNew.Asks[x].Size})
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Price: orderbookNew.Bids[x].Price, Amount: orderbookNew.Bids[x].Size})
	}

	orderbook.ProcessOrderbook(b.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(b.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies on the
// Bitflyer exchange
func (b *Bitflyer) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = b.GetName()
	// accountBalance, err := b.GetAccountBalance()
	// if err != nil {
	// 	return response, err
	// }
	if !b.Enabled {
		return response, errors.New("exchange not enabled")
	}

	// implement once authenticated requests are introduced

	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bitflyer) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Bitflyer) GetExchangeHistory(p pair.CurrencyPair, assetType assets.AssetType) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *Bitflyer) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse

	return submitOrderResponse, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bitflyer) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bitflyer) CancelOrder(order exchange.OrderCancellation) error {
	return common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Bitflyer) CancelAllOrders(orderCancellation exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	// TODO, implement BitFlyer API
	b.CancelAllExistingOrders()
	return exchange.CancelAllOrdersResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns information on a current open order
func (b *Bitflyer) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Bitflyer) GetDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bitflyer) WithdrawCryptocurrencyFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bitflyer) WithdrawFiatFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bitflyer) WithdrawFiatFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *Bitflyer) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (b *Bitflyer) GetWithdrawCapabilities() uint32 {
	return b.GetWithdrawPermissions()
}
