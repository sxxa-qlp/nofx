# Quant Signal & Candlestick Trend Signal

This document explains the current NOFX signal stack for backtesting and future live execution:

1. **Quantitative formula layer**
2. **Candlestick trend layer**
3. **How both are intended to work together**
4. **Their inputs and outputs**

---

## 1. Overview

The current signal pipeline is designed as:

```text
exchange market data
→ market feature extraction
→ quant signal
→ candlestick trend signal
→ AI execution decision
→ simulated/live execution
```

### Design goal

- The **quant layer** provides the primary directional bias.
- The **candlestick layer** provides reversal-pattern confirmation.
- The **AI layer** reads both and decides the final action.

---

## 2. Quantitative Formula Layer

### 2.1 Purpose

The quant layer converts exchange-native data and technical indicators into a structured directional score.

It answers:
- Is the market biased long or short?
- Is the signal strong enough to trade?
- What basic risk budget should apply?

### 2.2 Current inputs

The quant layer currently uses exchange-data-first inputs:

- Current price
- EMA structure
- MACD
- RSI
- ATR / volatility
- PriceChange1h
- PriceChange4h
- Open Interest (when available)
- Funding rate (when available)
- Optional extended quant data (currently disabled in exchange-only mode)

### 2.3 Current factor model

The quant signal is composed of four factor families:

#### A. Trend factor
Measures directional alignment.

Typical checks:
- current price vs current EMA20
- longer timeframe EMA20 vs EMA50
- current price vs longer timeframe EMA20
- longer timeframe price change direction

Outputs:
- `trend_long_score`
- `trend_short_score`

#### B. Momentum factor
Measures directional push.

Typical checks:
- MACD sign
- RSI location
- 1h price change confirming MACD
- 4h price change confirming RSI bias

Outputs:
- `momentum_long_score`
- `momentum_short_score`

#### C. Flow factor
Measures supporting price/participation flow.

In exchange-only mode, it is intentionally weaker and mainly uses:
- 1h price change
- 4h price change
- basic open-interest ratio (latest vs average)

Outputs:
- `flow_long_score`
- `flow_short_score`

#### D. Risk penalty
Reduces confidence under unstable conditions.

Typical checks:
- high volatility regime
- ATR percentage too high
- extreme funding rate

Output:
- `risk_penalty`

---

### 2.4 Current configurable weights

These are now stored in strategy config under:

```json
ai_config.indicators.quant_scoring
```

Fields:

- `trend_weight`
- `momentum_weight`
- `flow_weight`
- `risk_weight`
- `entry_threshold`
- `delta_threshold`

### 2.5 Current scoring formula

Current implementation logic:

```text
long_score  = trend_weight * trend_long_score
            + momentum_weight * momentum_long_score
            + flow_weight * flow_long_score
            - risk_weight * risk_penalty

short_score = trend_weight * trend_short_score
            + momentum_weight * momentum_short_score
            + flow_weight * flow_short_score
            - risk_weight * risk_penalty
```

Then:

```text
delta = abs(long_score - short_score)
confidence = 0.60 * max(long_score, short_score) + 0.40 * delta
```

Entry bias logic:

```text
if long_score >= entry_threshold
   and (long_score - short_score) > delta_threshold:
   action_bias = open_long

if short_score >= entry_threshold
   and (short_score - long_score) > delta_threshold:
   action_bias = open_short

otherwise:
   action_bias = wait
   no_trade = true
```

### 2.6 Quant output

Output object:

```json
{
  "symbol": "BTCUSDT",
  "regime": "standard",
  "long_score": 0.52,
  "short_score": 0.18,
  "confidence": 0.57,
  "action_bias": "open_long",
  "no_trade": false,
  "factor_breakdown": {
    "trend_long_score": 0.80,
    "trend_short_score": 0.00,
    "momentum_long_score": 0.60,
    "momentum_short_score": 0.00,
    "flow_long_score": 0.55,
    "flow_short_score": 0.45,
    "risk_penalty": 0.10
  },
  "risk_budget": {
    "max_position_pct": 0.10,
    "max_leverage": 3,
    "allow_new_position": true
  },
  "execution_hints": {
    "entry_style": "pullback_preferred",
    "stop_loss_hint_pct": 0.018,
    "take_profit_hint_pct": 0.042
  }
}
```

