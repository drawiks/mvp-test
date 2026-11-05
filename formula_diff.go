package main

import (
	"strings"

	"mvp/internal/formula"
)

type FormulaDiff struct {
	Current  string   `json:"current"`
	Proposed string   `json:"proposed"`
	Added    []string `json:"added"`
	Removed  []string `json:"removed"`
}

// GetActiveFormulaExpression returns the active preset as an expression.
func (a *App) GetActiveFormulaExpression() string {
	p := a.store.Active()
	if p.Kind == "expression" {
		return strings.TrimSpace(p.Expression)
	}
	return formula.LinearToExpression(p.Weights)
}

// DiffFormula compares two formulas by their top-level additive terms.
func (a *App) DiffFormula(current, proposed string) FormulaDiff {
	counts := map[string]int{}
	for _, term := range formula.Terms(current) {
		counts[term]++
	}
	added := []string{}
	for _, term := range formula.Terms(proposed) {
		if counts[term] > 0 {
			counts[term]--
			continue
		}
		added = append(added, term)
	}
	removed := []string{}
	for term, n := range counts {
		for i := 0; i < n; i++ {
			removed = append(removed, term)
		}
	}
	return FormulaDiff{
		Current: strings.TrimSpace(current), Proposed: strings.TrimSpace(proposed),
		Added: added, Removed: removed,
	}
}
