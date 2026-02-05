package ui

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"

	"github.com/iamgilwell/aura/internal/monitor"
)

func setupKeybindings(app *App) {
	app.tapp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF1:
			// Show AI decision history
			app.tapp.QueueUpdateDraw(func() {
				app.decisionPanel.ShowHistory()
			})
			return nil

		case tcell.KeyF2:
			// Toggle AI suggestions
			app.showAISugg = !app.showAISugg
			status := "OFF"
			if app.showAISugg {
				status = "ON"
			}
			app.tapp.QueueUpdateDraw(func() {
				app.decisionPanel.view.SetText(
					fmt.Sprintf("[yellow]AI Suggestions: %s", status))
			})
			return nil

		case tcell.KeyF3:
			// Show power metrics
			app.tapp.QueueUpdateDraw(func() {
				app.decisionPanel.ShowPowerMetrics()
			})
			return nil

		case tcell.KeyF4:
			// Show dependency graph for selected process
			pid := app.processTable.SelectedPID()
			if pid > 0 {
				app.tapp.QueueUpdateDraw(func() {
					showDependencyGraph(app, pid)
				})
			}
			return nil

		case tcell.KeyF5:
			// Refresh - no-op since monitor auto-refreshes
			return nil

		case tcell.KeyF6:
			// Cycle sort field
			app.processTable.CycleSort()
			procs := app.getProcesses()
			if procs != nil {
				app.tapp.QueueUpdateDraw(func() {
					app.processTable.Update(procs)
					app.decisionPanel.view.SetText(
						fmt.Sprintf("[yellow]Sorting by: %s", app.processTable.SortName()))
				})
			}
			return nil

		case tcell.KeyF7:
			// Decrease aggressiveness
			if app.cfg.AI.Aggressiveness > 1 {
				app.cfg.AI.Aggressiveness--
			}
			app.tapp.QueueUpdateDraw(func() {
				app.decisionPanel.view.SetText(
					fmt.Sprintf("[yellow]Aggressiveness: %d/10", app.cfg.AI.Aggressiveness))
			})
			return nil

		case tcell.KeyF8:
			// Increase aggressiveness
			if app.cfg.AI.Aggressiveness < 10 {
				app.cfg.AI.Aggressiveness++
			}
			app.tapp.QueueUpdateDraw(func() {
				app.decisionPanel.view.SetText(
					fmt.Sprintf("[yellow]Aggressiveness: %d/10", app.cfg.AI.Aggressiveness))
			})
			return nil

		case tcell.KeyF9:
			// Terminate selected process
			pid := app.processTable.SelectedPID()
			if pid > 0 {
				terminateSelected(app, pid)
			}
			return nil

		case tcell.KeyF10:
			// Quit
			app.stop()
			return nil

		case tcell.KeyRune:
			switch event.Rune() {
			case 'q', 'Q':
				app.stop()
				return nil
			case 'a', 'A':
				// AI evaluate selected process
				pid := app.processTable.SelectedPID()
				if pid > 0 && app.aiEngine != nil {
					go evaluateProcess(app, pid)
				}
				return nil
			}
		}

		return event
	})
}

func evaluateProcess(app *App, pid int) {
	procs := app.getProcesses()
	metrics := app.getMetrics()

	var proc *monitor.ProcessInfo
	for _, p := range procs {
		if p.PID == pid {
			proc = p
			break
		}
	}
	if proc == nil || metrics == nil {
		return
	}

	decision, err := app.aiEngine.EvaluateProcess(context.Background(), proc, metrics)
	if err != nil {
		app.tapp.QueueUpdateDraw(func() {
			app.decisionPanel.view.SetText(
				fmt.Sprintf("[red]AI evaluation failed: %v", err))
		})
		return
	}

	app.auditor.LogDecision(decision)
	app.tapp.QueueUpdateDraw(func() {
		app.decisionPanel.AddDecision(decision)
	})
}

func terminateSelected(app *App, pid int) {
	procs := app.getProcesses()

	var proc *monitor.ProcessInfo
	for _, p := range procs {
		if p.PID == pid {
			proc = p
			break
		}
	}
	if proc == nil {
		return
	}

	if app.safetyMgr.IsProtected(proc) {
		app.tapp.QueueUpdateDraw(func() {
			app.decisionPanel.view.SetText(
				fmt.Sprintf("[red]Cannot terminate protected process: %s (PID %d)", proc.Name, proc.PID))
		})
		return
	}

	err := app.procMgr.SafeTerminate(proc, false)
	if err != nil {
		app.tapp.QueueUpdateDraw(func() {
			app.decisionPanel.view.SetText(
				fmt.Sprintf("[red]Failed to terminate PID %d: %v", pid, err))
		})
		return
	}

	savings := app.powerCalc.EstimateSavings(proc)
	app.powerMetrics.RecordSaving(proc.Name, proc.PID, savings, "manual termination")
	app.auditor.LogTermination(proc.PID, proc.Name, "manual termination")

	app.tapp.QueueUpdateDraw(func() {
		app.decisionPanel.view.SetText(
			fmt.Sprintf("[green]Terminated: %s (PID %d) | Saved: %.2fW", proc.Name, proc.PID, savings))
	})
}

func showDependencyGraph(app *App, pid int) {
	procs := app.getProcesses()

	var proc *monitor.ProcessInfo
	var children []*monitor.ProcessInfo
	for _, p := range procs {
		if p.PID == pid {
			proc = p
		}
		if p.PPid == pid {
			children = append(children, p)
		}
	}

	if proc == nil {
		return
	}

	text := fmt.Sprintf("[yellow]Process Tree for PID %d (%s)[white]\n", pid, proc.Name)
	text += fmt.Sprintf("├─ Parent: PID %d\n", proc.PPid)
	text += fmt.Sprintf("├─ Category: %s\n", proc.Category)
	text += fmt.Sprintf("├─ CPU: %.1f%% | Mem: %.1f%%\n", proc.CPU, proc.Memory)

	if len(children) > 0 {
		text += fmt.Sprintf("└─ Children (%d):\n", len(children))
		for i, child := range children {
			prefix := "   ├─"
			if i == len(children)-1 {
				prefix = "   └─"
			}
			text += fmt.Sprintf("%s PID %d: %s (CPU: %.1f%%, Mem: %.1f%%)\n",
				prefix, child.PID, child.Name, child.CPU, child.Memory)
		}
	} else {
		text += "└─ No children\n"
	}

	app.decisionPanel.view.SetText(text)
}