---

## 3. Candlestick Trend Layer

### 3.1 Purpose

The candlestick trend layer is a dedicated reversal-pattern detector.

It answers:
- Is there a strong candlestick reversal signal right now?
- If yes, is it bullish or bearish?
- How strong is that reversal in current context?

### 3.2 Pattern families used

Current implementation targets three classic reversal groups:

#### A. Engulfing
- `bullish_engulfing`
- `bearish_engulfing`

#### B. Hammer family
- `hammer`
- `shooting_star`

#### C. Star family
- `morning_star`
- `evening_star`

### 3.3 Pattern intent

#### Engulfing
Detects strong two-candle reversal takeover.

#### Hammer / Shooting Star
Detects rejection tails.

#### Morning / Evening Star
Detects three-candle reversal transition.

---

### 3.4 Context checks

Pattern shape alone is not enough.
Current context checks include:

- `after_downtrend`
- `after_uptrend`
- `volume_confirm`

These improve confidence by checking whether the pattern appears in the right directional setting.

### 3.5 Current configurable weights

These are now stored in strategy config under:

```json
ai_config.indicators.candle_trend
```

Fields:

- `enabled`
- `engulfing_weight`
- `hammer_weight`
- `star_weight`
- `context_weight`
- `min_trend_index`

### 3.6 Current scoring method

The candlestick layer computes:

```text
pattern_weight_base = 1 - context_weight
trend_index = pattern_weight_base * (pattern_strength * pattern_weight)
            + context_weight * context_confirmation
```

Where context confirmation is split into:
- trend location contribution
- volume confirmation contribution

If the resulting `trend_index` is below `min_trend_index`, the signal is neutralized.

### 3.7 Candlestick output

Output object:

```json
{
  "trend_direction": "bullish_reversal",
  "trend_index": 0.71,
  "pattern": {
    "name": "bullish_engulfing",
    "direction": "bullish",
    "strength": 0.82,
    "confidence": 0.71
  },
  "context": {
    "after_downtrend": true,
    "after_uptrend": false,
    "volume_confirm": true
  }
}
```

If no pattern is strong enough:

```json
{
  "trend_direction": "neutral",
  "trend_index": 0,
  "pattern": {},
  "context": {...}
}
```

---

## 4. Multi-timeframe impact

### 4.1 Quant layer

- **15m**: local momentum / entry environment
- **1h**: medium-term directional confirmation
- **4h**: larger directional structure filter

### 4.2 Candlestick layer

Current implementation primarily evaluates the selected replay window and uses longer context to determine:
- whether the pattern appears after an uptrend/downtrend
- whether it should count as a meaningful reversal

In practical terms:
- 15m gives the local candle pattern
- 1h / 4h help determine whether the pattern is contextually important

---

## 5. How Quant + Candlestick work together

### Current intended relationship

- **Quant = primary bias layer**
- **Candlestick = reversal confirmation layer**
- **AI = final execution layer**

### Meaning

Quant answers:
- Is there enough directional structure to even consider a trade?

Candlestick answers:
- Is there a recognizable reversal pattern that confirms a turning point?

AI answers:
- Given quant + candlestick + account state, what action should we take now?

---

## 6. Backtest integration

Current backtest event flow is intended to evolve toward:

```text
cycle_start
→ quant_signal
→ candlestick_trend_signal
→ decision
→ simulated_execution
```

This lets the UI and debugging flow show:
- what data was used
- what quant thought
- what candlestick trend thought
- what AI finally decided

---

## 7. Summary

### Quant layer
- turns exchange + indicator data into directional scores
- decides whether there is a tradeable bias

### Candlestick layer
- detects reversal patterns
- outputs a trend index and direction

### AI layer
- reads both
- performs final execution reasoning

### Configurable parts
Now configurable in strategy config:
- quant scoring weights and thresholds
- candlestick pattern weights and thresholds

This allows future tuning without changing Go code every time.
