package formula

import (
	"fmt"
	"math"
	"regexp"
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
	{"buff_duration", "Buff Duration"}, {"save_duration", "Save Duration"},
	{"purge_duration", "Purge Duration"}, {"shield_duration", "Shield Duration"},
	{"buff_stats_duration", "Buff Stats Duration"}, {"invisibility_duration", "Invisibility Duration"},
	{"buff_haste_duration", "Buff Haste Duration"},

	{"fear_duration", "Fear"}, {"roots_duration", "Roots"},
	{"leash_duration", "Leash"}, {"trap_duration", "Trap"},
	{"taunt_duration", "Taunt"}, {"silence_duration", "Silence"},
	{"break_duration", "Break"}, {"disarm_duration", "Disarm"},
	{"heal_duration", "Heal Time"}, {"heal_value", "Heal Value"},
	{"time_dead", "Time Dead"}, {"creeps_stacked", "Creeps Stacked"},
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
		{"stun_duration", "stun_duration"}, {"healing", "healing"}, {"tower_damage", "tower_damage"},
		{"camps_stacked", "camps_stacked"}, {"rune_pickups", "rune_pickups"}, {"first_blood", "first_blood"},
		{"hero_damage", "hero_damage"}, {"damage_taken", "damage_taken"},
		{"gold_spent_wards", "gold_spent_wards"}, {"gold_spent_smoke", "gold_spent_smoke"},
		{"gold_spent_dust", "gold_spent_dust"}, {"buff_duration", "buff_duration"},
		{"save_duration", "save_duration"}, {"purge_duration", "purge_duration"}, {"shield_duration", "shield_duration"},
		{"buff_stats_duration", "buff_stats_duration"}, {"invisibility_duration", "invisibility_duration"},
		{"buff_haste_duration", "buff_haste_duration"},
		{"fear_duration", "fear_duration"}, {"roots_duration", "roots_duration"},
		{"leash_duration", "leash_duration"}, {"trap_duration", "trap_duration"},
		{"taunt_duration", "taunt_duration"}, {"silence_duration", "silence_duration"},
		{"break_duration", "break_duration"}, {"disarm_duration", "disarm_duration"},
		{"heal_duration", "heal_duration"}, {"heal_value", "heal_value"},
		{"time_dead", "time_dead"}, {"creeps_stacked", "creeps_stacked"},
		{"wisdoms_captured", "wisdoms_captured"}, {"watchers_captured", "watchers_captured"},
		{"lotuses_gathered", "lotuses_gathered"}, {"courier_kills", "courier_kills"},
	}
	exprTerms = terms
	for _, t := range terms {
		weightByKey[t.token] = t.key
	}
}

func StatTokens(token string) bool { return statTokens[token] }

func StatLabel(token string) string { return statLabels[token] }

var Examples = []struct{ Name, Expression string }{
	{"KDA ratio", "(kills * 3 + assists * 1.5) / max(deaths, 1)"},
	{"Support focus", "(kills * 2 + assists * 2.5) / max(deaths, 1) + healing * 0.005 + camps_stacked * 0.5 + stun_duration * 0.05"},
	{"Farm focus", "last_hits * 0.003 + gpm * 0.002 + xpm * 0.002 + tower_damage * 0.001"},
	{"Carry", "kills * 0.3 + (0 - deaths * 0.3) + assists * 0.15 + last_hits * 0.003 + gpm * 0.002 + xpm * 0.002"},
}

var DefaultLinearWeights = map[string]float64{
	"kills": 0.3, "deaths": 0.3, "assists": 0.15,
	"last_hits": 0.003, "gpm": 0.002, "xpm": 0.002, "stun_duration": 0.05,
	"healing": 0.004, "tower_damage": 0.001, "camps_stacked": 0.5, "rune_pickups": 0.2,
	"first_blood": 1.0, "hero_damage": 0.0, "damage_taken": 0.0,
	"gold_spent_wards": 0.0, "gold_spent_smoke": 0.0, "gold_spent_dust": 0.0,
	"buff_duration": 0.0, "save_duration": 0.0, "purge_duration": 0.0, "shield_duration": 0.0,
	"buff_stats_duration": 0.0, "invisibility_duration": 0.0, "buff_haste_duration": 0.0,
	"fear_duration": 0.0, "roots_duration": 0.0, "leash_duration": 0.0,
	"trap_duration": 0.0, "taunt_duration": 0.0, "silence_duration": 0.0,
	"break_duration": 0.0, "disarm_duration": 0.0, "heal_duration": 0.0,
	"heal_value": 0.0, "time_dead": 0.0, "creeps_stacked": 0.0,
}

