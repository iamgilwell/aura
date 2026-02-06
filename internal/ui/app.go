package ui

import (
	"context"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/iamgilwell/aura/internal/ai"
	"github.com/iamgilwell/aura/internal/config"
	"github.com/iamgilwell/aura/internal/monitor"
	"github.com/iamgilwell/aura/internal/notification"
	"github.com/iamgilwell/aura/internal/power"
	"github.com/iamgilwell/aura/internal/process"
	"github.com/iamgilwell/aura/internal/safety"
)

// App is the main TUI application.
type App struct {
	tapp         *tview.Application
	cfg          *config.Config
	mon          *monitor.ProcessMonitor
	aiEngine     *ai.Engine
	safetyMgr    *safety.Manager
	procMgr      *process.Manager
	powerCalc    *power.Calculator
	powerMetrics *power.Metrics
	notifier     *notification.Notifier
	auditor      *notification.Auditor

	dashboard     *Dashboard
	processTable  *ProcessTable
	decisionPanel *DecisionPanel

	mu           sync.RWMutex
	processes    []*monitor.ProcessInfo
	sysMetrics   *monitor.SystemMetrics
	startTime    time.Time
	showAISugg   bool

	ctx    context.Context
	cancel context.CancelFunc
}

// NewApp creates the TUI application.
func NewApp(
	cfg *config.Config,
	mon *monitor.ProcessMonitor,
	aiEngine *ai.Engine,
	safetyMgr *safety.Manager,
	procMgr *process.Manager,
	powerCalc *power.Calculator,
	powerMetrics *power.Metrics,
	notifier *notification.Notifier,
	auditor *notification.Auditor,
) *App {
	app := &App{
		tapp:         tview.NewApplication(),
		cfg:          cfg,
		mon:          mon,
		aiEngine:     aiEngine,
		safetyMgr:    safetyMgr,
		procMgr:      procMgr,
		powerCalc:    powerCalc,
		powerMetrics: powerMetrics,
		notifier:     notifier,
		auditor:      auditor,
		startTime:    time.Now(),
		showAISugg:   true,
	}

	app.ctx, app.cancel = context.WithCancel(context.Background())

	app.dashboard = NewDashboard(app)
	app.processTable = NewProcessTable(app)
	app.decisionPanel = NewDecisionPanel(app)

	return app
}

// Run starts the TUI.
func (a *App) Run() error {
	// Layout: header + process table + decision panel
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.dashboard.view, 3, 0, false).
		AddItem(a.processTable.table, 0, 3, true).
		AddItem(a.decisionPanel.view, 10, 0, false).
		AddItem(a.createFooter(), 1, 0, false)

	a.tapp.SetRoot(mainFlex, true)
	setupKeybindings(a)

	// Start monitor in background
	a.mon.OnUpdate(func(procs []*monitor.ProcessInfo, metrics *monitor.SystemMetrics) {
		a.mu.Lock()
		a.processes = procs
		a.sysMetrics = metrics
		a.mu.Unlock()

		a.tapp.QueueUpdateDraw(func() {
			a.dashboard.Update(metrics)
			a.processTable.Update(procs)
		})
	})

	go a.mon.Start(a.ctx)

	return a.tapp.Run()
}

func (a *App) createFooter() *tview.TextView {
	footer := tview.NewTextView().
		SetDynamicColors(true).
		SetText(" [yellow]F1[white]:AI History [yellow]F2[white]:Suggestions [yellow]F3[white]:Power [yellow]F5[white]:Refresh [yellow]F6[white]:Sort [yellow]F7[white]:Aggr- [yellow]F8[white]:Aggr+ [yellow]F9[white]:Kill [yellow]F10[white]:Quit")
	footer.SetBackgroundColor(tcell.ColorDarkSlateGray)
	return footer
}

func (a *App) stop() {
	a.cancel()
	a.tapp.Stop()
}

func (a *App) getProcesses() []*monitor.ProcessInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.processes
}

func (a *App) getMetrics() *monitor.SystemMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.sysMetrics
}
