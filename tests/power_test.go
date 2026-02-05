package tests

import (
	"math"
	"testing"

	"github.com/iamgilwell/aura/internal/monitor"
	"github.com/iamgilwell/aura/internal/power"
)

func TestProcessPower(t *testing.T) {
	calc := power.NewCalculator(0.5, 0.001, 0.02)

	proc := &monitor.ProcessInfo{
		PID:      1000,
		Name:     "test",
		CPU:      50.0,
		MemoryMB: 500.0,
		IORead:   0,
		IOWrite:  0,
	}

	watts := calc.ProcessPower(proc)
	if watts <= 0 {
		t.Errorf("expected positive power, got %.2f", watts)
	}

	// 50% CPU * 0.5 W/% = 25W, 500MB * 0.001 W/MB = 0.5W
	expectedMin := 25.0
	if watts < expectedMin {
		t.Errorf("expected at least %.1fW, got %.2fW", expectedMin, watts)
	}
}

func TestPowerMetrics(t *testing.T) {
	pm := power.NewMetrics()

	if pm.TotalSaved() != 0 {
		t.Error("initial total saved should be 0")
	}

	pm.RecordSaving("test1", 100, 5.0, "test reason")
	pm.RecordSaving("test2", 200, 3.0, "test reason 2")

	if pm.TotalSaved() != 8.0 {
		t.Errorf("expected 8.0W total saved, got %.1f", pm.TotalSaved())
	}

	if pm.Count() != 2 {
		t.Errorf("expected 2 events, got %d", pm.Count())
	}

	recent := pm.RecentSavings(5)
	if len(recent) != 2 {
		t.Errorf("expected 2 recent entries, got %d", len(recent))
	}
}

func TestMonthlykWh(t *testing.T) {
	// 1 watt for a month = 1 * 24 * 30 / 1000 = 0.72 kWh
	kwh := power.MonthlykWh(1.0)
	expected := 0.72
	if math.Abs(kwh-expected) > 0.01 {
		t.Errorf("MonthlykWh(1.0) = %.4f, want %.2f", kwh, expected)
	}
}

func TestMonthlyCost(t *testing.T) {
	// 100W at $0.12/kWh
	cost := power.MonthlyCost(100.0, 0.12)
	expected := power.MonthlykWh(100.0) * 0.12
	if math.Abs(cost-expected) > 0.01 {
		t.Errorf("MonthlyCost = %.4f, want %.4f", cost, expected)
	}
}
