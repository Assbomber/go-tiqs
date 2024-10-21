package tiqs

import (
	"math"
	"time"
)

const (
	MS_PER_MINUTE        = 1000 * 60
	TRADING_START_HOUR   = 9
	TRADING_START_MINUTE = 15
	TRADING_END_HOUR     = 15
	TRADING_END_MINUTE   = 30
)

var FULL_TRADING_DAY_MS = float64((TRADING_END_HOUR*60+TRADING_END_MINUTE)-(TRADING_START_HOUR*60+TRADING_START_MINUTE)) * MS_PER_MINUTE

func GetTimeInDays(currentExpiryDate, today time.Time) float64 {
	tradingStartTimeToday := time.Date(today.Year(), today.Month(), today.Day(), TRADING_START_HOUR, TRADING_START_MINUTE, 0, 0, today.Location())
	tradingEndTimeToday := time.Date(today.Year(), today.Month(), today.Day(), TRADING_END_HOUR, TRADING_END_MINUTE, 0, 0, today.Location())

	if today.Format("2006-01-02") == currentExpiryDate.Format("2006-01-02") {
		now := time.Now()
		if now.Before(tradingStartTimeToday) {
			return 1
		}
		if now.After(tradingEndTimeToday) {
			return 0.0000001
		}
		remainingTradingTimeMs := float64(tradingEndTimeToday.Sub(now).Milliseconds())
		return remainingTradingTimeMs / FULL_TRADING_DAY_MS
	}

	timeDifferenceMs := currentExpiryDate.Sub(today).Hours() * 24
	timeDifferenceDays := timeDifferenceMs / 24

	if today.Before(currentExpiryDate) {
		now := time.Now()
		if now.Before(tradingEndTimeToday) || now.Equal(tradingEndTimeToday) {
			return math.Ceil(timeDifferenceDays + 1)
		}
	}

	return math.Ceil(timeDifferenceDays)
}

type NormalDistribution struct{}

func (nd NormalDistribution) cdf(x float64) float64 {
	t := 1 / (1 + 0.2316419*math.Abs(x))
	a1 := 0.319381530
	a2 := -0.356563782
	a3 := 1.781477937
	a4 := -1.821255978
	a5 := 1.330274429
	result := 1 - (1/math.Sqrt(2*math.Pi))*math.Exp(-(x*x)/2)*(a1*t+a2*math.Pow(t, 2)+a3*math.Pow(t, 3)+a4*math.Pow(t, 4)+a5*math.Pow(t, 5))
	if x >= 0 {
		return result
	}
	return 1 - result
}

func (nd NormalDistribution) pdf(x float64) float64 {
	return math.Exp(-(x*x)/2) / math.Sqrt(2*math.Pi)
}

type Black76 struct {
	InterestRate float64
}

const (
	CALL                  = "C"
	PUT                   = "P"
	METHOD_BISECTION      = 1
	METHOD_NEWTON_RAPHSON = 2
)

func (b Black76) d1d2(underlyingPrice, strikePrice, timeToMaturity, volatility float64) (float64, float64) {
	d1 := (math.Log(underlyingPrice/strikePrice) + (math.Pow(volatility, 2)/2)*timeToMaturity) / (volatility * math.Sqrt(timeToMaturity))
	d2 := d1 - volatility*math.Sqrt(timeToMaturity)
	return d1, d2
}

type Greeks struct {
	Value float64
	Delta float64
	Gamma float64
	Vega  float64
	Theta float64
	Rho   float64
}

func (b Black76) GetGreeks(optionType string, underlyingPrice, strikePrice, timeToMaturity, volatility float64) Greeks {
	discountFactor := math.Exp(-b.InterestRate * timeToMaturity)
	d1, d2 := b.d1d2(underlyingPrice, strikePrice, timeToMaturity, volatility)
	sign := map[string]float64{"C": 1, "P": -1}[optionType]
	nd := NormalDistribution{}
	nd1 := nd.cdf(d1 * sign)
	nd2 := nd.cdf(d2 * sign)
	normpdf := nd.pdf(d1)
	sqrtTimeToMaturity := math.Sqrt(timeToMaturity)
	value := sign * discountFactor * (underlyingPrice*nd1 - strikePrice*nd2)
	delta := sign * discountFactor * nd1
	gamma := discountFactor * (normpdf / (volatility * underlyingPrice * sqrtTimeToMaturity))
	vega := 0.01 * underlyingPrice * discountFactor * normpdf * sqrtTimeToMaturity
	theta := (-underlyingPrice * discountFactor * normpdf * (volatility / (2 * sqrtTimeToMaturity))) +
		sign*b.InterestRate*discountFactor*(underlyingPrice*nd1-strikePrice*nd2)
	rho := -0.01 * timeToMaturity * value

	return Greeks{
		Value: value,
		Delta: delta,
		Gamma: gamma,
		Vega:  vega,
		Theta: theta / 365,
		Rho:   rho,
	}
}

func (b Black76) ImpliedVolaUsingBisection(optionType string, underlyingPrice, strikePrice, timeToMaturity, marketPrice float64) float64 {
	epsilon := 0.0001
	maxIterations := 100
	volMin := 0.00001
	volMax := 5.0
	volGuess := volMin

	for i := 0; i < maxIterations; i++ {
		valueMin := b.GetGreeks(optionType, underlyingPrice, strikePrice, timeToMaturity, volMin).Value
		valueMax := b.GetGreeks(optionType, underlyingPrice, strikePrice, timeToMaturity, volMax).Value

		if volMax-volMin <= epsilon && valueMin == valueMax {
			break
		}

		volBisection := volMin + (volMax-volMin)*((marketPrice-valueMin)/(valueMax-valueMin))
		volGuess = math.Max(volMin, math.Min(volBisection, volMax))
		valueGuess := b.GetGreeks(optionType, underlyingPrice, strikePrice, timeToMaturity, volGuess).Value

		if valueGuess < marketPrice {
			volMin = volGuess
		} else {
			volMax = volGuess
		}
	}

	return volGuess
}
