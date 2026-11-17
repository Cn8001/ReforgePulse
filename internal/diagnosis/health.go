package diagnosis

import (
	"sort"

	"github.com/cn8001/reforgepulse/internal/domain"
)

// cost is what an open finding takes off the score. Info costs nothing: it
// is something to know, not something wrong.
var cost = map[domain.Severity]int{
	domain.Critical: 25, domain.High: 15, domain.Medium: 7, domain.Low: 2,
}

// Health scores findings out of 100. A finding marked resolved costs
// nothing until it comes back. It is an estimate, as the findings are.
func Health(findings []domain.Finding) domain.Health {
	h := domain.Health{Score: 100, Deductions: []domain.Deduction{}}
	for _, f := range findings {
		p := cost[f.Severity]
		if p == 0 || f.Status == domain.Resolved {
			continue
		}
		h.Deductions = append(h.Deductions, domain.Deduction{Key: f.Key, Title: f.Title, Points: p})
		h.Score -= p
	}
	h.Score = max(h.Score, 0)
	sort.SliceStable(h.Deductions, func(i, j int) bool { return h.Deductions[i].Points > h.Deductions[j].Points })
	return h
}
