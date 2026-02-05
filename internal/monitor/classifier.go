package monitor

import "strings"

// knownSystemDaemons are processes that are typically system services.
var knownSystemDaemons = map[string]bool{
	"systemd":        true,
	"init":           true,
	"rsyslogd":       true,
	"syslogd":        true,
	"journald":       true,
	"udevd":          true,
	"dbus-daemon":    true,
	"polkitd":        true,
	"accounts-daemon": true,
	"cron":           true,
	"atd":            true,
	"acpid":          true,
	"thermald":       true,
	"irqbalance":     true,
	"snapd":          true,
	"packagekitd":    true,
	"udisksd":        true,
	"colord":         true,
	"cupsd":          true,
	"avahi-daemon":   true,
	"bluetoothd":     true,
	"wpa_supplicant": true,
	"dhclient":       true,
	"NetworkManager": true,
	"ModemManager":   true,
}

// Classifier categorizes processes.
type Classifier struct {
	protectedNames map[string]bool
}

// NewClassifier creates a new process classifier.
func NewClassifier(protectedNames []string) *Classifier {
	pmap := make(map[string]bool, len(protectedNames))
	for _, name := range protectedNames {
		pmap[name] = true
	}
	return &Classifier{protectedNames: pmap}
}

// Classify determines the ProcessCategory for a process.
func (c *Classifier) Classify(p *ProcessInfo) ProcessCategory {
	// Kernel threads: PID 1 or 2, or PPID is kthreadd (PID 2)
	if p.PID <= 2 || p.PPid == 2 {
		return CategoryKernel
	}

	// Kernel threads have empty cmdline and name in brackets
	if p.Cmdline == "" || (strings.HasPrefix(p.Cmdline, "[") && strings.HasSuffix(p.Cmdline, "]")) {
		return CategoryKernel
	}

	// Essential / protected processes
	if c.protectedNames[p.Name] {
		return CategoryEssential
	}

	// System daemons
	if p.UID == 0 || knownSystemDaemons[p.Name] {
		return CategorySystem
	}

	return CategoryUser
}
