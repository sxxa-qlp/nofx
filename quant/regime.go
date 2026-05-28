package quant

import "nofx/market"

func DetectRegime(data *market.Data) market.RegimeLevel {
	if data == nil {
		return market.RegimeLevelStandard
	}
	atr := 0.0
	if data.IntradaySeries != nil {
		atr = data.IntradaySeries.ATR14
	}
	if data.CurrentPrice > 0 && atr/data.CurrentPrice > 0.03 {
		return market.RegimeLevelVolatile
	}
	if data.LongerTermContext != nil {
		if data.LongerTermContext.EMA20 > 0 && data.LongerTermContext.EMA50 > 0 {
			if data.LongerTermContext.EMA20 > data.LongerTermContext.EMA50 {
				return market.RegimeLevelTrending
			}
			if data.LongerTermContext.EMA20 < data.LongerTermContext.EMA50 {
				return market.RegimeLevelWide
			}
		}
	}
	return market.RegimeLevelStandard
}
