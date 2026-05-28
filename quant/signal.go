package quant

import (
	"nofx/kernel"
	"nofx/market"
	"nofx/provider/nofxos"
)

func BuildSignal(symbol string, data *market.Data, qd *kernel.QuantData, oiRanking *nofxos.OIRankingData, netflowRanking *nofxos.NetFlowRankingData, priceRanking *nofxos.PriceRankingData) *Signal {
	regime := DetectRegime(data)
	trendLong, trendShort := trendScores(data)
	momLong, momShort := momentumScores(data)
	flowLong, flowShort := flowScores(data)
	if qd != nil {
		if qd.Netflow != nil && qd.Netflow.Institution != nil {
			var netLong, netShort float64
			for _, v := range qd.Netflow.Institution.Future {
				if v > 0 {
					netLong += v
				} else if v < 0 {
					netShort += -v
				}
			}
			if netLong > netShort && netLong > 0 {
				flowLong = clamp01(flowLong + 0.15)
			}
			if netShort > netLong && netShort > 0 {
				flowShort = clamp01(flowShort + 0.15)
			}
		}
		if qd.OI != nil {
			for _, oi := range qd.OI {
				if oi == nil || oi.Delta == nil {
					continue
				}
				if d, ok := oi.Delta["1h"]; ok && d != nil {
					if d.OIDeltaPercent > 0 {
						flowLong = clamp01(flowLong + 0.1)
					} else if d.OIDeltaPercent < 0 {
						flowShort = clamp01(flowShort + 0.1)
					}
					break
				}
			}
		}
	}
	if oiRanking != nil {
		for _, pos := range oiRanking.TopPositions {
			if pos.Symbol == symbol {
				flowLong = clamp01(flowLong + 0.10)
				break
			}
		}
		for _, pos := range oiRanking.LowPositions {
			if pos.Symbol == symbol {
				flowShort = clamp01(flowShort + 0.10)
				break
			}
		}
	}
	if netflowRanking != nil {
		for _, pos := range netflowRanking.InstitutionFutureTop {
			if pos.Symbol == symbol {
				flowLong = clamp01(flowLong + 0.10)
				break
			}
		}
		for _, pos := range netflowRanking.InstitutionFutureLow {
			if pos.Symbol == symbol {
				flowShort = clamp01(flowShort + 0.10)
				break
			}
		}
	}
	if priceRanking != nil {
		for _, dur := range []string{"1h", "4h", "24h"} {
			d, ok := priceRanking.Durations[dur]
			if !ok || d == nil {
				continue
			}
			for _, item := range d.Top {
				if item.Symbol == symbol {
					flowLong = clamp01(flowLong + 0.05)
					break
				}
			}
			for _, item := range d.Low {
				if item.Symbol == symbol {
					flowShort = clamp01(flowShort + 0.05)
					break
				}
			}
		}
	}
	risk := riskPenalty(data, regime)

	longScore := clamp01(0.35*trendLong + 0.30*momLong + 0.25*flowLong - 0.20*risk)
	shortScore := clamp01(0.35*trendShort + 0.30*momShort + 0.25*flowShort - 0.20*risk)
	delta := abs(longScore - shortScore)
	confidence := clamp01(0.7*max(longScore, shortScore) + 0.3*delta)
	noTrade := true
	actionBias := "wait"
	if longScore >= 0.72 && longScore-shortScore > 0.15 {
		noTrade = false
		actionBias = "open_long"
	}
	if shortScore >= 0.72 && shortScore-longScore > 0.15 {
		noTrade = false
		actionBias = "open_short"
	}

	budget := RiskBudget{MaxPositionPct: 0.10, MaxLeverage: 3, AllowNewPosition: !noTrade}
	if regime == market.RegimeLevelVolatile {
		budget.MaxPositionPct = 0.05
		budget.MaxLeverage = 2
	}

	return &Signal{
		Symbol:     symbol,
		Regime:     regime,
		LongScore:  longScore,
		ShortScore: shortScore,
		Confidence: confidence,
		ActionBias: actionBias,
		NoTrade:    noTrade,
		FactorBreakdown: FactorBreakdown{
			TrendLongScore:     trendLong,
			TrendShortScore:    trendShort,
			MomentumLongScore:  momLong,
			MomentumShortScore: momShort,
			FlowLongScore:      flowLong,
			FlowShortScore:     flowShort,
			RiskPenalty:        risk,
		},
		RiskBudget: budget,
		ExecutionHints: ExecutionHints{
			EntryStyle:        "pullback_preferred",
			StopLossHintPct:   0.018,
			TakeProfitHintPct: 0.042,
		},
	}
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
