package power

import (
	"github.com/iamgilwell/aura/internal/monitor"
)

// Calculator estimates power consumption and savings.
type Calculator struct {
	cpuWattPerPercent float64
	memoryWattPerMB   float64
	diskWattPerMBps   float64
}

// NewCalculator creates a power calculator with the given coefficients.
func NewCalculator(cpuWattPerPercent, memoryWattPerMB, diskWattPerMBps float64) *Calculator {
	return &Calculator{
		cpuWattPerPercent: cpuWattPerPercent,
		memoryWattPerMB:   memoryWattPerMB,
		diskWattPerMBps:   diskWattPerMBps,
	}
}

// ProcessPower estimates the power consumption of a process in watts.
func (c *Calculator) ProcessPower(proc *monitor.ProcessInfo) float64 {
	cpuWatts := proc.CPU * c.cpuWattPerPercent
	memWatts := proc.MemoryMB * c.memoryWattPerMB

	// Estimate IO in MB/s (rough approximation based on total bytes)
	ioMBps := float64(proc.IORead+proc.IOWrite) / (1024 * 1024) / 60.0 // rough per-second estimate
	diskWatts := ioMBps * c.diskWattPerMBps

	return cpuWatts + memWatts + diskWatts
}

// EstimateSavings returns the estimated power savings in watts if a process is terminated.
func (c *Calculator) EstimateSavings(proc *monitor.ProcessInfo) float64 {
	return c.ProcessPower(proc)
}

// MonthlykWh converts watts to monthly kWh.
func MonthlykWh(watts float64) float64 {
	hoursPerMonth := 24.0 * 30.0
	return watts * hoursPerMonth / 1000.0
}

// MonthlyCost estimates monthly cost in USD at a given rate per kWh.
func MonthlyCost(watts float64, ratePerKWh float64) float64 {
	return MonthlykWh(watts) * ratePerKWh
}
