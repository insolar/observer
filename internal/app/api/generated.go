// Package api provides primitives to interact the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen DO NOT EDIT.
package api

import (
	"fmt"
	"net/http"

	"github.com/deepmap/oapi-codegen/pkg/runtime"
	"github.com/labstack/echo/v4"
)

// ResponsesAddressCountYaml defines model for responses-addressCount-yaml.
type ResponsesAddressCountYaml struct {
	Count int `json:"count"`
}

// ResponsesAddressesYaml defines model for responses-addresses-yaml.
type ResponsesAddressesYaml []string

// ResponsesCoinsYaml defines model for responses-coins-yaml.
type ResponsesCoinsYaml struct {
	CirculatingSupply string `json:"circulatingSupply"`
	MaxSupply         string `json:"maxSupply"`
	TotalSupply       string `json:"totalSupply"`
}

// ResponsesDetailsYaml defines model for responses-details-yaml.
type ResponsesDetailsYaml struct {
	CostCenter SchemaCostCenter `json:"costCenter"`
	FeeMember  SchemaFeeMember  `json:"feeMember"`
	From       SchemaFromRefs   `json:"from"`
	To         SchemaToRefs     `json:"to"`
}

// ResponsesFeeYaml defines model for responses-fee-yaml.
type ResponsesFeeYaml struct {
	Fee string `json:"fee"`
}

// ResponsesInvalidAmountYaml defines model for responses-invalidAmount-yaml.
type ResponsesInvalidAmountYaml struct {
	Error []string `json:"error"`
}

// ResponsesInvalidReferenceYaml defines model for responses-invalidReference-yaml.
type ResponsesInvalidReferenceYaml struct {
	Error []string `json:"error"`
}

// ResponsesInvalidTransactionIdYaml defines model for responses-invalidTransactionId-yaml.
type ResponsesInvalidTransactionIdYaml struct {
	Error []string `json:"error"`
}

// ResponsesMemberYaml defines model for responses-member-yaml.
type ResponsesMemberYaml struct {
	AccountReference string           `json:"accountReference"`
	Balance          string           `json:"balance"`
	Deposits         *[]SchemaDeposit `json:"deposits,omitempty"`
	MigrationAddress *string          `json:"migrationAddress,omitempty"`
	WalletReference  string           `json:"walletReference"`
}

// ResponsesMemberBalanceYaml defines model for responses-memberBalance-yaml.
type ResponsesMemberBalanceYaml struct {
	Balance string `json:"balance"`
}

// ResponsesNotificationInfoYaml defines model for responses-notificationInfo-yaml.
type ResponsesNotificationInfoYaml struct {
	Notification string `json:"notification"`
}

// ResponsesStatisticsYaml defines model for responses-statistics-yaml.
type ResponsesStatisticsYaml struct {
	Accounts              float32 `json:"accounts"`
	CurrentTPS            float32 `json:"currentTPS"`
	LastMonthTransactions float32 `json:"lastMonthTransactions"`
	MaxTPS                float32 `json:"maxTPS"`
	Nodes                 float32 `json:"nodes"`
	TotalTransactions     float32 `json:"totalTransactions"`
}

// ResponsesTokenInfoYaml defines model for responses-tokenInfo-yaml.
type ResponsesTokenInfoYaml struct {
	CurrentPrice string `json:"currentPrice"`
	DailyChange  string `json:"dailyChange"`
	MarketCap    string `json:"marketCap"`
	Rank         string `json:"rank"`
	Volume       string `json:"volume"`
}

// ResponsesWeekPricesYaml defines model for responses-weekPrices-yaml.
type ResponsesWeekPricesYaml []struct {
	Price     string `json:"price"`
	Timestamp int64  `json:"timestamp"`
}

// SchemaAcceptRefs defines model for schema-accept-refs.
type SchemaAcceptRefs struct {
	Account string `json:"account"`
	Member  string `json:"member"`
	Wallet  string `json:"wallet"`
}

// SchemaCostCenter defines model for schema-cost-center.
type SchemaCostCenter struct {
	CalcFeeRequest string `json:"calcFeeRequest"`
	Reference      string `json:"reference"`
}

