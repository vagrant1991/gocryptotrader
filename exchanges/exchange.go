package exchange

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
)

const (
	warningBase64DecryptSecretKeyFailed = "WARNING -- Exchange %s unable to base64 decode secret key.. Disabling Authenticated API support."
	// WarningAuthenticatedRequestWithoutCredentialsSet error message for authenticated request without credentials set
	WarningAuthenticatedRequestWithoutCredentialsSet = "WARNING -- Exchange %s authenticated HTTP request called but not supported due to unset/default API credentials."
	// DefaultHTTPTimeout is the default HTTP/HTTPS Timeout for exchange requests
	DefaultHTTPTimeout = time.Second * 15
)

func (e *Base) checkAndInitRequester() {
	if e.Requester == nil {
		e.Requester = request.New(e.Name,
			request.NewRateLimit(time.Second, 0),
			request.NewRateLimit(time.Second, 0),
			new(http.Client))
	}
}

// SetHTTPClientTimeout sets the timeout value for the exchanges
// HTTP Client
func (e *Base) SetHTTPClientTimeout(t time.Duration) {
	e.checkAndInitRequester()
	e.Requester.HTTPClient.Timeout = t
}

// SetHTTPClient sets exchanges HTTP client
func (e *Base) SetHTTPClient(h *http.Client) {
	e.checkAndInitRequester()
	e.Requester.HTTPClient = h
}

// GetHTTPClient gets the exchanges HTTP client
func (e *Base) GetHTTPClient() *http.Client {
	e.checkAndInitRequester()
	return e.Requester.HTTPClient
}

// SetHTTPClientUserAgent sets the exchanges HTTP user agent
func (e *Base) SetHTTPClientUserAgent(ua string) {
	e.checkAndInitRequester()
	e.Requester.UserAgent = ua
	e.HTTPUserAgent = ua
}

// GetHTTPClientUserAgent gets the exchanges HTTP user agent
func (e *Base) GetHTTPClientUserAgent() string {
	return e.HTTPUserAgent
}

