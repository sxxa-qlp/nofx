package quant

import "nofx/market"

func trendScores(data *market.Data) (float64, float64) {
	if data == nil || data.CurrentPrice <= 0 {
		return 0, 0
	}
	longScore := 0.0
	shortScore := 0.0

	if data.CurrentEMA20 > 0 {
		if data.CurrentPrice >= data.CurrentEMA20 {
			longScore += 0.45
		} else {
			shortScore += 0.45
		}
	}
	if data.LongerTermContext != nil && data.LongerTermContext.EMA20 > 0 && data.LongerTermContext.EMA50 > 0 {
		if data.LongerTermContext.EMA20 >= data.LongerTermContext.EMA50 {
			longScore += 0.55
		} else {
			shortScore += 0.55
		}
	}
	return clamp01(longScore), clamp01(shortScore)
}

func momentumScores(data *market.Data) (float64, float64) {
	if data == nil {
		return 0, 0
	}
	longScore := 0.0
	shortScore := 0.0
	if data.CurrentMACD > 0 {
		longScore += 0.5
	} else if data.CurrentMACD < 0 {
		shortScore += 0.5
	}
	if data.CurrentRSI7 >= 52 && data.CurrentRSI7 <= 72 {
		longScore += 0.5
	}
	if data.CurrentRSI7 <= 48 && data.CurrentRSI7 >= 28 {
		shortScore += 0.5
	}
	return clamp01(longScore), clamp01(shortScore)
}

func flowScores(data *market.Data) (float64, float64) {
	if data == nil {
		return 0.5, 0.5
	}
	longScore := 0.5
	shortScore := 0.5
	if data.PriceChange1h > 0 {
		longScore += 0.15
		shortScore -= 0.15
	} else if data.PriceChange1h < 0 {
		shortScore += 0.15
		longScore -= 0.15
	}
	if data.PriceChange4h > 0 {
		longScore += 0.15
		shortScore -= 0.15
	} else if data.PriceChange4h < 0 {
		shortScore += 0.15
		longScore -= 0.15
	}
	if data.OpenInterest != nil {
		if data.OpenInterest.Latest > data.OpenInterest.Average && data.OpenInterest.Average > 0 {
			longScore += 0.1
		} else if data.OpenInterest.Latest < data.OpenInterest.Average && data.OpenInterest.Average > 0 {
			shortScore += 0.1
		}
	}
	return clamp01(longScore), clamp01(shortScore)
}

func riskPenalty(data *market.Data, regime market.RegimeLevel) float64 {
	if data == nil || data.CurrentPrice <= 0 {
		return 1
	}
	penalty := 0.0
	if regime == market.RegimeLevelVolatile {
		penalty += 0.35
	}
	if data.IntradaySeries != nil && data.IntradaySeries.ATR14 > 0 {
		atrPct := data.IntradaySeries.ATR14 / data.CurrentPrice
		if atrPct > 0.02 {
			penalty += 0.25
		}
	}
	if data.FundingRate > 0.0008 || data.FundingRate < -0.0008 {
		penalty += 0.15
	}
	return clamp01(penalty)
}