// SchemaDeposit defines model for schema-deposit.
type SchemaDeposit struct {
	AmountOnHold     string             `json:"amountOnHold"`
	AvailableAmount  string             `json:"availableAmount"`
	DepositReference string             `json:"depositReference"`
	EthTxHash        string             `json:"ethTxHash"`
	HoldReleaseDate  int64              `json:"holdReleaseDate"`
	Index            float32            `json:"index"`
	MemberReference  *string            `json:"memberReference,omitempty"`
	NextRelease      *SchemaNextRelease `json:"nextRelease,omitempty"`
	ReleaseEndDate   int64              `json:"releaseEndDate"`
	ReleasedAmount   string             `json:"releasedAmount"`
	Status           string             `json:"status"`
	Timestamp        int64              `json:"timestamp"`
}

// SchemaFeeMember defines model for schema-fee-member.
type SchemaFeeMember struct {
	AcceptRequest string `json:"acceptRequest"`
	Reference     string `json:"reference"`
}

// SchemaFromRefs defines model for schema-from-refs.
type SchemaFromRefs struct {
	AccountReference string             `json:"accountReference"`
	MemberReference  string             `json:"memberReference"`
	TransferRequests SchemaTransferRefs `json:"transferRequests"`
	WalletReference  string             `json:"walletReference"`
}

// SchemaMigration defines model for schema-migration.
type SchemaMigration struct {
	// Embedded struct due to allOf(#/components/schemas/schemas-transactionAbstract)
	SchemasTransactionAbstract
	// Embedded fields due to inline allOf schema
	FromMemberReference string `json:"fromMemberReference"`
	ToDepositReference  string `json:"toDepositReference"`
	ToMemberReference   string `json:"toMemberReference"`
	Type                string `json:"type"`
}

// SchemaNextRelease defines model for schema-next-release.
type SchemaNextRelease struct {
	Amount    string `json:"amount"`
	Timestamp int64  `json:"timestamp"`
}

// SchemaRelease defines model for schema-release.
type SchemaRelease struct {
	// Embedded struct due to allOf(#/components/schemas/schemas-transactionAbstract)
	SchemasTransactionAbstract
	// Embedded fields due to inline allOf schema
	FromDepositReference string `json:"fromDepositReference"`
	ToMemberReference    string `json:"toMemberReference"`
	Type                 string `json:"type"`
}

// SchemaToRefs defines model for schema-to-refs.
type SchemaToRefs struct {
	AcceptRequests   SchemaAcceptRefs `json:"acceptRequests"`
	AccountReference string           `json:"accountReference"`
	MemberReference  string           `json:"memberReference"`
	WalletReference  string           `json:"walletReference"`
}

// SchemaTransfer defines model for schema-transfer.
type SchemaTransfer struct {
	// Embedded struct due to allOf(#/components/schemas/schemas-transactionAbstract)
	SchemasTransactionAbstract
	// Embedded fields due to inline allOf schema
	FromMemberReference string `json:"fromMemberReference"`
	ToMemberReference   string `json:"toMemberReference"`
	Type                string `json:"type"`
}

// SchemaTransferRefs defines model for schema-transfer-refs.
type SchemaTransferRefs struct {
	Account string `json:"account"`
	Member  string `json:"member"`
	Wallet  string `json:"wallet"`
}

// SchemasTransaction defines model for schemas-transaction.
type SchemasTransaction interface{}

// SchemasTransactionAbstract defines model for schemas-transactionAbstract.
type SchemasTransactionAbstract struct {
	Amount      string  `json:"amount"`
	Fee         *string `json:"fee,omitempty"`
	Index       string  `json:"index"`
	PulseNumber int64   `json:"pulseNumber"`
	Status      string  `json:"status"`
	Timestamp   int64   `json:"timestamp"`
	TxID        string  `json:"txID"`
	Type        string  `json:"type"`
}

// SchemasTransactions defines model for schemas-transactions.
type SchemasTransactions []SchemasTransaction

// InvalidTransactionListParameters defines model for invalidTransactionListParameters.
type InvalidTransactionListParameters struct {
	Error []string `json:"error"`
}

// Transactions defines model for transactions.
type Transactions struct {
	// Embedded fields due to inline allOf schema
	// Embedded struct due to allOf(#/components/schemas/schemas-transactions)
	SchemasTransactions
}

// GetMigrationAddressesParams defines parameters for GetMigrationAddresses.
type GetMigrationAddressesParams struct {

	// The `migrationAddress` to start from. To start from the first added one, do not specify.
	MigrationAddress *string `json:"migrationAddress,omitempty"`

	// Number of entries per list.
	Limit int `json:"limit"`
}

