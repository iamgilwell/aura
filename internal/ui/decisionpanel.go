package ui

import (
	"fmt"

	"github.com/rivo/tview"

	"github.com/iamgilwell/aura/internal/ai"
	"github.com/iamgilwell/aura/internal/notification"
)

// DecisionPanel displays recent AI decisions.
type DecisionPanel struct {
	app  *App
	view *tview.TextView
}

// NewDecisionPanel creates the decision panel.
func NewDecisionPanel(app *App) *DecisionPanel {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			app.tapp.Draw()
		})

	tv.SetBorder(true).
		SetTitle(" AI Decisions ").
		SetBorderPadding(0, 0, 1, 1)

	return &DecisionPanel{app: app, view: tv}
}

// AddDecision appends a decision to the panel.
func (dp *DecisionPanel) AddDecision(d *ai.DecisionResponse) {
	var color string
	switch d.Action {
	case ai.ActionTerminate:
		color = "[red]"
	case ai.ActionNotify:
		color = "[yellow]"
	case ai.ActionKeep:
		color = "[green]"
	default:
		color = "[blue]"
	}

	ts := notification.FormatTimestamp(d.Timestamp)
	cached := ""
	if d.FromCache {
		cached = " [gray](cached)"
	}

	line := fmt.Sprintf("[white]%s %s%-10s[white] PID=%-7d %-20s conf=%.2f risk=%.2f save=%.1fW %s%s\n",
		ts, color, string(d.Action),
		d.ProcessPID, d.ProcessName,
		d.Confidence, d.RiskScore, d.SavingsWatt,
		d.Reason, cached)

	fmt.Fprint(dp.view, line)
	dp.view.ScrollToEnd()
}

// ShowHistory displays all past decisions.
func (dp *DecisionPanel) ShowHistory() {
	if dp.app.aiEngine == nil {
		dp.view.SetText("[yellow]AI engine is not configured. Set ANTHROPIC_API_KEY to enable.")
		return
	}

	history := dp.app.aiEngine.DecisionHistory()
	if len(history) == 0 {
		dp.view.SetText("[gray]No AI decisions yet.")
		return
	}

	dp.view.Clear()
	for _, d := range history {
		dp.AddDecision(d)
	}
}

// ShowPowerMetrics displays power savings information.
func (dp *DecisionPanel) ShowPowerMetrics() {
	pm := dp.app.powerMetrics
	dp.view.Clear()

	text := fmt.Sprintf(
		"[yellow]Power Metrics[white]\n"+
			"─────────────────────────────────────\n"+
			"Total Saved:        [green]%.2f W[white]\n"+
			"Session Duration:   %s\n"+
			"Monthly Projection: [green]%.2f kWh[white]\n"+
			"Events Recorded:    %d\n\n",
		pm.TotalSaved(),
		pm.SessionDuration().Truncate(1e9),
		pm.MonthlyProjection(),
		pm.Count(),
	)

	recent := pm.RecentSavings(5)
	if len(recent) > 0 {
		text += "[yellow]Recent Savings:[white]\n"
		for _, e := range recent {
			text += fmt.Sprintf("  %s  %-20s PID=%-7d [green]+%.2fW[white]  %s\n",
				notification.FormatTimestamp(e.Timestamp),
				e.ProcessName, e.PID, e.WattsSaved, e.Reason)
		}
	}

	dp.view.SetText(text)
}
