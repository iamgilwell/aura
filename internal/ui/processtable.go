package ui

import (
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/iamgilwell/aura/internal/monitor"
)

// SortField determines how the process table is sorted.
type SortField int

const (
	SortByCPU SortField = iota
	SortByMemory
	SortByPID
	SortByName
	SortByIO
)

var sortFieldNames = []string{"CPU%", "MEM%", "PID", "NAME", "IO"}

// ProcessTable displays processes in an htop-like table.
type ProcessTable struct {
	app       *App
	table     *tview.Table
	sortField SortField
	sortDesc  bool
}

// NewProcessTable creates the process table.
func NewProcessTable(app *App) *ProcessTable {
	table := tview.NewTable().
		SetSelectable(true, false).
		SetFixed(1, 0).
		SetSeparator(tview.Borders.Vertical)

	table.SetBorder(true).
		SetTitle(" Processes ").
		SetBorderPadding(0, 0, 0, 0)

	pt := &ProcessTable{
		app:       app,
		table:     table,
		sortField: SortByCPU,
		sortDesc:  true,
	}

	pt.setHeaders()
	return pt
}

func (pt *ProcessTable) setHeaders() {
	headers := []string{"PID", "NAME", "USER", "CPU%", "MEM%", "MEM(MB)", "IO(R+W)", "CAT", "STATE", "COMMAND"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetAttributes(tcell.AttrBold).
			SetSelectable(false).
			SetExpansion(1)
		pt.table.SetCell(0, i, cell)
	}
}

// Update refreshes the process table with new data.
func (pt *ProcessTable) Update(procs []*monitor.ProcessInfo) {
	// Sort processes
	sorted := make([]*monitor.ProcessInfo, len(procs))
	copy(sorted, procs)
	pt.sortProcesses(sorted)

	// Clear existing rows (keep header)
	rowCount := pt.table.GetRowCount()
	for r := rowCount - 1; r >= 1; r-- {
		pt.table.RemoveRow(r)
	}

	for i, p := range sorted {
		row := i + 1 // skip header

		catColor := tcell.ColorWhite
		switch p.Category {
		case monitor.CategoryKernel:
			catColor = tcell.ColorGray
		case monitor.CategorySystem:
			catColor = tcell.ColorBlue
		case monitor.CategoryEssential:
			catColor = tcell.ColorRed
		case monitor.CategoryUser:
			catColor = tcell.ColorGreen
		}

		cpuColor := tcell.ColorWhite
		if p.CPU > 50 {
			cpuColor = tcell.ColorRed
		} else if p.CPU > 20 {
			cpuColor = tcell.ColorYellow
		}

		memColor := tcell.ColorWhite
		if p.Memory > 50 {
			memColor = tcell.ColorRed
		} else if p.Memory > 20 {
			memColor = tcell.ColorYellow
		}

		ioTotal := p.IORead + p.IOWrite
		ioStr := formatBytes(ioTotal)

		pt.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", p.PID)).SetTextColor(tcell.ColorWhite))
		pt.table.SetCell(row, 1, tview.NewTableCell(truncate(p.Name, 25)).SetTextColor(catColor))
		pt.table.SetCell(row, 2, tview.NewTableCell(truncate(p.User, 10)).SetTextColor(tcell.ColorWhite))
		pt.table.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("%.1f", p.CPU)).SetTextColor(cpuColor))
		pt.table.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("%.1f", p.Memory)).SetTextColor(memColor))
		pt.table.SetCell(row, 5, tview.NewTableCell(fmt.Sprintf("%.1f", p.MemoryMB)).SetTextColor(tcell.ColorWhite))
		pt.table.SetCell(row, 6, tview.NewTableCell(ioStr).SetTextColor(tcell.ColorWhite))
		pt.table.SetCell(row, 7, tview.NewTableCell(p.Category.String()).SetTextColor(catColor))
		pt.table.SetCell(row, 8, tview.NewTableCell(p.State).SetTextColor(tcell.ColorWhite))
		pt.table.SetCell(row, 9, tview.NewTableCell(truncate(p.Cmdline, 50)).SetTextColor(tcell.ColorGray))
	}
}

// SelectedPID returns the PID of the currently selected process.
func (pt *ProcessTable) SelectedPID() int {
	row, _ := pt.table.GetSelection()
	if row < 1 {
		return -1
	}
	cell := pt.table.GetCell(row, 0)
	if cell == nil {
		return -1
	}
	var pid int
	fmt.Sscanf(cell.Text, "%d", &pid)
	return pid
}

// CycleSort advances to the next sort field.
func (pt *ProcessTable) CycleSort() {
	pt.sortField = (pt.sortField + 1) % 5
}

// SortName returns the current sort field name.
func (pt *ProcessTable) SortName() string {
	return sortFieldNames[pt.sortField]
}

func (pt *ProcessTable) sortProcesses(procs []*monitor.ProcessInfo) {
	sort.Slice(procs, func(i, j int) bool {
		var less bool
		switch pt.sortField {
		case SortByCPU:
			less = procs[i].CPU < procs[j].CPU
		case SortByMemory:
			less = procs[i].Memory < procs[j].Memory
		case SortByPID:
			less = procs[i].PID < procs[j].PID
		case SortByName:
			less = procs[i].Name < procs[j].Name
		case SortByIO:
			less = (procs[i].IORead + procs[i].IOWrite) < (procs[j].IORead + procs[j].IOWrite)
		}
		if pt.sortDesc {
			return !less
		}
		return less
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func formatBytes(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.1fG", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1fM", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1fK", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%dB", b)
	}
}