// MemberTransactionsParams defines parameters for MemberTransactions.
type MemberTransactionsParams struct {

	// Number of entries per list.
	Limit int `json:"limit"`

	// Unique index of the transaction from which to start the list. Format: `<pulse_number>:<sequence_number>`.
	//
	// Note: since path parameters must be valid parts of URL, the `:` is to be replaced with `%3A` in accordance with the HTML URL encoding.
	//
	// Each returned transaction has an `index` that can be specified as the value of this parameter. To get the list of most recent transactions, omit the index.
	Index *string `json:"index,omitempty"`

	// Transaction's direction:
	//
	//   * `incoming` - transactions only to member,
	//   * `outgoing` - transactions only from member,
	//   * `all` - both to and from.
	Direction *string `json:"direction,omitempty"`

	// Chronological `order` of the transaction list starting from a given `index`:
	//
	//   * `chronological` — get transactions that chronologically follow a transaction with a given `index`;
	//   * `reverse` — get transactions that chronologically preceed a transaction with a given `index`.
	Order *string `json:"order,omitempty"`

	// Transaction type:
	//
	// * `transfer` - transactions to/from to member,
	// * `migration` - transactions only to member's deposits,
	// * `release` - transactions only from member's deposits to member's account.
	Type *string `json:"type,omitempty"`

	// Transaction status:
	//
	// * `registered` — transfer request is registered;
	// * `sent` — transfer of funds from the sender is finalized;
	// * `received` — transfer of funds to the receiver is finalized.
	// * `failed` — transfer of funds is finalized with an error, e.g., in case of insufficient balance.
	Status *string `json:"status,omitempty"`
}

// TokenGetInfoParams defines parameters for TokenGetInfo.
type TokenGetInfoParams struct {

	// Сurrency code:
	//
	//   * `USD` - US dollar, default if the parameter is not specified.
	//   * `BTC` - Bitcoin.
	//   * `ETH` - Ethereum.
	Currency *string `json:"currency,omitempty"`
}

// TransactionsSearchParams defines parameters for TransactionsSearch.
type TransactionsSearchParams struct {

	// Value of `txID`, `fromMemberReference`, `toMemberReference` or `pulseNumber` by which to search (filter) transactions.
	//
	// Note: since path parameters must be valid parts of URL, the `:` after `insolar` in references and IDs is to be replaced with `%3A` in accordance with the HTML URL encoding.
	Value *string `json:"value,omitempty"`

	// Number of entries per list.
	Limit int `json:"limit"`

	// Unique index of the transaction from which to start the list. Format: `<pulse_number>:<sequence_number>`.
	//
	// Note: since path parameters must be valid parts of URL, the `:` is to be replaced with `%3A` in accordance with the HTML URL encoding.
	//
	// Each returned transaction has an `index` that can be specified as the value of this parameter. To get the list of most recent transactions, omit the index.
	Index *string `json:"index,omitempty"`

	// Chronological `order` of the transaction list starting from a given `index`:
	//
	//   * `chronological` — get transactions that chronologically follow a transaction with a given `index`;
	//   * `reverse` — get transactions that chronologically preceed a transaction with a given `index`.
	Order *string `json:"order,omitempty"`

	// Transaction type:
	//
	// * `transfer` - transactions to/from to member,
	// * `migration` - transactions only to member's deposits,
	// * `release` - transactions only from member's deposits to member's account.
	Type *string `json:"type,omitempty"`

	// Transaction status:
	//
	// * `registered` — transfer request is registered;
	// * `sent` — transfer of funds from the sender is finalized;
	// * `received` — transfer of funds to the receiver is finalized.
	// * `failed` — transfer of funds is finalized with an error, e.g., in case of insufficient balance.
	Status *string `json:"status,omitempty"`
}

