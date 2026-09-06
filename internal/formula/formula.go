package formula

import (
	"fmt"
	"math"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
)

type Stat struct {
	Token string
	Label string
}

var Stats = []Stat{
	{"kills", "Kills"}, {"deaths", "Deaths"}, {"assists", "Assists"},
	{"last_hits", "Last Hits"}, {"gpm", "GPM"}, {"xpm", "XPM"},
	{"stun_duration", "Stun"}, {"healing", "Healing"}, {"tower_damage", "Tower Dmg"},
	{"camps_stacked", "Camps Stacked"}, {"rune_pickups", "Rune Pickups"},
	{"first_blood", "First Blood"}, {"hero_damage", "Hero Dmg"},
	{"damage_taken", "Dmg Taken"}, {"gold_spent_wards", "Wards Gold"},
	{"gold_spent_smoke", "Smoke Gold"}, {"gold_spent_dust", "Dust Gold"},
	{"buffs_duration", "Buffs Duration"}, {"save", "Save Uptime"},
	{"purge", "Purge Uptime"}, {"shield_uptime", "Shield Uptime"},

	{"fear_duration", "Fear"}, {"roots_duration", "Roots"},
	{"leash_duration", "Leash"}, {"trap_duration", "Trap"},
	{"taunt_duration", "Taunt"}, {"silence_duration", "Silence"},
	{"break_duration", "Break"}, {"disarm_duration", "Disarm"},
	{"heal_duration", "Heal Time"}, {"heal_value", "Heal Value"},
	{"gold_lost", "Gold Lost"}, {"time_dead", "Time Dead"},
	{"wisdoms_captured", "Wisdoms Captured"}, {"watchers_captured", "Watchers Captured"},
	{"lotuses_gathered", "Lotuses Gathered"}, {"courier_kills", "Courier Kills"},
	{"match_duration", "Match Duration"}, {"position", "Position"},
}

var (
	statTokens  = map[string]bool{}
	statLabels  = map[string]string{}
	Funcs       = map[string]bool{"max": true, "min": true, "abs": true, "round": true}
	binaryOps   = map[string]bool{"+": true, "-": true, "*": true, "/": true, "**": true, "==": true, "!=": true, "<": true, ">": true, "<=": true, ">=": true, "&&": true, "||": true}
	unaryOps    = map[string]bool{"+": true, "-": true, "!": true}
	exprTerms   []struct{ token, key string }
	weightByKey = map[string]string{}
)

func init() {
	for _, st := range Stats {
		statTokens[st.Token] = true
		statLabels[st.Token] = st.Label
	}
	terms := []struct{ token, key string }{
		{"kills", "kills"}, {"deaths", "deaths"}, {"assists", "assists"},
		{"last_hits", "last_hits"}, {"gpm", "gpm"}, {"xpm", "xpm"},
		{"stun_duration", "stun"}, {"healing", "healing"}, {"tower_damage", "tower_damage"},
		{"camps_stacked", "camps"}, {"rune_pickups", "runes"}, {"first_blood", "first_blood"},
		{"hero_damage", "hero_damage"}, {"damage_taken", "damage_taken"},
		{"gold_spent_wards", "gold_spent_wards"}, {"gold_spent_smoke", "gold_spent_smoke"},
		{"gold_spent_dust", "gold_spent_dust"}, {"buffs_duration", "buffs_duration"},
		{"save", "save"}, {"purge", "purge"}, {"shield_uptime", "shield_uptime"},
		{"fear_duration", "fear_duration"}, {"roots_duration", "roots_duration"},
		{"leash_duration", "leash_duration"}, {"trap_duration", "trap_duration"},
		{"taunt_duration", "taunt_duration"}, {"silence_duration", "silence_duration"},
		{"break_duration", "break_duration"}, {"disarm_duration", "disarm_duration"},
		{"heal_duration", "heal_duration"}, {"heal_value", "heal_value"},
		{"gold_lost", "gold_lost"}, {"time_dead", "time_dead"},
		{"wisdoms_captured", "wisdoms_captured"}, {"watchers_captured", "watchers_captured"},
		{"lotuses_gathered", "lotuses_gathered"}, {"courier_kills", "courier_kills"},
	}
	exprTerms = terms
	for _, t := range terms {
		weightByKey["deaths"] = "deaths"
		weightByKey["deaths_base"] = "deaths_base"
		weightByKey[t.token] = t.key
	}
}

