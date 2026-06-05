package quant

import "nofx/market"

func detectBullishEngulfing(klines []market.Kline) (float64, bool) {
	if len(klines) < 2 {
		return 0, false
	}
	a := klines[len(klines)-2]
	b := klines[len(klines)-1]
	if a.Close >= a.Open || b.Close <= b.Open {
		return 0, false
	}
	if b.Open > a.Close || b.Close < a.Open {
		return 0, false
	}
	bodyA := abs(a.Close - a.Open)
	bodyB := abs(b.Close - b.Open)
	if bodyA <= 0 || bodyB <= 0 {
		return 0, false
	}
	strength := clamp01((bodyB / bodyA) * 0.7)
	if strength < 0.45 {
		return 0, false
	}
	return strength, true
}

func detectBearishEngulfing(klines []market.Kline) (float64, bool) {
	if len(klines) < 2 {
		return 0, false
	}
	a := klines[len(klines)-2]
	b := klines[len(klines)-1]
	if a.Close <= a.Open || b.Close >= b.Open {
		return 0, false
	}
	if b.Open < a.Close || b.Close > a.Open {
		return 0, false
	}
	bodyA := abs(a.Close - a.Open)
	bodyB := abs(b.Close - b.Open)
	if bodyA <= 0 || bodyB <= 0 {
		return 0, false
	}
	strength := clamp01((bodyB / bodyA) * 0.7)
	if strength < 0.45 {
		return 0, false
	}
	return strength, true
}

func detectHammer(klines []market.Kline) (float64, bool) {
	if len(klines) < 1 {
		return 0, false
	}
	k := klines[len(klines)-1]
	body := abs(k.Close - k.Open)
	if body <= 0 {
		body = 0.0001
	}
	lower := minf(k.Open, k.Close) - k.Low
	upper := k.High - maxf(k.Open, k.Close)
	if lower >= body*2.2 && upper <= body*0.8 {
		return clamp01(lower / (body * 3)), true
	}
	return 0, false
}

func detectShootingStar(klines []market.Kline) (float64, bool) {
	if len(klines) < 1 {
		return 0, false
	}
	k := klines[len(klines)-1]
	body := abs(k.Close - k.Open)
	if body <= 0 {
		body = 0.0001
	}
	upper := k.High - maxf(k.Open, k.Close)
	lower := minf(k.Open, k.Close) - k.Low
	if upper >= body*2.2 && lower <= body*0.8 {
		return clamp01(upper / (body * 3)), true
	}
	return 0, false
}

func detectMorningStar(klines []market.Kline) (float64, bool) {
	if len(klines) < 3 {
		return 0, false
	}
	a, b, c := klines[len(klines)-3], klines[len(klines)-2], klines[len(klines)-1]
	if a.Close >= a.Open || c.Close <= c.Open {
		return 0, false
	}
	bodyA := abs(a.Close - a.Open)
	bodyB := abs(b.Close - b.Open)
	bodyC := abs(c.Close - c.Open)
	if bodyA <= 0 || bodyC <= 0 {
		return 0, false
	}
	if bodyB > bodyA*0.6 {
		return 0, false
	}
	if c.Close < (a.Open+a.Close)/2 {
		return 0, false
	}
	return clamp01((bodyC/bodyA)*0.6 + 0.3), true
}

func detectEveningStar(klines []market.Kline) (float64, bool) {
	if len(klines) < 3 {
		return 0, false
	}
	a, b, c := klines[len(klines)-3], klines[len(klines)-2], klines[len(klines)-1]
	if a.Close <= a.Open || c.Close >= c.Open {
		return 0, false
	}
	bodyA := abs(a.Close - a.Open)
	bodyB := abs(b.Close - b.Open)
	bodyC := abs(c.Close - c.Open)
	if bodyA <= 0 || bodyC <= 0 {
		return 0, false
	}
	if bodyB > bodyA*0.6 {
		return 0, false
	}
	if c.Close > (a.Open+a.Close)/2 {
		return 0, false
	}
	return clamp01((bodyC/bodyA)*0.6 + 0.3), true
}

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
