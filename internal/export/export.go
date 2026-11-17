// Package export writes saved reports as Markdown: readable in any editor,
// and pasted as is into an issue, a forum post or a message to whoever helps.
package export

import (
	"fmt"
	"strings"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/recorder"
)

// Estimate closes every report, as it closes every finding in the window.
const Estimate = "These are estimates drawn from what Windows recorded, not a diagnosis. " +
	"Do not rely on them alone; what you do with them is your decision and your responsibility."

const stamp = "2006-01-02 15:04"

type doc struct{ strings.Builder }

func (d *doc) line(format string, a ...any) { fmt.Fprintf(&d.Builder, format+"\n", a...) }

func (d *doc) footer() string {
	d.line("\n---\n\n%s\n\nMade with ReforgePulse.", Estimate)
	return d.String()
}

// Scan is a saved scan with its health score, findings and suspects.
func Scan(r domain.ScanResult) string {
	var d doc
	d.line("# ReforgePulse scan, %s\n", r.At.Local().Format(stamp))
	d.line("Health: **%d out of 100**.", r.Health.Score)
	if !r.Elevated {
		d.line("\nRan without administrator rights, so some logs were not read.")
	}
	if len(r.Health.Deductions) > 0 {
		d.line("\n| Points | Finding |\n|---:|---|")
		for _, x := range r.Health.Deductions {
			d.line("| -%d | %s |", x.Points, cell(x.Title))
		}
	}
	d.line("\n## Findings\n")
	if len(r.Findings) == 0 {
		d.line("Nothing in the last 30 days of records needed attention.")
	}
	for _, f := range r.Findings {
		d.line("### %s\n", f.Title)
		d.line("%s severity, %s.%s\n", capital(string(f.Severity)), f.Confidence, status(f))
		d.line("%s\n", f.Summary)
		if len(f.Evidence) > 0 {
			d.line("What Windows recorded:\n")
			for _, e := range f.Evidence {
				d.line("- `%s`", e)
			}
			d.line("")
		}
		if f.Alternatives != "" {
			d.line("Other explanations the evidence allows: %s\n", f.Alternatives)
		}
	}
	if len(r.Suspects) > 0 {
		d.line("## Where the instability points\n")
		for _, s := range r.Suspects {
			d.line("### %s\n", s.Title)
			for _, x := range s.For {
				d.line("- For: %s", x)
			}
			for _, x := range s.Against {
				d.line("- Against: %s", x)
			}
			for _, x := range s.Checks {
				d.line("- To check next: %s", x)
			}
			d.line("")
		}
	}
	return d.footer()
}

func status(f domain.Finding) string {
	switch f.Status {
	case domain.Resolved:
		return fmt.Sprintf(" Marked resolved %s.", f.ResolvedAt.Local().Format(stamp))
	case domain.Returned:
		return fmt.Sprintf(" Marked resolved %s, seen again %s.", f.ResolvedAt.Local().Format(stamp), f.Latest.Local().Format(stamp))
	}
	return ""
}

// Analysis is the debugger's verdict on one crash dump, with its output.
func Analysis(a domain.SavedAnalysis) string {
	var d doc
	d.line("# Crash dump %s, %s\n", a.Name, a.Dumped.Local().Format(stamp))
	d.line("| | |\n|---|---|")
	d.line("| Blames | %s |", cell(or(a.Image, a.Module, "Not named")))
	d.line("| Stopped in | `%s` |", or(a.Symbol, "-"))
	d.line("| Process | %s |", cell(or(a.Process, "-")))
	d.line("| Failure bucket | `%s` |", or(a.Bucket, "-"))
	if !a.At.IsZero() {
		d.line("| Analysed | %s |", a.At.Local().Format(stamp))
	}
	d.line("\nThe module the debugger names is where the crash happened. That is usually, not always, the cause: unstable hardware makes healthy drivers crash too.")
	if a.Output != "" {
		d.line("\n## Debugger output\n\n```\n%s\n```", strings.TrimRight(a.Output, "\n"))
	}
	return d.footer()
}

// Recording is a recorder session: load and what each process did.
func Recording(r recorder.Recording) string {
	var d doc
	d.line("# Recording, %s, %d min\n", r.Started.Local().Format(stamp), max(1, r.Seconds/60))
	d.line("| | Peak | Average |\n|---|---:|---:|")
	for _, s := range []struct {
		name string
		v    []float64
	}{{"Processor", r.CPU}, {"Graphics", r.GPU}, {"Memory", r.Memory}} {
		peak, avg := stats(s.v)
		d.line("| %s | %.0f%% | %.0f%% |", s.name, peak, avg)
	}
	d.line("\n## Processes by highest memory\n")
	d.line("| Process | PID | Start | End | Highest | Growth | CPU | |\n|---|---:|---:|---:|---:|---:|---:|---|")
	for _, p := range r.Top {
		leak := ""
		if p.Leak {
			leak = "likely leak"
		}
		d.line("| %s | %d | %.0f MB | %.0f MB | %.0f MB | %.1f MB/min | %.1f%% | %s |", cell(p.Name), p.PID, p.StartMB, p.EndMB, p.PeakMB, p.MBPerMinute, p.AvgCPU, leak)
	}
	d.line("\nA likely leak is memory that kept growing steadily, not a single jump.")
	return d.footer()
}

// Check is a health check's verdict and where Windows keeps its full log.
func Check(c domain.CheckResult, title string) string {
	var d doc
	d.line("# %s check, %s\n", title, c.At.Local().Format(stamp))
	d.line("```\n%s\n```\n", or(c.Summary, "It printed no verdict."))
	if c.FromWindows {
		d.line("Windows keeps this result in its System event log.")
	} else {
		d.line("Exit code %d.", c.ExitCode)
	}
	if c.Log != "" {
		d.line("\nThe full log is in `%s`.", c.Log)
	}
	return d.footer()
}

// cell keeps a value from closing a Markdown table cell early.
func cell(s string) string { return strings.ReplaceAll(s, "|", `\|`) }

func stats(v []float64) (peak, avg float64) {
	for _, x := range v {
		peak, avg = max(peak, x), avg+x
	}
	if len(v) > 0 {
		avg /= float64(len(v))
	}
	return peak, avg
}

func or(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func capital(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