func StatTokens(token string) bool { return statTokens[token] }

func StatLabel(token string) string { return statLabels[token] }

var Examples = []struct{ Name, Expression string }{
	{"KDA ratio", "(kills * 3 + assists * 1.5) / max(deaths, 1)"},
	{"Support focus", "(kills * 2 + assists * 2.5) / max(deaths, 1) + healing * 0.005 + camps_stacked * 0.5 + stun_duration * 0.05"},
	{"Farm focus", "last_hits * 0.003 + gpm * 0.002 + xpm * 0.002 + tower_damage * 0.001"},
	{"Carry", "kills * 0.3 + (3 - deaths * 0.3) + assists * 0.15 + last_hits * 0.003 + gpm * 0.002 + xpm * 0.002"},
}

var DefaultLinearWeights = map[string]float64{
	"kills": 0.3, "deaths": 0.3, "deaths_base": 3.0, "assists": 0.15,
	"last_hits": 0.003, "gpm": 0.002, "xpm": 0.002, "stun": 0.05,
	"healing": 0.004, "tower_damage": 0.001, "camps": 0.5, "runes": 0.2,
	"first_blood": 1.0, "hero_damage": 0.0, "damage_taken": 0.0,
	"gold_spent_wards": 0.0, "gold_spent_smoke": 0.0, "gold_spent_dust": 0.0,
	"buffs_duration": 0.0, "save": 0.0, "purge": 0.0, "shield_uptime": 0.0,
	"fear_duration": 0.0, "roots_duration": 0.0, "leash_duration": 0.0,
	"trap_duration": 0.0, "taunt_duration": 0.0, "silence_duration": 0.0,
	"break_duration": 0.0, "disarm_duration": 0.0, "heal_duration": 0.0,
	"heal_value": 0.0, "gold_lost": 0.0,
}

const StandardV2Formula = "kills * 0.2 + assists * 0.15 + last_hits * 0.003 + xpm * 0.003" +
	" + tower_damage * 0.0007 + hero_damage * 0.00005" +
	" + (damage_taken / max(deaths, 1)) * 0.00015" +
	" + min(healing, 8000) * 0.001 + stun_duration * 0.02" +
	" + camps_stacked * 0.15 + rune_pickups * 0.1" +
	" + min(save + purge + shield_uptime, 500) * 0.015" +
	" + min(buffs_duration, 600) * 0.008" +
	" + gold_spent_wards * 0.0015 + gold_spent_smoke * 0.0015 + gold_spent_dust * 0.0015" +
	" + first_blood * 1 + (3 - deaths * 0.3)"

type FormulaError struct{ msg string }

func (e *FormulaError) Error() string { return e.msg }

func errf(format string, args ...any) error {
	return &FormulaError{msg: fmt.Sprintf(format, args...)}
}

func fmtWeight(v float64) string { return fmt.Sprintf("%g", v) }

func constValue(n ast.Node) (float64, bool) {
	switch node := n.(type) {
	case *ast.IntegerNode:
		return float64(node.Value), true
	case *ast.FloatNode:
		return node.Value, true
	case *ast.UnaryNode:
		if node.Operator != "+" && node.Operator != "-" {
			return 0, false
		}
		v, ok := constValue(node.Node)
		if !ok {
			return 0, false
		}
		if node.Operator == "-" {
			return -v, true
		}
		return v, true
	}
	return 0, false
}

