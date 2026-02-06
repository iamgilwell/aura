package ui

import (
	"fmt"
	"time"

	"github.com/rivo/tview"

	"github.com/iamgilwell/aura/internal/monitor"
	"github.com/iamgilwell/aura/internal/safety"
)

// Dashboard is the top status bar.
type Dashboard struct {
	app  *App
	view *tview.TextView
}

// NewDashboard creates the dashboard widget.
func NewDashboard(app *App) *Dashboard {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	tv.SetBorder(true).
		SetTitle(" Aura - AI-Powered Process Optimizer ").
		SetBorderPadding(0, 0, 1, 1)

	return &Dashboard{app: app, view: tv}
}

// Update refreshes the dashboard display.
func (d *Dashboard) Update(metrics *monitor.SystemMetrics) {
	if metrics == nil {
		return
	}

	runtime := time.Since(d.app.startTime).Truncate(time.Second)

	mode := "Interactive"
	if d.app.safetyMgr.IsMonitorOnly() {
		mode = "Monitor Only"
	}

	consentLevel := d.app.safetyMgr.ConsentLevel()
	consentDesc := safety.LevelDescription(consentLevel)

	aiStatus := "[red]OFF"
	if d.app.aiEngine != nil {
		aiStatus = "[green]ON"
	}

	powerSaved := d.app.powerMetrics.TotalSaved()

	text := fmt.Sprintf(
		" [yellow]Runtime:[white] %s | [yellow]Mode:[white] %s | [yellow]Safety:[white] %s (L%d) | [yellow]AI:[white] %s[white] | "+
			"[yellow]Procs:[white] %d | [yellow]CPU:[white] %.1f%% | [yellow]Mem:[white] %.1f%% | [yellow]Load:[white] %.2f | "+
			"[yellow]Power Saved:[white] %.2fW",
		runtime, mode, consentDesc, consentLevel, aiStatus,
		metrics.NumProcs, metrics.TotalCPU, metrics.TotalMemory, metrics.LoadAvg1,
		powerSaved,
	)

	d.view.SetText(text)
}