// ClosedTransactionsParams defines parameters for ClosedTransactions.
type ClosedTransactionsParams struct {

	// Number of entries per list.
	Limit int `json:"limit"`

	// Unique index of the transaction from which to start the list. Format: `<pulse_number>:<sequence_number>`.
	//
	// Note: since path parameters must be valid parts of URL, the `:` is to be replaced with `%3A` in accordance with the HTML URL encoding.
	//
	// Each returned transaction has an `index` that can be specified as the value of this parameter. To get the list of most recent transactions, omit the index.
	Index *string `json:"index,omitempty"`

	// Chronological `order` of the transaction list starting from a given `index`:
	//
	//   * `chronological` — get transactions that chronologically follow a transaction with a given `index`;
	//   * `reverse` — get transactions that chronologically preceed a transaction with a given `index`.
	Order *string `json:"order,omitempty"`
}

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// addresses// (GET /admin/migration/addresses)
	GetMigrationAddresses(ctx echo.Context, params GetMigrationAddressesParams) error
	// addresses/count// (GET /admin/migration/addresses/count)
	GetMigrationAddressCount(ctx echo.Context) error
	// coins// (GET /api/coins)
	Coins(ctx echo.Context) error
	// coins/circulating// (GET /api/coins/circulating)
	CoinsCirculating(ctx echo.Context) error
	// coins/max// (GET /api/coins/max)
	CoinsMax(ctx echo.Context) error
	// coins/total// (GET /api/coins/total)
	CoinsTotal(ctx echo.Context) error
	// fee// (GET /api/fee/{amount})
	Fee(ctx echo.Context, amount string) error
	// member// (GET /api/member/{reference})
	Member(ctx echo.Context, reference string) error
	// member balance// (GET /api/member/{reference}/balance)
	Balance(ctx echo.Context, reference string) error
	// member transactions// (GET /api/member/{reference}/transactions)
	MemberTransactions(ctx echo.Context, reference string, params MemberTransactionsParams) error
	// notification// (GET /api/notification)
	Notification(ctx echo.Context) error
	// network statistics// (GET /api/statistics)
	GetStatistics(ctx echo.Context) error
	// token// (GET /api/token)
	TokenGetInfo(ctx echo.Context, params TokenGetInfoParams) error
	// token/weekPrices// (GET /api/token/weekPrices/{interval})
	TokenWeekPrice(ctx echo.Context, interval int) error
	// transaction// (GET /api/transaction/{txID})
	Transaction(ctx echo.Context, txID string) error
	// transaction details// (GET /api/transaction/{txID}/details)
	TransactionsDetails(ctx echo.Context, txID string) error
	// transactions// (GET /api/transactions)
	TransactionsSearch(ctx echo.Context, params TransactionsSearchParams) error
	// closed transactions// (GET /api/transactions/closed)
	ClosedTransactions(ctx echo.Context, params ClosedTransactionsParams) error
}

// ServerInterfaceWrapper converts echo contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler ServerInterface
}

// GetMigrationAddresses converts echo context to params.
func (w *ServerInterfaceWrapper) GetMigrationAddresses(ctx echo.Context) error {
	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params GetMigrationAddressesParams
	// ------------- Optional query parameter "migrationAddress" -------------
	if paramValue := ctx.QueryParam("migrationAddress"); paramValue != "" {

	}

	err = runtime.BindQueryParameter("form", true, false, "migrationAddress", ctx.QueryParams(), &params.MigrationAddress)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter migrationAddress: %s", err))
	}

	// ------------- Required query parameter "limit" -------------
	if paramValue := ctx.QueryParam("limit"); paramValue != "" {

	} else {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query argument limit is required, but not found"))
	}

	err = runtime.BindQueryParameter("form", true, true, "limit", ctx.QueryParams(), &params.Limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter limit: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetMigrationAddresses(ctx, params)
	return err
}

// GetMigrationAddressCount converts echo context to params.
func (w *ServerInterfaceWrapper) GetMigrationAddressCount(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetMigrationAddressCount(ctx)
	return err
}

// Coins converts echo context to params.
func (w *ServerInterfaceWrapper) Coins(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.Coins(ctx)
	return err
}

// CoinsCirculating converts echo context to params.
func (w *ServerInterfaceWrapper) CoinsCirculating(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.CoinsCirculating(ctx)
	return err
}

// CoinsMax converts echo context to params.
func (w *ServerInterfaceWrapper) CoinsMax(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.CoinsMax(ctx)
	return err
}

// CoinsTotal converts echo context to params.
func (w *ServerInterfaceWrapper) CoinsTotal(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.CoinsTotal(ctx)
	return err
}

// Fee converts echo context to params.
func (w *ServerInterfaceWrapper) Fee(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "amount" -------------
	var amount string

	err = runtime.BindStyledParameter("simple", false, "amount", ctx.Param("amount"), &amount)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter amount: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.Fee(ctx, amount)
	return err
}