const TutorialFormula = `kills * 0.2 + assists * 0.15 + last_hits * 0.003 + xpm * 0.003` +
	` + (damage_taken / max(deaths, 1)) * 0.00015 + min(healing, 8000) * 0.001` +
	` + (if position == 1 { kills * 2 } else if position == 2 { kills * 1.5 } else { kills })` +
	` + (if kills > 10 && deaths < 5 { 3 } else { 0 })` +
	` + (if !(assists < 3) { 2 } else { 0 })` +
	` + (stun_duration >= 30 ? 1.5 : 0)` +
	` + round(save_duration * 0.01)` +
	` + min(save_duration + purge_duration + shield_duration, 500) * 0.015 + min(buff_duration, 600) * 0.008`

const StandardV2Formula = "kills * 0.2 + assists * 0.15 + last_hits * 0.003 + xpm * 0.003" +
	" + tower_damage * 0.0007 + hero_damage * 0.00005" +
	" + (damage_taken / max(deaths, 1)) * 0.00015" +
	" + min(healing, 8000) * 0.001 + stun_duration * 0.02" +
	" + camps_stacked * 0.15 + rune_pickups * 0.1" +
	" + min(save_duration + purge_duration + shield_duration, 500) * 0.015" +
	" + min(buff_duration, 600) * 0.008" +
	" + gold_spent_wards * 0.0015 + gold_spent_smoke * 0.0015 + gold_spent_dust * 0.0015" +
	" + first_blood * 1 + (0 - deaths * 0.2)"

type FormulaError struct{ msg string }

func (e *FormulaError) Error() string { return e.msg }

func errf(format string, args ...any) error {
	return &FormulaError{msg: fmt.Sprintf(format, args...)}
}

func fmtWeight(v float64) string { return fmt.Sprintf("%g", v) }

func stripComments(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	i, n := 0, len(s)
	for i < n {
		if s[i] == '#' || (s[i] == '/' && i+1 < n && s[i+1] == '/') {
			for i < n && s[i] != '\n' {
				i++
			}
			continue
		}
		if s[i] == '/' && i+1 < n && s[i+1] == '*' {
			i += 2
			for i+1 < n && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			i += 2
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// Legacy token → current token. Removed tokens (deaths_base, gold_lost) are
// neutralized to 0 so old saved formulas keep validating.
var migrateRe = regexp.MustCompile(`\b(buffs_duration|deaths_base|save|purge|shield_uptime|gold_lost)\b`)

func migrate(s string) string {
	return migrateRe.ReplaceAllStringFunc(s, func(w string) string {
		switch w {
		case "buffs_duration":
			return "buff_duration"
		case "save":
			return "save_duration"
		case "purge":
			return "purge_duration"
		case "shield_uptime":
			return "shield_duration"
		default:
			return "0"
		}
	})
}

// MigrateExpression rewrites legacy stat tokens to their current names. Used
// when loading stored presets/variables so saved formulas keep working.
func MigrateExpression(s string) string { return migrate(stripComments(s)) }

// splitStatements splits a formula on top-level ";" separators (respecting
// parentheses and braces) so multiline formulas can be written as:
//
//	kills * 0.2;
//	assists * 0.15
func splitStatements(s string) []string {
	var stmts []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '{':
			depth++
		case ')', '}':
			if depth > 0 {
				depth--
			}
		case ';':
			if depth == 0 {
				if t := strings.TrimSpace(s[start:i]); t != "" {
					stmts = append(stmts, t)
				}
				start = i + 1
			}
		}
	}
	if t := strings.TrimSpace(s[start:]); t != "" {
		stmts = append(stmts, t)
	}
	return stmts
}

// prepareFormula normalizes a formula for parsing/evaluation: strips comments,
// migrates legacy tokens, and joins `;`-separated statements back into a sum.
func prepareFormula(s string) string {
	var joined []string
	for _, stmt := range splitStatements(stripComments(s)) {
		joined = append(joined, migrate(stmt))
	}
	return strings.Join(joined, " + ")
}

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
		if sub, ok := b.Right.(*ast.BinaryNode); ok && sub.Operator == "*" {
			name, val, found := timesNameValue(sub)
			if !found || name != "deaths" {
				return false
			}
			if _, dup := weights["deaths"]; dup {
				return false
			}
			weights["deaths"] = val
			return true
		}
		return false
	}
	if _, ok := constValue(term); ok {
		return true
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
	tree, err := parser.Parse(prepareFormula(expression))
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
		parts = append(parts, fmt.Sprintf("(0 - deaths * %s)", fmtWeight(v)))
	}
	return strings.Join(parts, " + ")
}

