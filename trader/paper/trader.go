package paper

import (
	"fmt"
	"math"
	"nofx/market"
	"nofx/trader/types"
	"strings"
	"sync"
	"time"
)

const (
	defaultInitialBalance = 10000.0
	defaultTakerFeeRate   = 0.0004 // 0.04%
)

// PriceProvider returns the current market price for a symbol.
type PriceProvider func(symbol string) (float64, error)

// Option customizes a PaperTrader.
type Option func(*PaperTrader)

// WithPriceProvider injects a price provider, mainly for tests.
func WithPriceProvider(provider PriceProvider) Option {
	return func(t *PaperTrader) {
		if provider != nil {
			t.priceProvider = provider
		}
	}
}

// WithTakerFeeRate sets the taker fee rate. Use 0 for no fees in tests.
func WithTakerFeeRate(rate float64) Option {
	return func(t *PaperTrader) {
		if rate >= 0 {
			t.takerFeeRate = rate
		}
	}
}

type paperPosition struct {
	Symbol     string
	Side       string // long or short
	Quantity   float64
	EntryPrice float64
	Leverage   int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type paperOrder struct {
	OrderID      string
	Symbol       string
	Side         string
	PositionSide string
	Type         string
	Price        float64
	StopPrice    float64
	Quantity     float64
	Status       string
	AvgPrice     float64
	ExecutedQty  float64
	Commission   float64
	CreatedAt    time.Time
}

// PaperTrader is a local paper-trading exchange implementation.
// It deliberately imports no real exchange trading package, so it cannot place live orders.
type PaperTrader struct {
	mu sync.RWMutex

	initialBalance float64
	walletBalance  float64 // realized wallet balance, excluding unrealized PnL
	takerFeeRate   float64
	isCrossMargin  bool
	priceProvider  PriceProvider

	positions map[string]*paperPosition
	orders    map[string]*paperOrder
	closedPnL []types.ClosedPnLRecord
	nextSeq   int64
}

// NewPaperTrader creates a local paper trader with virtual initial balance.
func NewPaperTrader(initialBalance float64, opts ...Option) *PaperTrader {
	if initialBalance <= 0 {
		initialBalance = defaultInitialBalance
	}
	t := &PaperTrader{
		initialBalance: initialBalance,
		walletBalance:  initialBalance,
		takerFeeRate:   defaultTakerFeeRate,
		isCrossMargin:  true,
		positions:      make(map[string]*paperPosition),
		orders:         make(map[string]*paperOrder),
	}
	t.priceProvider = t.defaultPriceProvider
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (t *PaperTrader) defaultPriceProvider(symbol string) (float64, error) {
	data, err := market.Get(symbol)
	if err != nil {
		return 0, err
	}
	if data == nil || data.CurrentPrice <= 0 {
		return 0, fmt.Errorf("market price not available for %s", symbol)
	}
	return data.CurrentPrice, nil
}

func normalizeSymbol(symbol string) string {
	return strings.ToUpper(strings.TrimSpace(symbol))
}

func normalizeSide(side string) string {
	side = strings.ToLower(strings.TrimSpace(side))
	if side == "short" {
		return "short"
	}
	return "long"
}

func positionKey(symbol, side string) string {
	return normalizeSymbol(symbol) + "|" + normalizeSide(side)
}

func (t *PaperTrader) nextOrderIDLocked() string {
	t.nextSeq++
	return fmt.Sprintf("paper-%d-%d", time.Now().UnixNano(), t.nextSeq)
}

func (t *PaperTrader) currentPrice(symbol string) (float64, error) {
	price, err := t.priceProvider(normalizeSymbol(symbol))
	if err != nil {
		return 0, err
	}
	if price <= 0 || math.IsNaN(price) || math.IsInf(price, 0) {
		return 0, fmt.Errorf("invalid market price for %s: %v", symbol, price)
	}
	return price, nil
}

func positionUnrealized(pos *paperPosition, markPrice float64) float64 {
	if pos == nil || pos.Quantity == 0 {
		return 0
	}
	if pos.Side == "short" {
		return (pos.EntryPrice - markPrice) * pos.Quantity
	}
	return (markPrice - pos.EntryPrice) * pos.Quantity
}

func (t *PaperTrader) marginUsedLocked() float64 {
	total := 0.0
	for _, pos := range t.positions {
		price, err := t.currentPrice(pos.Symbol)
		if err != nil || price <= 0 {
			price = pos.EntryPrice
		}
		lev := pos.Leverage
		if lev <= 0 {
			lev = 1
		}
		total += pos.Quantity * price / float64(lev)
	}
	return total
}

func (t *PaperTrader) GetBalance() (map[string]interface{}, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	unrealized := 0.0
	for _, pos := range t.positions {
		price, err := t.currentPrice(pos.Symbol)
		if err != nil {
			return nil, err
		}
		unrealized += positionUnrealized(pos, price)
	}
	marginUsed := t.marginUsedLocked()
	available := t.walletBalance - marginUsed
	if available < 0 {
		available = 0
	}
	totalEquity := t.walletBalance + unrealized

	return map[string]interface{}{
		"totalWalletBalance":    t.walletBalance,
		"availableBalance":      available,
		"available_balance":     available,
		"totalUnrealizedProfit": unrealized,
		"totalEquity":           totalEquity,
		"total_equity":          totalEquity,
		"wallet_balance":        t.walletBalance,
		"balance":               t.walletBalance,
		"initial_balance":       t.initialBalance,
	}, nil
}

func (t *PaperTrader) GetPositions() ([]map[string]interface{}, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	positions := make([]map[string]interface{}, 0, len(t.positions))
	for _, pos := range t.positions {
		markPrice, err := t.currentPrice(pos.Symbol)
		if err != nil {
			return nil, err
		}
		amt := pos.Quantity
		if pos.Side == "short" {
			amt = -amt
		}
		positions = append(positions, map[string]interface{}{
			"symbol":           pos.Symbol,
			"side":             pos.Side,
			"positionAmt":      amt,
			"entryPrice":       pos.EntryPrice,
			"markPrice":        markPrice,
			"unRealizedProfit": positionUnrealized(pos, markPrice),
			"liquidationPrice": 0.0,
			"leverage":         float64(pos.Leverage),
			"createdTime":      pos.CreatedAt.UnixMilli(),
			"updateTime":       pos.UpdatedAt.UnixMilli(),
		})
	}
	return positions, nil
}

func (t *PaperTrader) openPosition(symbol, side string, quantity float64, leverage int) (map[string]interface{}, error) {
	if quantity <= 0 {
		return nil, fmt.Errorf("quantity must be greater than 0")
	}
	if leverage <= 0 {
		leverage = 1
	}
	symbol = normalizeSymbol(symbol)
	side = normalizeSide(side)
	price, err := t.currentPrice(symbol)
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	notional := quantity * price
	fee := notional * t.takerFeeRate
	margin := notional / float64(leverage)
	if t.walletBalance-t.marginUsedLocked() < margin+fee {
		return nil, fmt.Errorf("insufficient paper balance: need %.4f, available %.4f", margin+fee, t.walletBalance-t.marginUsedLocked())
	}
	t.walletBalance -= fee

	key := positionKey(symbol, side)
	now := time.Now()
	if pos := t.positions[key]; pos != nil {
		totalQty := pos.Quantity + quantity
		pos.EntryPrice = (pos.EntryPrice*pos.Quantity + price*quantity) / totalQty
		pos.Quantity = totalQty
		pos.Leverage = leverage
		pos.UpdatedAt = now
	} else {
		t.positions[key] = &paperPosition{Symbol: symbol, Side: side, Quantity: quantity, EntryPrice: price, Leverage: leverage, CreatedAt: now, UpdatedAt: now}
	}

	orderID := t.nextOrderIDLocked()
	t.orders[orderID] = &paperOrder{OrderID: orderID, Symbol: symbol, Side: mapOrderSide(side, true), PositionSide: strings.ToUpper(side), Type: "MARKET", Quantity: quantity, Status: "FILLED", AvgPrice: price, ExecutedQty: quantity, Commission: fee, CreatedAt: now}
	return map[string]interface{}{"orderId": orderID, "symbol": symbol, "status": "FILLED", "avgPrice": price, "executedQty": quantity, "commission": fee}, nil
}

func mapOrderSide(positionSide string, opening bool) string {
	positionSide = normalizeSide(positionSide)
	if positionSide == "long" {
		if opening {
			return "BUY"
		}
		return "SELL"
	}
	if opening {
		return "SELL"
	}
	return "BUY"
}

func (t *PaperTrader) OpenLong(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	return t.openPosition(symbol, "long", quantity, leverage)
}

func (t *PaperTrader) OpenShort(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	return t.openPosition(symbol, "short", quantity, leverage)
}

func (t *PaperTrader) closePosition(symbol, side string, quantity float64) (map[string]interface{}, error) {
	symbol = normalizeSymbol(symbol)
	side = normalizeSide(side)
	price, err := t.currentPrice(symbol)
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	key := positionKey(symbol, side)
	pos := t.positions[key]
	if pos == nil || pos.Quantity <= 0 {
		return nil, fmt.Errorf("no %s position found for %s", side, symbol)
	}
	if quantity <= 0 || quantity > pos.Quantity {
		quantity = pos.Quantity
	}

	closedRatio := quantity / pos.Quantity
	realized := positionUnrealized(&paperPosition{Symbol: symbol, Side: side, Quantity: quantity, EntryPrice: pos.EntryPrice, Leverage: pos.Leverage}, price)
	fee := quantity * price * t.takerFeeRate
	t.walletBalance += realized - fee

	now := time.Now()
	orderID := t.nextOrderIDLocked()
	t.orders[orderID] = &paperOrder{OrderID: orderID, Symbol: symbol, Side: mapOrderSide(side, false), PositionSide: strings.ToUpper(side), Type: "MARKET", Quantity: quantity, Status: "FILLED", AvgPrice: price, ExecutedQty: quantity, Commission: fee, CreatedAt: now}
	t.closedPnL = append(t.closedPnL, types.ClosedPnLRecord{Symbol: symbol, Side: side, EntryPrice: pos.EntryPrice, ExitPrice: price, Quantity: quantity, RealizedPnL: realized, Fee: fee, Leverage: pos.Leverage, EntryTime: pos.CreatedAt, ExitTime: now, OrderID: orderID, CloseType: "paper_market", ExchangeID: orderID})

	pos.Quantity -= quantity
	pos.UpdatedAt = now
	if pos.Quantity <= 1e-12 || closedRatio >= 1 {
		delete(t.positions, key)
	}
	_ = closedRatio

	return map[string]interface{}{"orderId": orderID, "symbol": symbol, "status": "FILLED", "avgPrice": price, "executedQty": quantity, "realizedPnl": realized, "commission": fee}, nil
}

func (t *PaperTrader) CloseLong(symbol string, quantity float64) (map[string]interface{}, error) {
	return t.closePosition(symbol, "long", quantity)
}

func (t *PaperTrader) CloseShort(symbol string, quantity float64) (map[string]interface{}, error) {
	return t.closePosition(symbol, "short", quantity)
}

func (t *PaperTrader) SetLeverage(symbol string, leverage int) error {
	if leverage <= 0 {
		return fmt.Errorf("leverage must be greater than 0")
	}
	symbol = normalizeSymbol(symbol)
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, side := range []string{"long", "short"} {
		if pos := t.positions[positionKey(symbol, side)]; pos != nil {
			pos.Leverage = leverage
			pos.UpdatedAt = time.Now()
		}
	}
	return nil
}

func (t *PaperTrader) SetMarginMode(symbol string, isCrossMargin bool) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.isCrossMargin = isCrossMargin
	return nil
}

func (t *PaperTrader) GetMarketPrice(symbol string) (float64, error) {
	return t.currentPrice(symbol)
}

func (t *PaperTrader) recordConditionalOrder(symbol, positionSide, orderType string, quantity, triggerPrice float64) error {
	if quantity < 0 || triggerPrice <= 0 {
		return fmt.Errorf("invalid paper conditional order")
	}
	symbol = normalizeSymbol(symbol)
	positionSide = strings.ToUpper(normalizeSide(positionSide))
	t.mu.Lock()
	defer t.mu.Unlock()
	orderID := t.nextOrderIDLocked()
	side := "SELL"
	if positionSide == "SHORT" {
		side = "BUY"
	}
	t.orders[orderID] = &paperOrder{OrderID: orderID, Symbol: symbol, Side: side, PositionSide: positionSide, Type: orderType, StopPrice: triggerPrice, Quantity: quantity, Status: "NEW", CreatedAt: time.Now()}
	return nil
}

func (t *PaperTrader) SetStopLoss(symbol string, positionSide string, quantity, stopPrice float64) error {
	return t.recordConditionalOrder(symbol, positionSide, "STOP_MARKET", quantity, stopPrice)
}

func (t *PaperTrader) SetTakeProfit(symbol string, positionSide string, quantity, takeProfitPrice float64) error {
	return t.recordConditionalOrder(symbol, positionSide, "TAKE_PROFIT_MARKET", quantity, takeProfitPrice)
}

func (t *PaperTrader) cancelOrders(symbol string, match func(*paperOrder) bool) error {
	symbol = normalizeSymbol(symbol)
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, order := range t.orders {
		if order.Symbol == symbol && order.Status == "NEW" && match(order) {
			order.Status = "CANCELED"
		}
	}
	return nil
}

func (t *PaperTrader) CancelStopLossOrders(symbol string) error {
	return t.cancelOrders(symbol, func(o *paperOrder) bool {
		return strings.Contains(o.Type, "STOP") && !strings.Contains(o.Type, "TAKE_PROFIT")
	})
}

func (t *PaperTrader) CancelTakeProfitOrders(symbol string) error {
	return t.cancelOrders(symbol, func(o *paperOrder) bool { return strings.Contains(o.Type, "TAKE_PROFIT") })
}

func (t *PaperTrader) CancelAllOrders(symbol string) error {
	return t.cancelOrders(symbol, func(o *paperOrder) bool { return true })
}

func (t *PaperTrader) CancelStopOrders(symbol string) error {
	return t.cancelOrders(symbol, func(o *paperOrder) bool {
		return strings.Contains(o.Type, "STOP") || strings.Contains(o.Type, "TAKE_PROFIT")
	})
}

func (t *PaperTrader) FormatQuantity(symbol string, quantity float64) (string, error) {
	if quantity <= 0 {
		return "0", fmt.Errorf("quantity must be greater than 0")
	}
	return fmt.Sprintf("%.8f", quantity), nil
}

func (t *PaperTrader) GetOrderStatus(symbol string, orderID string) (map[string]interface{}, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	order := t.orders[orderID]
	if order == nil || (symbol != "" && order.Symbol != normalizeSymbol(symbol)) {
		return nil, fmt.Errorf("paper order not found: %s", orderID)
	}
	return map[string]interface{}{"orderId": order.OrderID, "status": order.Status, "avgPrice": order.AvgPrice, "executedQty": order.ExecutedQty, "commission": order.Commission}, nil
}

func (t *PaperTrader) GetClosedPnL(startTime time.Time, limit int) ([]types.ClosedPnLRecord, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if limit <= 0 {
		limit = 100
	}
	records := make([]types.ClosedPnLRecord, 0, limit)
	for _, record := range t.closedPnL {
		if !record.ExitTime.Before(startTime) {
			records = append(records, record)
			if len(records) >= limit {
				break
			}
		}
	}
	return records, nil
}

func (t *PaperTrader) GetOpenOrders(symbol string) ([]types.OpenOrder, error) {
	symbol = normalizeSymbol(symbol)
	t.mu.RLock()
	defer t.mu.RUnlock()
	orders := make([]types.OpenOrder, 0)
	for _, order := range t.orders {
		if order.Status != "NEW" {
			continue
		}
		if symbol != "" && order.Symbol != symbol {
			continue
		}
		orders = append(orders, types.OpenOrder{OrderID: order.OrderID, Symbol: order.Symbol, Side: order.Side, PositionSide: order.PositionSide, Type: order.Type, Price: order.Price, StopPrice: order.StopPrice, Quantity: order.Quantity, Status: order.Status})
	}
	return orders, nil
}