func flattenAdd(node ast.Node) []ast.Node {
	if b, ok := node.(*ast.BinaryNode); ok && b.Operator == "+" {
		return append(flattenAdd(b.Left), flattenAdd(b.Right)...)
	}
	return []ast.Node{node}
}

func extractWeightTerm(term ast.Node, weights map[string]float64) bool {
	if v, ok := constValue(term); ok {
		if _, dup := weights["deaths_base"]; dup {
			return false
		}
		weights["deaths_base"] = v
		return true
	}
	if b, ok := term.(*ast.BinaryNode); ok && b.Operator == "*" {
		lv, lok := constValue(b.Left)
		rv, rok := constValue(b.Right)
		if name, ok := b.Left.(*ast.IdentifierNode); ok && rok {
			key, known := weightByKey[name.Value]
			if !known || key == "deaths" {
				return false
			}
			if _, dup := weights[key]; dup {
				return false
			}
			weights[key] = rv
			return true
		}
		if name, ok := b.Right.(*ast.IdentifierNode); ok && lok {
			key, known := weightByKey[name.Value]
			if !known || key == "deaths" {
				return false
			}
			if _, dup := weights[key]; dup {
				return false
			}
			weights[key] = lv
			return true
		}
		return false
	}
	if b, ok := term.(*ast.BinaryNode); ok && b.Operator == "-" {
		if base, ok := constValue(b.Left); ok {
			if sub, ok := b.Right.(*ast.BinaryNode); ok && sub.Operator == "*" {
				name, val, found := timesNameValue(sub)
				if !found || name != "deaths" {
					return false
				}
				if _, dup := weights["deaths"]; dup {
					return false
				}
				if _, dup := weights["deaths_base"]; dup {
					return false
				}
				weights["deaths_base"] = base
				weights["deaths"] = val
				return true
			}
		}
		return false
	}
	return false
}

func timesNameValue(sub *ast.BinaryNode) (name string, val float64, found bool) {
	lv, lok := constValue(sub.Left)
	rv, rok := constValue(sub.Right)
	if id, ok := sub.Left.(*ast.IdentifierNode); ok && rok {
		return id.Value, rv, true
	}
	if id, ok := sub.Right.(*ast.IdentifierNode); ok && lok {
		return id.Value, lv, true
	}
	return "", 0, false
}

func parseExpression(expression string) (*parser.Tree, error) {
	tree, err := parser.Parse(expression)
	if err != nil {
		return nil, err
	}
	return tree, nil
}

func ExpressionToWeights(expression string) (map[string]float64, bool) {
	tree, err := parseExpression(expression)
	if err != nil {
		return nil, false
	}
	weights := map[string]float64{}
	for _, term := range flattenAdd(tree.Node) {
		if !extractWeightTerm(term, weights) {
			return nil, false
		}
	}
	if len(weights) == 0 {
		return nil, false
	}
	return weights, true
}

func SplitExpression(expression string) (map[string]float64, string) {
	tree, err := parseExpression(expression)
	if err != nil {
		return map[string]float64{}, strings.TrimSpace(expression)
	}
	weights := map[string]float64{}
	var tail []string
	for _, term := range flattenAdd(tree.Node) {
		if extractWeightTerm(term, weights) {
			continue
		}
		tail = append(tail, term.String())
	}
	return weights, strings.Join(tail, " + ")
}

func LinearToExpression(weights map[string]float64) string {
	var parts []string
	for _, t := range exprTerms {
		if t.key == "deaths" {
			continue
		}
		if v, ok := weights[t.key]; ok && v != 0 {
			parts = append(parts, fmt.Sprintf("%s * %s", t.token, fmtWeight(v)))
		}
	}
	if v, ok := weights["deaths"]; ok && v != 0 {
		base := weights["deaths_base"]
		parts = append(parts, fmt.Sprintf("(%s - deaths * %s)", fmtWeight(base), fmtWeight(v)))
	} else if base, ok := weights["deaths_base"]; ok && base != 0 {
		parts = append(parts, fmt.Sprintf("%s - deaths * 0", fmtWeight(base)))
	}
	return strings.Join(parts, " + ")
}