// SetClientProxyAddress sets a proxy address for REST and websocket requests
func (e *Base) SetClientProxyAddress(addr string) error {
	if addr != "" {
		proxy, err := url.Parse(addr)
		if err != nil {
			return fmt.Errorf("exchange.go - setting proxy address error %s",
				err)
		}

		err = e.Requester.SetProxy(proxy)
		if err != nil {
			return fmt.Errorf("exchange.go - setting proxy address error %s",
				err)
		}

		if e.Websocket != nil {
			err = e.Websocket.SetProxyAddress(addr)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// SetFeatureDefaults sets the exchanges default feature
// support set
func (e *Base) SetFeatureDefaults() {
	if e.Config.Features == nil {
		s := &config.FeaturesConfig{
			Supports: config.FeaturesSupportedConfig{
				Websocket: e.Features.Supports.Websocket,
				REST:      e.Features.Supports.REST,
				RESTCapabilities: config.ProtocolFeaturesConfig{
					AutoPairUpdates: e.Features.Supports.RESTCapabilities.AutoPairUpdates,
				},
				Trading: config.TradingConfig{
					Spot:           e.Features.Supports.Trading.Spot,
					Futures:        e.Features.Supports.Trading.Futures,
					Margin:         e.Features.Supports.Trading.Margin,
					PerpetualSwaps: e.Features.Supports.Trading.PerpetualSwaps,
					Index:          e.Features.Supports.Trading.Index,
				},
			},
		}

		if e.Config.SupportsAutoPairUpdates != nil {
			s.Supports.RESTCapabilities.AutoPairUpdates = *e.Config.SupportsAutoPairUpdates
			s.Enabled.AutoPairUpdates = *e.Config.SupportsAutoPairUpdates
		} else {
			s.Supports.RESTCapabilities.AutoPairUpdates = e.Features.Supports.RESTCapabilities.AutoPairUpdates
			s.Enabled.AutoPairUpdates = e.Features.Supports.RESTCapabilities.AutoPairUpdates
			if !s.Supports.RESTCapabilities.AutoPairUpdates {
				e.Config.CurrencyPairs.LastUpdated = time.Now().Unix()
				e.CurrencyPairs.LastUpdated = e.Config.CurrencyPairs.LastUpdated
			}
		}
		e.Config.Features = s
		e.Config.SupportsAutoPairUpdates = nil
	} else {
		e.Config.Features.Supports.Trading = config.TradingConfig{
			Spot:           e.Features.Supports.Trading.Spot,
			Futures:        e.Features.Supports.Trading.Futures,
			Margin:         e.Features.Supports.Trading.Margin,
			PerpetualSwaps: e.Features.Supports.Trading.PerpetualSwaps,
			Index:          e.Features.Supports.Trading.Index,
		}

		if e.Features.Supports.RESTCapabilities.AutoPairUpdates != e.Config.Features.Supports.RESTCapabilities.AutoPairUpdates {
			e.Config.Features.Supports.RESTCapabilities.AutoPairUpdates = e.Features.Supports.RESTCapabilities.AutoPairUpdates

			if !e.Config.Features.Supports.RESTCapabilities.AutoPairUpdates {
				e.Config.CurrencyPairs.LastUpdated = time.Now().Unix()
			}
		}

		if e.Features.Supports.REST != e.Config.Features.Supports.REST {
			e.Config.Features.Supports.REST = e.Features.Supports.REST
		}

		if e.Features.Supports.RESTCapabilities.AutoPairUpdates != e.Config.Features.Supports.RESTCapabilities.TickerBatching {
			e.Config.Features.Supports.RESTCapabilities.TickerBatching = e.Features.Supports.RESTCapabilities.AutoPairUpdates
		}

		if e.Features.Supports.Websocket != e.Config.Features.Supports.Websocket {
			e.Config.Features.Supports.Websocket = e.Features.Supports.Websocket
		}

		e.Features.Enabled.AutoPairUpdates = e.Config.Features.Enabled.AutoPairUpdates
	}
}

// SetAPICredentialDefaults sets the API Credential validator defaults
func (e *Base) SetAPICredentialDefaults() {
	// Exchange hardcoded settings take precedence and overwrite the config settings
	if e.Config.API.CredentialsValidator.RequiresKey != e.API.CredentialsValidator.RequiresKey {
		e.Config.API.CredentialsValidator.RequiresKey = e.API.CredentialsValidator.RequiresKey
	}

	if e.Config.API.CredentialsValidator.RequiresSecret != e.API.CredentialsValidator.RequiresSecret {
		e.Config.API.CredentialsValidator.RequiresSecret = e.API.CredentialsValidator.RequiresSecret
	}

	if e.Config.API.CredentialsValidator.RequiresBase64DecodeSecret != e.API.CredentialsValidator.RequiresBase64DecodeSecret {
		e.Config.API.CredentialsValidator.RequiresBase64DecodeSecret = e.API.CredentialsValidator.RequiresBase64DecodeSecret
	}

	if e.Config.API.CredentialsValidator.RequiresClientID != e.API.CredentialsValidator.RequiresClientID {
		e.Config.API.CredentialsValidator.RequiresClientID = e.API.CredentialsValidator.RequiresClientID
	}

	if e.Config.API.CredentialsValidator.RequiresPEM != e.API.CredentialsValidator.RequiresPEM {
		e.Config.API.CredentialsValidator.RequiresPEM = e.API.CredentialsValidator.RequiresPEM
	}
}

// SetHTTPRateLimiter sets the exchanges default HTTP rate limiter and updates the exchange's config
// to default settings if it doesn't exist
func (e *Base) SetHTTPRateLimiter() {
	e.checkAndInitRequester()

	if e.RequiresRateLimiter() {
		if e.Config.HTTPRateLimiter == nil {
			e.Config.HTTPRateLimiter = new(config.HTTPRateLimitConfig)
			e.Config.HTTPRateLimiter.Authenticated.Duration = e.GetRateLimit(true).Duration
			e.Config.HTTPRateLimiter.Authenticated.Rate = e.GetRateLimit(true).Rate
			e.Config.HTTPRateLimiter.Unauthenticated.Duration = e.GetRateLimit(false).Duration
			e.Config.HTTPRateLimiter.Unauthenticated.Rate = e.GetRateLimit(false).Rate
		} else {
			e.SetRateLimit(true, e.Config.HTTPRateLimiter.Authenticated.Duration,
				e.Config.HTTPRateLimiter.Authenticated.Rate)
			e.SetRateLimit(false, e.Config.HTTPRateLimiter.Unauthenticated.Duration,
				e.Config.HTTPRateLimiter.Unauthenticated.Rate)
		}
	}
}

// SupportsRESTTickerBatchUpdates returns whether or not the
// exhange supports REST batch ticker fetching
func (e *Base) SupportsRESTTickerBatchUpdates() bool {
	return e.Features.Supports.RESTCapabilities.TickerBatching
}

// SupportsAutoPairUpdates returns whether or not the exchange supports
// auto currency pair updating
func (e *Base) SupportsAutoPairUpdates() bool {
	if e.Features.Supports.RESTCapabilities.AutoPairUpdates || e.Features.Supports.WebsocketCapabilities.AutoPairUpdates {
		return true
	}
	return false
}

// GetLastPairsUpdateTime returns the unix timestamp of when the exchanges
// currency pairs were last updated
func (e *Base) GetLastPairsUpdateTime() int64 {
	return e.CurrencyPairs.LastUpdated
}

// SetAssetTypes checks the exchange asset types (whether it supports SPOT,
// Binary or Futures) and sets it to a default setting if it doesn't exist
func (e *Base) SetAssetTypes() {
	if e.Config.CurrencyPairs.AssetTypes == "" {
		e.Config.CurrencyPairs.AssetTypes = e.CurrencyPairs.AssetTypes.JoinToString(",")
	} else {
		if e.Config.CurrencyPairs.AssetTypes != e.CurrencyPairs.AssetTypes.JoinToString(",") {
			e.Config.CurrencyPairs.AssetTypes = e.CurrencyPairs.AssetTypes.JoinToString(",")
			if e.IsAssetTypeSupported(assets.AssetTypeFutures) {
				e.Config.CurrencyPairs.Futures = new(config.CurrencyPairConfig)
			}
		}
	}
}

// GetAssetTypes returns the available asset types for an individual exchange
func (e *Base) GetAssetTypes() assets.AssetTypes {
	return e.CurrencyPairs.AssetTypes
}

// GetClientBankAccounts returns banking details associated with
// a client for withdrawal purposes
func (e *Base) GetClientBankAccounts(exchangeName, withdrawalCurrency string) (config.BankAccount, error) {
	cfg := config.GetConfig()
	return cfg.GetClientBankAccounts(exchangeName, withdrawalCurrency)
}

// GetExchangeBankAccounts returns banking details associated with an
// exchange for funding purposes
func (e *Base) GetExchangeBankAccounts(exchangeName, depositCurrency string) (config.BankAccount, error) {
	cfg := config.GetConfig()
	return cfg.GetExchangeBankAccounts(exchangeName, depositCurrency)
}

// CompareCurrencyPairFormats checks and returns whether or not the two supplied
// config currency pairs match
func CompareCurrencyPairFormats(pair1 config.CurrencyPairFormatConfig, pair2 *config.CurrencyPairFormatConfig) bool {
	if pair1.Delimiter != pair2.Delimiter ||
		pair1.Uppercase != pair2.Uppercase ||
		pair1.Separator != pair2.Separator ||
		pair1.Index != pair2.Index {
		return false
	}
	return true
}

// SetCurrencyPairFormat checks the exchange request and config currency pair
// formats and sets it to a default setting if it doesn't exist
func (e *Base) SetCurrencyPairFormat() {
	if e.Config.CurrencyPairs == nil {
		e.Config.CurrencyPairs = new(config.CurrencyPairsConfig)
	}

	if e.Features.Supports.Trading.Spot {
		if e.Config.CurrencyPairs.Spot == nil {
			e.Config.CurrencyPairs.Spot = new(config.CurrencyPairConfig)
		}
	}

	if e.CurrencyPairs.UseGlobalPairFormat {
		if e.Config.CurrencyPairs.RequestFormat == nil {
			e.Config.CurrencyPairs.RequestFormat = &config.CurrencyPairFormatConfig{
				Delimiter: e.CurrencyPairs.RequestFormat.Delimiter,
				Uppercase: e.CurrencyPairs.RequestFormat.Uppercase,
				Separator: e.CurrencyPairs.RequestFormat.Separator,
				Index:     e.CurrencyPairs.RequestFormat.Index,
			}
		}

		if e.Config.CurrencyPairs.ConfigFormat == nil {
			e.Config.CurrencyPairs.ConfigFormat = &config.CurrencyPairFormatConfig{
				Delimiter: e.CurrencyPairs.ConfigFormat.Delimiter,
				Uppercase: e.CurrencyPairs.ConfigFormat.Uppercase,
				Separator: e.CurrencyPairs.ConfigFormat.Separator,
				Index:     e.CurrencyPairs.ConfigFormat.Index,
			}
		}
	} else {
		if e.Features.Supports.Trading.Spot {
			if e.Config.CurrencyPairs.Spot.ConfigFormat == nil {
				e.Config.CurrencyPairs.Spot.ConfigFormat = &config.CurrencyPairFormatConfig{
					Delimiter: e.CurrencyPairs.Spot.ConfigFormat.Delimiter,
					Uppercase: e.CurrencyPairs.Spot.ConfigFormat.Uppercase,
					Separator: e.CurrencyPairs.Spot.ConfigFormat.Separator,
					Index:     e.CurrencyPairs.Spot.ConfigFormat.Index,
				}
			}
			if e.Config.CurrencyPairs.Spot.RequestFormat == nil {
				e.Config.CurrencyPairs.Spot.RequestFormat = &config.CurrencyPairFormatConfig{
					Delimiter: e.CurrencyPairs.Spot.RequestFormat.Delimiter,
					Uppercase: e.CurrencyPairs.Spot.RequestFormat.Uppercase,
					Separator: e.CurrencyPairs.Spot.RequestFormat.Separator,
					Index:     e.CurrencyPairs.Spot.RequestFormat.Index,
				}
			}
		}
	}
}

// GetAuthenticatedAPISupport returns whether the exchange supports
// authenticated API requests
func (e *Base) GetAuthenticatedAPISupport() bool {
	return e.API.AuthenticatedSupport
}

// GetName is a method that returns the name of the exchange base
func (e *Base) GetName() string {
	return e.Name
}

// GetEnabledFeatures returns the exchanges enabled features
func (e *Base) GetEnabledFeatures() FeaturesEnabled {
	return e.Features.Enabled
}

// GetSupportedFeatures returns the exchanges supported features
func (e *Base) GetSupportedFeatures() FeaturesSupported {
	return e.Features.Supports
}

// GetPairFormat returns the pair format based on the exchange and
// asset type
func (e *Base) GetPairFormat(assetType assets.AssetType, requestFormat bool) config.CurrencyPairFormatConfig {
	if e.CurrencyPairs.UseGlobalPairFormat {
		if requestFormat {
			return e.CurrencyPairs.RequestFormat
		}
		return e.CurrencyPairs.ConfigFormat
	}

	if assetType == assets.AssetTypeSpot {
		if requestFormat {
			return e.CurrencyPairs.Spot.RequestFormat
		}
		return e.CurrencyPairs.Spot.ConfigFormat
	}

	if requestFormat {
		return e.CurrencyPairs.Futures.RequestFormat
	}
	return e.CurrencyPairs.Futures.ConfigFormat
}

// GetEnabledPairs is a method that returns the enabled currency pairs of
// the exchange by asset type
func (e *Base) GetEnabledPairs(assetType assets.AssetType) []pair.CurrencyPair {
	format := e.GetPairFormat(assetType, false)

	if assetType == assets.AssetTypeSpot {
		return pair.FormatPairs(e.CurrencyPairs.Spot.Enabled,
			format.Delimiter,
			format.Index)
	}

	return pair.FormatPairs(e.CurrencyPairs.Futures.Enabled,
		format.Delimiter,
		format.Index)
}

// GetAvailablePairs is a method that returns the available currency pairs
// of the exchange by asset type
func (e *Base) GetAvailablePairs(assetType assets.AssetType) []pair.CurrencyPair {
	format := e.GetPairFormat(assetType, false)

	if assetType == assets.AssetTypeSpot {
		return pair.FormatPairs(e.CurrencyPairs.Spot.Available,
			format.Delimiter,
			format.Index)
	}

	return pair.FormatPairs(e.CurrencyPairs.Futures.Available,
		format.Delimiter,
		format.Index)
}

// SupportsPair returns true or not whether a currency pair exists in the
// exchange available currencies or not
func (e *Base) SupportsPair(p pair.CurrencyPair, enabledPairs bool, assetType assets.AssetType) bool {
	if enabledPairs {
		return pair.Contains(e.GetEnabledPairs(assetType), p, false)
	}
	return pair.Contains(e.GetAvailablePairs(assetType), p, false)
}

// FormatExchangeCurrencies returns a pair.CurrencyItem string containing
// the exchanges formatted currency pairs
func (e *Base) FormatExchangeCurrencies(pairs []pair.CurrencyPair, assetType assets.AssetType) (pair.CurrencyItem, error) {
	var currencyItems pair.CurrencyItem
	pairFmt := e.GetPairFormat(assetType, true)

	for x := range pairs {
		currencyItems += e.FormatExchangeCurrency(pairs[x], assetType)
		if x == len(pairs)-1 {
			continue
		}
		currencyItems += pair.CurrencyItem(pairFmt.Separator)
	}

	if len(currencyItems.String()) == 0 {
		return "", errors.New("returned empty string")
	}
	return currencyItems, nil
}

// FormatExchangeCurrency is a method that formats and returns a currency pair
// based on the user currency display preferences
func (e *Base) FormatExchangeCurrency(p pair.CurrencyPair, assetType assets.AssetType) pair.CurrencyItem {
	pairFmt := e.GetPairFormat(assetType, true)
	return p.Display(pairFmt.Delimiter, pairFmt.Uppercase)
}

// SetEnabled is a method that sets if the exchange is enabled
func (e *Base) SetEnabled(enabled bool) {
	e.Enabled = enabled
}

// IsEnabled is a method that returns if the current exchange is enabled
func (e *Base) IsEnabled() bool {
	return e.Enabled
}

// SetAPIKeys is a method that sets the current API keys for the exchange
func (e *Base) SetAPIKeys(APIKey, APISecret, ClientID string) {
	e.API.Credentials.Key = APIKey
	e.API.Credentials.ClientID = ClientID

	if e.API.CredentialsValidator.RequiresBase64DecodeSecret {
		result, err := common.Base64Decode(APISecret)
		if err != nil {
			e.API.AuthenticatedSupport = false
			log.Printf(warningBase64DecryptSecretKeyFailed, e.Name)
		}
		e.API.Credentials.Secret = string(result)
	} else {
		e.API.Credentials.Secret = APISecret
	}
}

// SetupDefaults sets the exchange settings based on the supplied config
func (e *Base) SetupDefaults(exch *config.ExchangeConfig) error {
	e.Enabled = true
	e.LoadedByConfig = true
	e.Config = exch
	e.Verbose = exch.Verbose

	e.API.AuthenticatedSupport = exch.API.AuthenticatedSupport
	if e.API.AuthenticatedSupport {
		e.SetAPIKeys(exch.API.Credentials.Key, exch.API.Credentials.Secret, e.API.Credentials.ClientID)
	}

	if exch.HTTPTimeout <= time.Duration(0) {
		exch.HTTPTimeout = DefaultHTTPTimeout
	} else {
		e.SetHTTPClientTimeout(exch.HTTPTimeout)
	}

	e.SetHTTPClientUserAgent(exch.HTTPUserAgent)
	e.SetHTTPRateLimiter()
	e.SetCurrencyPairFormat()
	e.SetAssetTypes()
	e.SetFeatureDefaults()
	e.SetAPIURL()
	e.SetAPICredentialDefaults()
	e.SetClientProxyAddress(exch.ProxyAddress)
	e.SetHTTPRateLimiter()

	e.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
	if e.Features.Supports.Trading.Spot {
		e.CurrencyPairs.Spot.Available = common.SplitStrings(exch.CurrencyPairs.Spot.Available, ",")
		e.CurrencyPairs.Spot.Enabled = common.SplitStrings(exch.CurrencyPairs.Spot.Enabled, ",")
	}

	if e.Features.Supports.Trading.Futures {
		e.CurrencyPairs.Futures.Available = common.SplitStrings(exch.CurrencyPairs.Futures.Available, ",")
		e.CurrencyPairs.Futures.Enabled = common.SplitStrings(exch.CurrencyPairs.Futures.Enabled, ",")
	}

	if e.Features.Supports.Websocket {
		e.Websocket.SetEnabled(exch.Features.Enabled.Websocket)
	}
	return nil
}

// AllowAuthenticatedRequest checks to see if the required fields have been set before sending an authenticated
// API request
func (e *Base) AllowAuthenticatedRequest() bool {
	if (!e.API.AuthenticatedSupport && e.LoadedByConfig) || (!e.LoadedByConfig && !e.ValidateAPICredentials()) {
		return false
	}
	return true
}

// ValidateAPICredentials validates the exchanges API credentials
func (e *Base) ValidateAPICredentials() bool {
	if e.API.CredentialsValidator.RequiresKey {
		if e.API.Credentials.Key == "" || e.API.Credentials.Key == config.DefaultAPIKey {
			return false
		}
	}

	if e.API.CredentialsValidator.RequiresSecret {
		if e.API.Credentials.Secret == "" || e.API.Credentials.Secret == config.DefaultAPISecret {
			return false
		}
	}

	if e.API.CredentialsValidator.RequiresPEM {
		if e.API.Credentials.PEMKey == "" || common.StringContains(e.API.Credentials.PEMKey, "JUSTADUMMY") {
			return false
		}
	}

	if e.API.CredentialsValidator.RequiresClientID {
		if e.API.Credentials.ClientID == "" || e.API.Credentials.ClientID == config.DefaultAPIClientID {
			return false
		}
	}

	if e.API.CredentialsValidator.RequiresBase64DecodeSecret {
		_, err := common.Base64Decode(e.API.Credentials.Secret)
		if err != nil {
			return false
		}
	}

	return true
}

// SetPairs sets the exchange currency pairs for either enabledPairs or
// availablePairs
func (e *Base) SetPairs(pairs []pair.CurrencyPair, assetType assets.AssetType, enabled bool) error {
	if len(pairs) == 0 {
		return fmt.Errorf("%s SetPairs error - pairs is empty", e.Name)
	}

	pairFmt := e.GetPairFormat(assetType, false)
	var pairsStr []string
	for x := range pairs {
		pairsStr = append(pairsStr, pairs[x].Display(pairFmt.Delimiter,
			pairFmt.Uppercase).String())
	}

	switch assetType {
	case assets.AssetTypeSpot:
		if enabled {
			e.Config.CurrencyPairs.Spot.Enabled = common.JoinStrings(pairsStr, ",")
			e.CurrencyPairs.Spot.Enabled = pairsStr
		} else {
			e.Config.CurrencyPairs.Spot.Available = common.JoinStrings(pairsStr, ",")
			e.CurrencyPairs.Spot.Available = pairsStr
		}
	case assets.AssetTypeFutures:
		if enabled {
			e.Config.CurrencyPairs.Futures.Enabled = common.JoinStrings(pairsStr, ",")
			e.CurrencyPairs.Futures.Enabled = pairsStr
		} else {
			e.Config.CurrencyPairs.Futures.Available = common.JoinStrings(pairsStr, ",")
			e.CurrencyPairs.Futures.Available = pairsStr
		}
	}
	return nil
}

// UpdatePairs updates the exchange currency pairs for either enabledPairs or
// availablePairs
func (e *Base) UpdatePairs(exchangeProducts []string, assetType assets.AssetType, enabled, force bool) error {
	if len(exchangeProducts) == 0 {
		return fmt.Errorf("%s UpdatePairs error - exchangeProducts is empty", e.Name)
	}

	exchangeProducts = common.SplitStrings(common.StringToUpper(common.JoinStrings(exchangeProducts, ",")), ",")
	var products []string

	for x := range exchangeProducts {
		if exchangeProducts[x] == "" {
			continue
		}
		products = append(products, exchangeProducts[x])
	}

	var newPairs, removedPairs []string
	var updateType string
	var targetPairs []string

	switch assetType {
	case assets.AssetTypeSpot:
		if enabled {
			targetPairs = e.CurrencyPairs.Spot.Enabled
		} else {
			targetPairs = e.CurrencyPairs.Spot.Available
		}
	case assets.AssetTypeFutures:
		if enabled {
			targetPairs = e.CurrencyPairs.Futures.Enabled
		} else {
			targetPairs = e.CurrencyPairs.Futures.Available
		}
	}

	if enabled {
		newPairs, removedPairs = pair.FindPairDifferences(targetPairs, products)
		updateType = "enabled"
	} else {
		newPairs, removedPairs = pair.FindPairDifferences(targetPairs, products)
		updateType = "available"
	}

	if force || len(newPairs) > 0 || len(removedPairs) > 0 {
		if force {
			log.Printf("%s forced update of %s pairs.", e.Name, updateType)
		} else {
			if len(newPairs) > 0 {
				log.Printf("%s Updating pairs - New: %s.\n", e.Name, newPairs)
			}
			if len(removedPairs) > 0 {
				log.Printf("%s Updating pairs - Removed: %s.\n", e.Name, removedPairs)
			}
		}

		switch assetType {
		case assets.AssetTypeSpot:
			if enabled {
				e.Config.CurrencyPairs.Spot.Enabled = common.JoinStrings(products, ",")
				e.CurrencyPairs.Spot.Enabled = products
			} else {
				e.Config.CurrencyPairs.Spot.Available = common.JoinStrings(products, ",")
				e.CurrencyPairs.Spot.Available = products
			}
		case assets.AssetTypeFutures:
			if enabled {
				e.Config.CurrencyPairs.Futures.Enabled = common.JoinStrings(products, ",")
				e.CurrencyPairs.Futures.Enabled = products
			} else {
				e.Config.CurrencyPairs.Futures.Available = common.JoinStrings(products, ",")
				e.CurrencyPairs.Futures.Available = products
			}
		}
	}
	return nil
}

// SetAPIURL sets configuration API URL for an exchange
func (e *Base) SetAPIURL() error {
	if e.Config.API.Endpoints.URL == "" || e.Config.API.Endpoints.URLSecondary == "" {
		return fmt.Errorf("exchange %s: SetAPIURL error. URL vals are empty", e.Name)
	}
	if e.Config.API.Endpoints.URL != config.APIURLNonDefaultMessage {
		e.API.Endpoints.URL = e.Config.API.Endpoints.URL
	}
	if e.Config.API.Endpoints.URLSecondary != config.APIURLNonDefaultMessage {
		e.API.Endpoints.URLSecondary = e.Config.API.Endpoints.URLSecondary
	}
	return nil
}

// GetAPIURL returns the set API URL
func (e *Base) GetAPIURL() string {
	return e.API.Endpoints.URL
}

// GetSecondaryAPIURL returns the set Secondary API URL
func (e *Base) GetSecondaryAPIURL() string {
	return e.API.Endpoints.URLSecondary
}

// GetAPIURLDefault returns exchange default URL
func (e *Base) GetAPIURLDefault() string {
	return e.API.Endpoints.URLDefault
}

// GetAPIURLSecondaryDefault returns exchange default secondary URL
func (e *Base) GetAPIURLSecondaryDefault() string {
	return e.API.Endpoints.URLSecondaryDefault
}

// SupportsWebsocket returns whether or not the exchange supports
// websocket
func (e *Base) SupportsWebsocket() bool {
	return e.Features.Supports.Websocket
}

// SupportsREST returns whether or not the exchange supports
// REST
func (e *Base) SupportsREST() bool {
	return e.Features.Supports.REST
}

// IsWebsocketEnabled returns whether or not the exchange has its
// websocket client enabled
func (e *Base) IsWebsocketEnabled() bool {
	if e.Websocket != nil {
		return e.Websocket.IsEnabled()
	}
	return false
}

// GetWithdrawPermissions passes through the exchange's withdraw permissions
func (e *Base) GetWithdrawPermissions() uint32 {
	return e.APIWithdrawPermissions
}

// SupportsWithdrawPermissions compares the supplied permissions with the exchange's to verify they're supported
func (e *Base) SupportsWithdrawPermissions(permissions uint32) bool {
	exchangePermissions := e.GetWithdrawPermissions()
	if permissions&exchangePermissions == permissions {
		return true
	}
	return false
}

// FormatWithdrawPermissions will return each of the exchange's compatible withdrawal methods in readable form
func (e *Base) FormatWithdrawPermissions() string {
	services := []string{}
	for i := 0; i < 32; i++ {
		var check uint32 = 1 << uint32(i)
		if e.GetWithdrawPermissions()&check != 0 {
			switch check {
			case AutoWithdrawCrypto:
				services = append(services, AutoWithdrawCryptoText)
			case AutoWithdrawCryptoWithAPIPermission:
				services = append(services, AutoWithdrawCryptoWithAPIPermissionText)
			case AutoWithdrawCryptoWithSetup:
				services = append(services, AutoWithdrawCryptoWithSetupText)
			case WithdrawCryptoWith2FA:
				services = append(services, WithdrawCryptoWith2FAText)
			case WithdrawCryptoWithSMS:
				services = append(services, WithdrawCryptoWithSMSText)
			case WithdrawCryptoWithEmail:
				services = append(services, WithdrawCryptoWithEmailText)
			case WithdrawCryptoWithWebsiteApproval:
				services = append(services, WithdrawCryptoWithWebsiteApprovalText)
			case WithdrawCryptoWithAPIPermission:
				services = append(services, WithdrawCryptoWithAPIPermissionText)
			case AutoWithdrawFiat:
				services = append(services, AutoWithdrawFiatText)
			case AutoWithdrawFiatWithAPIPermission:
				services = append(services, AutoWithdrawFiatWithAPIPermissionText)
			case AutoWithdrawFiatWithSetup:
				services = append(services, AutoWithdrawFiatWithSetupText)
			case WithdrawFiatWith2FA:
				services = append(services, WithdrawFiatWith2FAText)
			case WithdrawFiatWithSMS:
				services = append(services, WithdrawFiatWithSMSText)
			case WithdrawFiatWithEmail:
				services = append(services, WithdrawFiatWithEmailText)
			case WithdrawFiatWithWebsiteApproval:
				services = append(services, WithdrawFiatWithWebsiteApprovalText)
			case WithdrawFiatWithAPIPermission:
				services = append(services, WithdrawFiatWithAPIPermissionText)
			case WithdrawCryptoViaWebsiteOnly:
				services = append(services, WithdrawCryptoViaWebsiteOnlyText)
			case WithdrawFiatViaWebsiteOnly:
				services = append(services, WithdrawFiatViaWebsiteOnlyText)
			default:
				services = append(services, fmt.Sprintf("%s[1<<%v]", UnknownWithdrawalTypeText, i))
			}
		}
	}
	if len(services) > 0 {
		return strings.Join(services, " & ")
	}

	return NoAPIWithdrawalMethodsText
}

// IsAssetTypeSupported whether or not the supplied asset is supported
// by the exchange
func (e *Base) IsAssetTypeSupported(asset assets.AssetType) bool {
	return e.CurrencyPairs.AssetTypes.Contains(asset)
}
