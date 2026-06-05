package quant

import "nofx/market"

func BuildCandleTrendSignal(klines []market.Kline, longer *market.Data) *CandleTrendSignal {
	if len(klines) == 0 {
		return &CandleTrendSignal{TrendDirection: "neutral", TrendIndex: 0}
	}
	ctx := CandleContext{}
	if longer != nil {
		ctx.AfterDowntrend = longer.PriceChange4h < 0
		ctx.AfterUptrend = longer.PriceChange4h > 0
	}
	if len(klines) >= 2 {
		avgVol := 0.0
		for _, k := range klines[:len(klines)-1] {
			avgVol += k.Volume
		}
		avgVol /= float64(len(klines) - 1)
		ctx.VolumeConfirm = avgVol > 0 && klines[len(klines)-1].Volume > avgVol*1.1
	}

	best := CandleTrendSignal{TrendDirection: "neutral", TrendIndex: 0, Context: ctx}
	setBull := func(name string, strength float64) {
		conf := clamp01(0.6*strength + 0.25*boolScore(ctx.AfterDowntrend) + 0.15*boolScore(ctx.VolumeConfirm))
		if conf > best.TrendIndex {
			best = CandleTrendSignal{TrendDirection: "bullish_reversal", TrendIndex: conf, Context: ctx, Pattern: CandlePattern{Name: name, Direction: "bullish", Strength: strength, Confidence: conf}}
		}
	}
	setBear := func(name string, strength float64) {
		conf := clamp01(0.6*strength + 0.25*boolScore(ctx.AfterUptrend) + 0.15*boolScore(ctx.VolumeConfirm))
		if conf > best.TrendIndex {
			best = CandleTrendSignal{TrendDirection: "bearish_reversal", TrendIndex: conf, Context: ctx, Pattern: CandlePattern{Name: name, Direction: "bearish", Strength: strength, Confidence: conf}}
		}
	}

	if s, ok := detectBullishEngulfing(klines); ok {
		setBull("bullish_engulfing", s)
	}
	if s, ok := detectBearishEngulfing(klines); ok {
		setBear("bearish_engulfing", s)
	}
	if s, ok := detectHammer(klines); ok {
		setBull("hammer", s)
	}
	if s, ok := detectShootingStar(klines); ok {
		setBear("shooting_star", s)
	}
	if s, ok := detectMorningStar(klines); ok {
		setBull("morning_star", s)
	}
	if s, ok := detectEveningStar(klines); ok {
		setBear("evening_star", s)
	}
	return &best
}

func boolScore(v bool) float64 {
	if v {
		return 1
	}
	return 0
}
