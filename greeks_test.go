package tiqs

import (
	"fmt"
	"testing"
	"time"
)

func TestCallDelta(t *testing.T) {
	// Example usage
	loc, _ := time.LoadLocation("Asia/Kolkata")
	today := time.Date(2024, 10, 10, 0, 0, 0, 0, loc)
	currentExpiryDate := time.Date(2024, 10, 17, 0, 0, 0, 0, loc) // Set expiry date to tomorrow for example
	timeInDays := GetTimeInDays(currentExpiryDate, today)

	strikePrice := 25050.0
	ceLtp := 168.0
	peLtp := 178.45
	syntheticPrice := strikePrice + ceLtp - peLtp

	b76 := Black76{InterestRate: 0.0}
	iv := 0.0
	if ceLtp != 0 {
		iv = b76.ImpliedVolaUsingBisection(CALL, syntheticPrice, strikePrice, timeInDays/365, ceLtp)
	}

	values := b76.GetGreeks(CALL, syntheticPrice, strikePrice, timeInDays/365, iv)

	ceDelta := values.Delta
	gamma := values.Gamma
	vega := values.Vega
	theta := values.Theta
	peDelta := ceDelta - 1

	if fmt.Sprintf("%.2f", ceDelta) != "0.49" {
		t.Errorf("ceDelta failed")
	}
	if fmt.Sprintf("%.2f", peDelta) != "-0.51" {
		t.Errorf("peDelta failed")
	}
	if fmt.Sprintf("%.2f", gamma) != "0.00" {
		t.Errorf("gamma failed")
	}
	if fmt.Sprintf("%.2f", vega) != "67.76" {
		t.Errorf("vega failed")
	}
	if fmt.Sprintf("%.2f", theta) != "-0.52" {
		t.Errorf("theta failed")
	}
}
