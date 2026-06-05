package quant

type CandlePattern struct {
	Name       string  `json:"name"`
	Direction  string  `json:"direction"`
	Strength   float64 `json:"strength"`
	Confidence float64 `json:"confidence"`
}

type CandleContext struct {
	AfterDowntrend bool `json:"after_downtrend"`
	AfterUptrend   bool `json:"after_uptrend"`
	VolumeConfirm  bool `json:"volume_confirm"`
}

type CandleTrendSignal struct {
	TrendDirection string        `json:"trend_direction"`
	TrendIndex     float64       `json:"trend_index"`
	Pattern        CandlePattern `json:"pattern"`
	Context        CandleContext `json:"context"`
}