func Validate(expression string, allowed map[string]bool) error {
	if strings.TrimSpace(expression) == "" {
		return errf("Формула пустая")
	}
	tree, err := parseExpression(expression)
	if err != nil {
		return errf("Ошибка в формуле: %v", err)
	}
	return checkNode(tree.Node, allowed)
}

func checkNode(n ast.Node, allowed map[string]bool) error {
	switch node := n.(type) {
	case *ast.IntegerNode, *ast.FloatNode, *ast.BoolNode:
		return nil
	case *ast.IdentifierNode:
		if allowed[node.Value] {
			return nil
		}
		if Funcs[node.Value] {
			return errf("'%s' используется неверно — нужно %s(число)", node.Value, node.Value)
		}
		return errf("Неизвестная переменная: %s. Доступно: %s", node.Value, availableTokens())
	case *ast.BinaryNode:
		if !binaryOps[node.Operator] {
			return errf("Недопустимая часть формулы: %s", typeName(n))
		}
		if err := checkNode(node.Left, allowed); err != nil {
			return err
		}
		return checkNode(node.Right, allowed)
	case *ast.UnaryNode:
		if !unaryOps[node.Operator] {
			return errf("Недопустимая часть формулы: %s", typeName(n))
		}
		return checkNode(node.Node, allowed)
	case *ast.CallNode:
		id, ok := node.Callee.(*ast.IdentifierNode)
		if !ok || !Funcs[id.Value] {
			return errf("Недопустимая часть формулы: %s", typeName(n))
		}
		for _, arg := range node.Arguments {
			if err := checkNode(arg, allowed); err != nil {
				return err
			}
		}
		return nil
	case *ast.BuiltinNode:
		if !Funcs[node.Name] {
			return errf("Недопустимая часть формулы: %s", typeName(n))
		}
		for _, arg := range node.Arguments {
			if err := checkNode(arg, allowed); err != nil {
				return err
			}
		}
		return nil
	case *ast.ConditionalNode:
		for _, sub := range []ast.Node{node.Cond, node.Exp1, node.Exp2} {
			if err := checkNode(sub, allowed); err != nil {
				return err
			}
		}
		return nil
	}
	return errf("Недопустимая часть формулы: %s", typeName(n))
}

func availableTokens() string {
	names := make([]string, 0, len(Stats))
	for _, st := range Stats {
		names = append(names, st.Token)
	}
	return strings.Join(names, ", ")
}

func typeName(n ast.Node) string {
	return fmt.Sprintf("%T", n)[4:] // strip the "ast." prefix
}

func Eval(expression string, vars map[string]float64) (float64, error) {
	allowed := map[string]bool{}
	for k := range vars {
		allowed[k] = true
	}
	if err := Validate(expression, allowed); err != nil {
		return 0, err
	}
	return run(expression, vars)
}

func run(expression string, vars map[string]float64) (float64, error) {
	env := make(map[string]any, len(vars))
	for k, v := range vars {
		env[k] = v
	}
	program, err := expr.Compile(expression)
	if err != nil {
		return 0, errf("Ошибка в формуле: %v", err)
	}
	out, err := expr.Run(program, env)
	if err != nil {
		return 0, errf("Ошибка вычисления: %v", err)
	}
	score, ok := toFloat(out)
	if !ok {
		return 0, errf("Ошибка вычисления: нечисловой результат")
	}
	if math.IsInf(score, 0) || math.IsNaN(score) {
		return 0, errf("Деление на ноль")
	}
	return score, nil
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint64:
		return float64(n), true
	}
	return 0, false
}