// Member converts echo context to params.
func (w *ServerInterfaceWrapper) Member(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "reference" -------------
	var reference string

	err = runtime.BindStyledParameter("simple", false, "reference", ctx.Param("reference"), &reference)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter reference: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.Member(ctx, reference)
	return err
}

// Balance converts echo context to params.
func (w *ServerInterfaceWrapper) Balance(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "reference" -------------
	var reference string

	err = runtime.BindStyledParameter("simple", false, "reference", ctx.Param("reference"), &reference)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter reference: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.Balance(ctx, reference)
	return err
}

// MemberTransactions converts echo context to params.
func (w *ServerInterfaceWrapper) MemberTransactions(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "reference" -------------
	var reference string

	err = runtime.BindStyledParameter("simple", false, "reference", ctx.Param("reference"), &reference)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter reference: %s", err))
	}

	// Parameter object where we will unmarshal all parameters from the context
	var params MemberTransactionsParams
	// ------------- Required query parameter "limit" -------------
	if paramValue := ctx.QueryParam("limit"); paramValue != "" {

	} else {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query argument limit is required, but not found"))
	}

	err = runtime.BindQueryParameter("form", true, true, "limit", ctx.QueryParams(), &params.Limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter limit: %s", err))
	}

	// ------------- Optional query parameter "index" -------------
	if paramValue := ctx.QueryParam("index"); paramValue != "" {

	}

	err = runtime.BindQueryParameter("form", true, false, "index", ctx.QueryParams(), &params.Index)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter index: %s", err))
	}

	// ------------- Optional query parameter "direction" -------------
	if paramValue := ctx.QueryParam("direction"); paramValue != "" {

	}

	err = runtime.BindQueryParameter("form", true, false, "direction", ctx.QueryParams(), &params.Direction)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter direction: %s", err))
	}

	// ------------- Optional query parameter "order" -------------
	if paramValue := ctx.QueryParam("order"); paramValue != "" {

	}

	err = runtime.BindQueryParameter("form", true, false, "order", ctx.QueryParams(), &params.Order)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter order: %s", err))
	}

	// ------------- Optional query parameter "type" -------------
	if paramValue := ctx.QueryParam("type"); paramValue != "" {

	}

	err = runtime.BindQueryParameter("form", true, false, "type", ctx.QueryParams(), &params.Type)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter type: %s", err))
	}

	// ------------- Optional query parameter "status" -------------
	if paramValue := ctx.QueryParam("status"); paramValue != "" {

	}

	err = runtime.BindQueryParameter("form", true, false, "status", ctx.QueryParams(), &params.Status)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter status: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.MemberTransactions(ctx, reference, params)
	return err
}

// Notification converts echo context to params.
func (w *ServerInterfaceWrapper) Notification(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.Notification(ctx)
	return err
}

// GetStatistics converts echo context to params.
func (w *ServerInterfaceWrapper) GetStatistics(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetStatistics(ctx)
	return err
}

// TokenGetInfo converts echo context to params.
func (w *ServerInterfaceWrapper) TokenGetInfo(ctx echo.Context) error {
	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params TokenGetInfoParams
	// ------------- Optional query parameter "currency" -------------
	if paramValue := ctx.QueryParam("currency"); paramValue != "" {

	}

	err = runtime.BindQueryParameter("form", true, false, "currency", ctx.QueryParams(), &params.Currency)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter currency: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.TokenGetInfo(ctx, params)
	return err
}

// TokenWeekPrice converts echo context to params.
func (w *ServerInterfaceWrapper) TokenWeekPrice(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "interval" -------------
	var interval int

	err = runtime.BindStyledParameter("simple", false, "interval", ctx.Param("interval"), &interval)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter interval: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.TokenWeekPrice(ctx, interval)
	return err
}