func Validate(expression string, allowed map[string]bool) error {
	if strings.TrimSpace(prepareFormula(expression)) == "" {
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
	program, err := expr.Compile(prepareFormula(expression))
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

// BreakdownRow is one statement of a formula and its evaluated score for a
// single player (used for the per-stat/per-statement breakdown table).
type BreakdownRow struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

// Breakdown evaluates each top-level term of a formula separately, returning
// one row per term in source order. Terms are split on `+`/`-` at brace depth
// zero (so flat sums without `;` decompose too); `;`-statements act as stronger
// boundaries. The sum of the rows equals the full-formula score. If term
// splitting fails or yields a single term, the whole statement becomes one row.
func Breakdown(expression string, vars map[string]float64) ([]BreakdownRow, error) {
	env := make(map[string]any, len(vars))
	for k, v := range vars {
		env[k] = v
	}
	rows := []BreakdownRow{}
	for _, raw := range splitStatements(stripComments(expression)) {
		stmt := migrate(raw)
		trimmed := strings.TrimSpace(stmt)
		if trimmed == "" {
			continue
		}
		if terms := splitTerms(stmt); len(terms) > 1 {
			if termRows, ok := evalTerms(terms, env); ok {
				rows = append(rows, termRows...)
				continue
			}
		}
		program, err := expr.Compile(stmt)
		if err != nil {
			return nil, errf("Ошибка в формуле: %v", err)
		}
		out, err := expr.Run(program, env)
		if err != nil {
			return nil, errf("Ошибка вычисления: %v", err)
		}
		v, ok := toFloat(out)
		if !ok {
			return nil, errf("Ошибка вычисления: нечисловой результат")
		}
		if math.IsInf(v, 0) || math.IsNaN(v) {
			return nil, errf("Деление на ноль")
		}
		rows = append(rows, BreakdownRow{Label: trimmed, Value: v})
	}
	return rows, nil
}

// evalTerms evaluates each term independently, returning ok=false if any term
// fails to compile or evaluate so the caller can fall back to the whole
// statement.
func evalTerms(terms []string, env map[string]any) ([]BreakdownRow, bool) {
	rows := make([]BreakdownRow, 0, len(terms))
	for _, t := range terms {
		trimmed := strings.TrimSpace(t)
		if trimmed == "" {
			continue
		}
		program, err := expr.Compile(trimmed)
		if err != nil {
			return nil, false
		}
		out, err := expr.Run(program, env)
		if err != nil {
			return nil, false
		}
		v, ok := toFloat(out)
		if !ok || math.IsInf(v, 0) || math.IsNaN(v) {
			return nil, false
		}
		rows = append(rows, BreakdownRow{Label: trimmed, Value: v})
	}
	return rows, true
}

// splitTerms splits a top-level statement into additive terms on `+`/`-` at
// brace depth zero. A leading `-` sign is kept with its term so each term
// evaluates independently. Signs inside parentheses/braces or in unary position
// (e.g. `** -2`) are left untouched.
func splitTerms(stmt string) []string {
	terms := []string{}
	depth := 0
	start := 0
	for i := 0; i < len(stmt); i++ {
		switch stmt[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case '+', '-':
			if depth > 0 || !prevOperandEnd(stmt, i) {
				continue
			}
			terms = append(terms, strings.TrimSpace(stmt[start:i]))
			start = i
			if stmt[i] == '+' {
				start = i + 1
			}
		}
	}
	return append(terms, strings.TrimSpace(stmt[start:]))
}

// prevOperandEnd reports whether the last significant character before i
// completes a value (digit, identifier character or closing bracket), i.e. the
// operator at i is a binary one rather than a unary sign.
func prevOperandEnd(s string, i int) bool {
	for i--; i >= 0 && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n'); i-- {
	}
	if i < 0 {
		return false
	}
	c := s[i]
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c == ')' || c == ']' || c == '}'
}
