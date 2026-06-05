package quant

import (
	"nofx/market"
	"nofx/store"
)

func BuildCandleTrendSignal(klines []market.Kline, longer *market.Data, indicators store.IndicatorConfig) *CandleTrendSignal {
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
	cw := indicators.CandleTrend
	engulfingWeight := cw.EngulfingWeight
	hammerWeight := cw.HammerWeight
	starWeight := cw.StarWeight
	contextWeight := cw.ContextWeight
	minTrendIndex := cw.MinTrendIndex
	if engulfingWeight == 0 {
		engulfingWeight = 1.0
	}
	if hammerWeight == 0 {
		hammerWeight = 0.9
	}
	if starWeight == 0 {
		starWeight = 1.1
	}
	if contextWeight == 0 {
		contextWeight = 0.4
	}
	if minTrendIndex == 0 {
		minTrendIndex = 0.45
	}
	patternWeightBase := 1 - contextWeight
	setBull := func(name string, strength float64, patternWeight float64) {
		conf := clamp01(patternWeightBase*(strength*patternWeight) + 0.6*contextWeight*boolScore(ctx.AfterDowntrend) + 0.4*contextWeight*boolScore(ctx.VolumeConfirm))
		if conf > best.TrendIndex {
			best = CandleTrendSignal{TrendDirection: "bullish_reversal", TrendIndex: conf, Context: ctx, Pattern: CandlePattern{Name: name, Direction: "bullish", Strength: strength, Confidence: conf}}
		}
	}
	setBear := func(name string, strength float64, patternWeight float64) {
		conf := clamp01(patternWeightBase*(strength*patternWeight) + 0.6*contextWeight*boolScore(ctx.AfterUptrend) + 0.4*contextWeight*boolScore(ctx.VolumeConfirm))
		if conf > best.TrendIndex {
			best = CandleTrendSignal{TrendDirection: "bearish_reversal", TrendIndex: conf, Context: ctx, Pattern: CandlePattern{Name: name, Direction: "bearish", Strength: strength, Confidence: conf}}
		}
	}

	if s, ok := detectBullishEngulfing(klines); ok {
		setBull("bullish_engulfing", s, engulfingWeight)
	}
	if s, ok := detectBearishEngulfing(klines); ok {
		setBear("bearish_engulfing", s, engulfingWeight)
	}
	if s, ok := detectHammer(klines); ok {
		setBull("hammer", s, hammerWeight)
	}
	if s, ok := detectShootingStar(klines); ok {
		setBear("shooting_star", s, hammerWeight)
	}
	if s, ok := detectMorningStar(klines); ok {
		setBull("morning_star", s, starWeight)
	}
	if s, ok := detectEveningStar(klines); ok {
		setBear("evening_star", s, starWeight)
	}
	if best.TrendIndex < minTrendIndex {
		best.TrendDirection = "neutral"
		best.Pattern = CandlePattern{}
	}
	return &best
}

func boolScore(v bool) float64 {
	if v {
		return 1
	}
	return 0
}