// Transaction converts echo context to params.
func (w *ServerInterfaceWrapper) Transaction(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "txID" -------------
	var txID string

	err = runtime.BindStyledParameter("simple", false, "txID", ctx.Param("txID"), &txID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter txID: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.Transaction(ctx, txID)
	return err
}

// TransactionsDetails converts echo context to params.
func (w *ServerInterfaceWrapper) TransactionsDetails(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "txID" -------------
	var txID string

	err = runtime.BindStyledParameter("simple", false, "txID", ctx.Param("txID"), &txID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter txID: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.TransactionsDetails(ctx, txID)
	return err
}

// TransactionsSearch converts echo context to params.
func (w *ServerInterfaceWrapper) TransactionsSearch(ctx echo.Context) error {
	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params TransactionsSearchParams
	// ------------- Optional query parameter "value" -------------
	if paramValue := ctx.QueryParam("value"); paramValue != "" {

	}

	err = runtime.BindQueryParameter("form", true, false, "value", ctx.QueryParams(), &params.Value)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter value: %s", err))
	}

	// ------------- Required query parameter "limit" -------------
	if paramValue := ctx.QueryParam("limit"); paramValue != "" {

	} else {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query argument limit is required, but not found"))
	}

	err = runtime.BindQueryParameter("form", true, true, "limit", ctx.QueryParams(), &params.Limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter limit: %s", err))
	}

	// ------------- Optional query parameter "index" -------------
	if paramValue := ctx.QueryParam("index"); paramValue != "" {

	}

	err = runtime.BindQueryParameter("form", true, false, "index", ctx.QueryParams(), &params.Index)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter index: %s", err))
	}

	// ------------- Optional query parameter "order" -------------
	if paramValue := ctx.QueryParam("order"); paramValue != "" {

	}

	err = runtime.BindQueryParameter("form", true, false, "order", ctx.QueryParams(), &params.Order)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter order: %s", err))
	}

	// ------------- Optional query parameter "type" -------------
	if paramValue := ctx.QueryParam("type"); paramValue != "" {

	}

	err = runtime.BindQueryParameter("form", true, false, "type", ctx.QueryParams(), &params.Type)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter type: %s", err))
	}

	// ------------- Optional query parameter "status" -------------
	if paramValue := ctx.QueryParam("status"); paramValue != "" {

	}

	err = runtime.BindQueryParameter("form", true, false, "status", ctx.QueryParams(), &params.Status)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter status: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.TransactionsSearch(ctx, params)
	return err
}

// ClosedTransactions converts echo context to params.
func (w *ServerInterfaceWrapper) ClosedTransactions(ctx echo.Context) error {
	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params ClosedTransactionsParams
	// ------------- Required query parameter "limit" -------------
	if paramValue := ctx.QueryParam("limit"); paramValue != "" {

	} else {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query argument limit is required, but not found"))
	}

	err = runtime.BindQueryParameter("form", true, true, "limit", ctx.QueryParams(), &params.Limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter limit: %s", err))
	}

	// ------------- Optional query parameter "index" -------------
	if paramValue := ctx.QueryParam("index"); paramValue != "" {

	}

	err = runtime.BindQueryParameter("form", true, false, "index", ctx.QueryParams(), &params.Index)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter index: %s", err))
	}

	// ------------- Optional query parameter "order" -------------
	if paramValue := ctx.QueryParam("order"); paramValue != "" {

	}

	err = runtime.BindQueryParameter("form", true, false, "order", ctx.QueryParams(), &params.Order)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter order: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.ClosedTransactions(ctx, params)
	return err
}

// RegisterHandlers adds each server route to the EchoRouter.
func RegisterHandlers(router runtime.EchoRouter, si ServerInterface) {

	wrapper := ServerInterfaceWrapper{
		Handler: si,
	}

	router.GET("/admin/migration/addresses", wrapper.GetMigrationAddresses)
	router.GET("/admin/migration/addresses/count", wrapper.GetMigrationAddressCount)
	router.GET("/api/coins", wrapper.Coins)
	router.GET("/api/coins/circulating", wrapper.CoinsCirculating)
	router.GET("/api/coins/max", wrapper.CoinsMax)
	router.GET("/api/coins/total", wrapper.CoinsTotal)
	router.GET("/api/fee/:amount", wrapper.Fee)
	router.GET("/api/member/:reference", wrapper.Member)
	router.GET("/api/member/:reference/balance", wrapper.Balance)
	router.GET("/api/member/:reference/transactions", wrapper.MemberTransactions)
	router.GET("/api/notification", wrapper.Notification)
	router.GET("/api/statistics", wrapper.GetStatistics)
	router.GET("/api/token", wrapper.TokenGetInfo)
	router.GET("/api/token/weekPrices/:interval", wrapper.TokenWeekPrice)
	router.GET("/api/transaction/:txID", wrapper.Transaction)
	router.GET("/api/transaction/:txID/details", wrapper.TransactionsDetails)
	router.GET("/api/transactions", wrapper.TransactionsSearch)
	router.GET("/api/transactions/closed", wrapper.ClosedTransactions)

}
