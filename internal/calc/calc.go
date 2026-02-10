package calc

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"smartcalc/internal/cert"
	"smartcalc/internal/color"
	"smartcalc/internal/constants"
	"smartcalc/internal/cooking"
	"smartcalc/internal/crontab"
	"smartcalc/internal/datetime"
	"smartcalc/internal/eval"
	"smartcalc/internal/finance"
	"smartcalc/internal/hourlycost"
	"smartcalc/internal/httpstatus"
	"smartcalc/internal/jwt"
	"smartcalc/internal/manhour"
	"smartcalc/internal/network"
	"smartcalc/internal/percentage"
	"smartcalc/internal/permissions"
	"smartcalc/internal/programmer"
	"smartcalc/internal/radio"
	"smartcalc/internal/regex"
	"smartcalc/internal/stats"
	"smartcalc/internal/units"
	"smartcalc/internal/utils"
)

// tryBaseConversion handles expressions like "24 in dec", "25 in hex", "25 in oct", "25 in bin"
// Also handles hex input like "0xFF in dec" or "0b1010 in dec"
func tryBaseConversion(expr string) (string, bool) {
	exprLower := strings.ToLower(strings.TrimSpace(expr))

	// Pattern: "number in base"
	re := regexp.MustCompile(`(?i)^(0x[0-9a-fA-F]+|0b[01]+|0o[0-7]+|\d+)\s+in\s+(dec|decimal|hex|hexadecimal|oct|octal|bin|binary)$`)
	matches := re.FindStringSubmatch(exprLower)
	if matches == nil {
		return "", false
	}

	numStr := matches[1]
	targetBase := matches[2]

	// Parse the input number
	var value int64
	var err error

	if strings.HasPrefix(numStr, "0x") {
		// Hexadecimal input
		value, err = strconv.ParseInt(numStr[2:], 16, 64)
	} else if strings.HasPrefix(numStr, "0b") {
		// Binary input
		value, err = strconv.ParseInt(numStr[2:], 2, 64)
	} else if strings.HasPrefix(numStr, "0o") {
		// Octal input
		value, err = strconv.ParseInt(numStr[2:], 8, 64)
	} else {
		// Decimal input
		value, err = strconv.ParseInt(numStr, 10, 64)
	}

	if err != nil {
		return "", false
	}

	// Convert to target base
	var result string
	switch {
	case strings.HasPrefix(targetBase, "dec"):
		result = strconv.FormatInt(value, 10)
	case strings.HasPrefix(targetBase, "hex"):
		result = "0x" + strings.ToUpper(strconv.FormatInt(value, 16))
	case strings.HasPrefix(targetBase, "oct"):
		result = "0o" + strconv.FormatInt(value, 8)
	case strings.HasPrefix(targetBase, "bin"):
		result = "0b" + strconv.FormatInt(value, 2)
	default:
		return "", false
	}

	return result, true
}

// isBaseConversionExpr checks if expression is a base conversion
func isBaseConversionExpr(expr string) bool {
	exprLower := strings.ToLower(expr)
	return strings.Contains(exprLower, " in dec") ||
		strings.Contains(exprLower, " in hex") ||
		strings.Contains(exprLower, " in oct") ||
		strings.Contains(exprLower, " in bin") ||
		strings.Contains(exprLower, " in decimal") ||
		strings.Contains(exprLower, " in hexadecimal") ||
		strings.Contains(exprLower, " in octal") ||
		strings.Contains(exprLower, " in binary")
}

// findResultEquals finds the position of the trailing '=' that marks the result,
// skipping '=' characters that are part of comparison operators (>=, <=, ==, !=)
// or base64 padding (trailing = or == without space before).
// Returns -1 if no result '=' is found.
func findResultEquals(s string) int {
	// Find the last '=' that is a result delimiter (has space before it)
	// and is not part of a comparison operator
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '=' {
			// Check if this '=' is part of >=, <=, ==, or !=
			if i > 0 {
				prev := s[i-1]
				if prev == '>' || prev == '<' || prev == '=' || prev == '!' {
					continue // Skip this '=', it's part of a comparison operator
				}
				// Result delimiter should have a space before it (e.g., "2 + 2 =")
				// Skip '=' that doesn't have space before (likely base64 padding)
				if prev != ' ' {
					continue
				}
			}
			return i
		}
	}
	return -1
}

// extractInlineComment extracts an inline comment from a line.
// Returns the comment string (including the # prefix) if found, empty string otherwise.
// The comment must appear after the result '=' to be preserved.
func extractInlineComment(line string, eqPos int) string {
	// Look for # after the = sign
	afterEq := line[eqPos+1:]
	hashIdx := strings.Index(afterEq, "#")
	if hashIdx >= 0 {
		// Preserve the comment exactly as typed, no trimming
		comment := afterEq[hashIdx:]
		return " " + comment
	}
	return ""
}

// isComparisonExpr checks if an expression contains comparison operators
func isComparisonExpr(expr string) bool {
	// Check for comparison operators: >, <, >=, <=, ==, !=
	if strings.Contains(expr, ">=") || strings.Contains(expr, "<=") ||
		strings.Contains(expr, "==") || strings.Contains(expr, "!=") {
		return true
	}
	// Check for single > or < (but not part of >= or <=)
	for i, r := range expr {
		if r == '>' || r == '<' {
			// Make sure it's not followed by =
			if i+1 < len(expr) && expr[i+1] == '=' {
				continue
			}
			return true
		}
	}
	return false
}

// formatExpression adds proper spacing around operators in an expression.
// Transforms "2+3" to "2 + 3", "$100-20%" to "$100 - 20%", etc.
func formatExpression(expr string) string {
	// Operators that should have spaces around them
	// Handle multi-char operators first, then single-char
	result := expr

	// Add space around basic operators: +, -, *, /, x, ×, ÷, ^
	// But be careful not to affect:
	// - Negative numbers at start or after operator
	// - Currency symbols like $
	// - Decimal points
	// - CIDR notation like /24
	// - Time notation like 6:00

	// First, normalize multiple spaces to single space
	spaceRe := regexp.MustCompile(`\s+`)
	result = spaceRe.ReplaceAllString(result, " ")

	// Add spaces around operators (but not inside numbers or special notations)
	// Match operator not preceded/followed by space
	operators := []struct {
		pattern string
		replace string
	}{
		// Multiplication variants
		{`(\S)\s*×\s*(\S)`, `$1 × $2`},
		{`(\S)\s*÷\s*(\S)`, `$1 ÷ $2`},
		{`([^0])\s*x\s*(\d)`, `$1 x $2`}, // x as multiplication, but not after 0 (hex notation 0x)
		{`(\d)\s*\*\s*(\d)`, `$1 * $2`},  // * between digits
		{`(\d)\s*\^\s*(\d)`, `$1 ^ $2`},  // ^ between digits
		// Addition - digit/paren/percent followed by +
		{`([\d\)%])\s*\+\s*(\S)`, `$1 + $2`},
		// Subtraction - digit/paren/percent followed by -
		{`([\d\)%])\s*-\s*(\S)`, `$1 - $2`},
		// Division - but not CIDR notation (/24)
		{`(\d)\s*/\s*(\d{3,})`, `$1 / $2`}, // Only if divisor is 3+ digits (not CIDR)
	}

	for _, op := range operators {
		re := regexp.MustCompile(op.pattern)
		result = re.ReplaceAllString(result, op.replace)
	}

	return result
}

// LineResult holds the result of evaluating a single line.
type LineResult struct {
	Output      string
	Value       float64
	HasResult   bool
	IsCurrency  bool
	IsDateTime  bool
	DateTimeStr string // raw datetime result for reference
}

// cleanOutputLines removes stale output lines ("> " prefixed) that follow expression lines.
// This ensures old multi-line output is cleared before new evaluation.
func cleanOutputLines(lines []string) []string {
	var result []string
	for _, line := range lines {
		// Skip output lines (start with ">") - they will be regenerated
		if strings.HasPrefix(line, ">") {
			continue
		}
		result = append(result, line)
	}
	return result
}

// EvalLines evaluates all lines and returns the processed output lines.
// activeLineNum is 1-based line number of the line currently being edited.
// When activeLineNum > 0, only that line and its dependents are re-evaluated.
// Pass 0 or negative to evaluate all lines (used for initial load).
func EvalLines(lines []string, activeLineNum int) []LineResult {
	// Build a map of expression lines that have multi-line output (lines starting with ">")
	// This is used to preserve existing multi-line output for lines that aren't re-evaluated
	hasMultiLineOutput := make(map[int][]string) // maps cleaned line index to its output lines
	cleanedIdx := 0
	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], ">") {
			continue // Skip output lines in this pass
		}
		// Check if next lines are output lines
		var outputLines []string
		for j := i + 1; j < len(lines) && strings.HasPrefix(lines[j], ">"); j++ {
			outputLines = append(outputLines, lines[j])
		}
		if len(outputLines) > 0 {
			hasMultiLineOutput[cleanedIdx] = outputLines
		}
		cleanedIdx++
	}

	// First pass: remove stale output lines ("> " lines that follow an expression)
	cleanedLines := cleanOutputLines(lines)

	// Determine which lines need evaluation
	// If activeLineNum > 0, only evaluate that line and its dependents
	linesToEvaluate := make(map[int]bool)
	if activeLineNum > 0 {
		linesToEvaluate[activeLineNum] = true
		dependents := FindDependentLines(cleanedLines, activeLineNum)
		for _, dep := range dependents {
			linesToEvaluate[dep] = true
		}
	}

	results := make([]LineResult, len(cleanedLines))
	values := make([]float64, len(cleanedLines))
	haveRes := make([]bool, len(cleanedLines))
	currencyByLine := make([]bool, len(cleanedLines))

	// Helper to conditionally format expression (skip formatting for active line)
	maybeFormat := func(lineIdx int, expr string) string {
		// lineIdx is 0-based, activeLineNum is 1-based
		if activeLineNum > 0 && lineIdx+1 == activeLineNum {
			return expr // Skip formatting for active line
		}
		return formatExpression(expr)
	}

	for i, line := range cleanedLines {
		results[i].Output = line
		lineNum := i + 1 // 1-based line number

		// Skip empty lines
		if line == "" {
			continue
		}
		// Skip comment lines (starting with #, allowing leading whitespace)
		// But don't skip hex color expressions like "#FF5733 to rgb"
		trimmedLine := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmedLine, "#") {
			// Check if this looks like a hex color expression
			hexColorPattern := regexp.MustCompile(`^#[0-9a-fA-F]{3,6}\s+(?:to|in)\s+`)
			if !hexColorPattern.MatchString(trimmedLine) {
				continue
			}
		}

		// Handle inline comments - strip everything after #
		// But don't treat hex colors (#FF5733) as comments
		workingLine := line
		inlineComment := ""
		if hashIdx := strings.Index(line, "#"); hashIdx >= 0 {
			// Check if this looks like a hex color (# followed by hex digits)
			isHexColor := false
			if hashIdx < len(line)-1 {
				rest := line[hashIdx+1:]
				// Check if it starts with hex digits (color code)
				hexPattern := regexp.MustCompile(`^[0-9a-fA-F]{3,6}(?:\s|$)`)
				if hexPattern.MatchString(rest) {
					isHexColor = true
				}
			}
			if !isHexColor {
				workingLine = line[:hashIdx]
			}
		}

		eq := findResultEquals(workingLine)
		if eq < 0 {
			continue
		}
		expr := strings.TrimSpace(workingLine[:eq])
		if expr == "" {
			continue
		}

		// Skip evaluation for lines that don't need it (not active line or dependent)
		// Preserve existing results for these lines
		if activeLineNum > 0 && !linesToEvaluate[lineNum] {
			// Preserve existing multi-line output if present
			if outputLines, ok := hasMultiLineOutput[i]; ok {
				results[i].Output = line + "\n" + strings.Join(outputLines, "\n")
			}
			// Keep the line as-is (with its existing result)
			continue
		}

		// Extract inline comment from original line (after the = sign)
		inlineComment = extractInlineComment(line, eq)

		// Try base conversion first (24 in hex, 0xFF in dec, etc.)
		if isBaseConversionExpr(expr) {
			if baseResult, ok := tryBaseConversion(expr); ok {
				results[i].Output = expr + " = " + baseResult + inlineComment
				results[i].HasResult = true
				continue
			}
		}

		// Try physical constants
		if constants.IsConstantExpression(expr) {
			constResult, err := constants.EvalConstants(expr)
			if err == nil {
				results[i].Output = maybeFormat(i, expr) + " = " + constResult + inlineComment
				results[i].HasResult = true
				continue
			}
		}

		// Try unit conversions
		if units.IsUnitExpression(expr) {
			unitResult, err := units.EvalUnits(expr)
			if err == nil {
				results[i].Output = maybeFormat(i, expr) + " = " + unitResult + inlineComment
				results[i].HasResult = true
				continue
			}
		}

		// Try radio/electrical calculations
		if radio.IsRadioExpression(expr) {
			radioResult, err := radio.EvalRadio(expr)
			if err == nil {
				// Multi-line results start with \n>, single-line results don't
				if strings.HasPrefix(radioResult, "\n>") {
					results[i].Output = maybeFormat(i, expr) + " =" + radioResult + inlineComment
				} else {
					results[i].Output = maybeFormat(i, expr) + " = " + radioResult + inlineComment
				}
				results[i].HasResult = true
				continue
			}
		}

		// Try percentage calculations
		if percentage.IsPercentageExpression(expr) {
			pctResult, err := percentage.EvalPercentage(expr)
			if err == nil {
				results[i].Output = maybeFormat(i, expr) + " = " + pctResult + inlineComment
				results[i].HasResult = true
				continue
			}
		}

		// Try financial calculations
		if finance.IsFinanceExpression(expr) {
			finResult, err := finance.EvalFinance(expr)
			if err == nil {
				results[i].Output = maybeFormat(i, expr) + " = " + finResult + inlineComment
				results[i].HasResult = true
				continue
			}
		}

		// Try statistics functions
		if stats.IsStatsExpression(expr) {
			statsResult, err := stats.EvalStats(expr)
			if err == nil {
				results[i].Output = maybeFormat(i, expr) + " = " + statsResult + inlineComment
				results[i].HasResult = true
				continue
			}
		}

		// Try programmer utilities
		if programmer.IsProgrammerExpression(expr) {
			progResult, err := programmer.EvalProgrammer(expr)
			if err == nil {
				results[i].Output = maybeFormat(i, expr) + " = " + progResult + inlineComment
				results[i].HasResult = true
				continue
			}
		}

		// Try regex testing
		if regex.IsRegexExpression(expr) {
			regexResult, err := regex.EvalRegex(expr)
			if err == nil {
				results[i].Output = maybeFormat(i, expr) + " =" + regexResult + inlineComment
				results[i].HasResult = true
				continue
			}
		}

		// Try Unix permissions evaluation
		if permissions.IsPermissionsExpression(expr) {
			permResult, err := permissions.EvalPermissions(expr)
			if err == nil {
				results[i].Output = maybeFormat(i, expr) + " = " + permResult + inlineComment
				results[i].HasResult = true
				continue
			}
		}

		// Try crontab explanation
		if crontab.IsCrontabExpression(expr) {
			cronResult, err := crontab.EvalCrontab(expr)
			if err == nil {
				results[i].Output = maybeFormat(i, expr) + " = " + cronResult + inlineComment
				results[i].HasResult = true
				continue
			}
		}

		// Try HTTP status code explanation
		if httpstatus.IsHTTPStatusExpression(expr) {
			httpResult, err := httpstatus.EvalHTTPStatus(expr)
			if err == nil {
				results[i].Output = maybeFormat(i, expr) + " = " + httpResult + inlineComment
				results[i].HasResult = true
				continue
			}
		}

		// Try cooking conversions
		if cooking.IsCookingExpression(expr) {
			cookResult, err := cooking.EvalCooking(expr)
			if err == nil {
				results[i].Output = maybeFormat(i, expr) + " = " + cookResult + inlineComment
				results[i].HasResult = true
				continue
			}
		}

		// Try man-hour calculations
		if manhour.IsManHourExpression(expr) {
			mhResult, err := manhour.EvalManHour(expr)
			if err == nil {
				results[i].Output = maybeFormat(i, expr) + " = " + mhResult + inlineComment
				results[i].HasResult = true
				continue
			}
		}

		// Try hourly cost calculations
		if hourlycost.IsHourlyCostExpression(expr) {
			hcResult, err := hourlycost.EvalHourlyCost(expr)
			if err == nil {
				results[i].Output = maybeFormat(i, expr) + " = " + hcResult + inlineComment
				results[i].HasResult = true
				continue
			}
		}

		// Try JWT decoding
		// Note: Don't use maybeFormat for JWT expressions as it corrupts base64url tokens
		if jwt.IsJWTExpression(expr) {
			jwtResult, err := jwt.EvalJWT(expr)
			if err == nil {
				results[i].Output = expr + " =\n> " + jwtResult + inlineComment
				results[i].HasResult = true
				continue
			}
		}

		// Try SSL certificate decoding
		// Note: Don't use maybeFormat for cert expressions as URLs should not be modified
		// Skip re-evaluation if line already has a result and is not the active line (expensive network operation)
		if cert.IsCertExpression(expr) {
			isActiveLine := activeLineNum > 0 && i+1 == activeLineNum

			// Check if line already has an inline result (like "ERR: ..." after =)
			existingResult := strings.TrimSpace(workingLine[eq+1:])
			if existingResult != "" && !isActiveLine {
				// Line already has an inline result, keep it as-is
				results[i].Output = line
				results[i].HasResult = true
				continue
			}

			// Check if line had multi-line output (successful cert decode)
			if outputLines, ok := hasMultiLineOutput[i]; ok && !isActiveLine {
				// Reconstruct the output with the existing multi-line result
				results[i].Output = line + "\n" + strings.Join(outputLines, "\n")
				results[i].HasResult = true
				continue
			}

			certResult, err := cert.EvalCert(expr)
			if err == nil {
				results[i].Output = expr + " =\n> " + certResult + inlineComment
				results[i].HasResult = true
				continue
			} else {
				// Show the error message for cert decode failures
				results[i].Output = expr + " = ERR: " + err.Error() + inlineComment
				results[i].HasResult = true
				continue
			}
		}

		// Try DNS lookup
		// Note: Don't use maybeFormat for DNS expressions as domain names should not be modified
		// Skip re-evaluation if line already has a result and is not the active line (expensive network operation)
		if network.IsDNSExpression(expr) {
			isActiveLine := activeLineNum > 0 && i+1 == activeLineNum

			// Check if line already has an inline result
			existingResult := strings.TrimSpace(workingLine[eq+1:])
			if existingResult != "" && !isActiveLine {
				results[i].Output = line
				results[i].HasResult = true
				continue
			}

			// Check if line had multi-line output
			if outputLines, ok := hasMultiLineOutput[i]; ok && !isActiveLine {
				results[i].Output = line + "\n" + strings.Join(outputLines, "\n")
				results[i].HasResult = true
				continue
			}

			dnsResult, err := network.EvalDNS(expr)
			if err == nil {
				results[i].Output = expr + " =\n" + dnsResult + inlineComment
				results[i].HasResult = true
				continue
			} else {
				results[i].Output = expr + " = ERR: " + err.Error() + inlineComment
				results[i].HasResult = true
				continue
			}
		}

		// Try WHOIS lookup
		// Note: Don't use maybeFormat for WHOIS expressions as domain names should not be modified
		// Skip re-evaluation if line already has a result and is not the active line (expensive network operation)
		if network.IsWhoisExpression(expr) {
			isActiveLine := activeLineNum > 0 && i+1 == activeLineNum

			// Check if line already has an inline result
			existingResult := strings.TrimSpace(workingLine[eq+1:])
			if existingResult != "" && !isActiveLine {
				results[i].Output = line
				results[i].HasResult = true
				continue
			}

			// Check if line had multi-line output
			if outputLines, ok := hasMultiLineOutput[i]; ok && !isActiveLine {
				results[i].Output = line + "\n" + strings.Join(outputLines, "\n")
				results[i].HasResult = true
				continue
			}

			whoisResult, err := network.EvalWhois(expr)
			if err == nil {
				results[i].Output = expr + " =\n" + whoisResult + inlineComment
				results[i].HasResult = true
				continue
			} else {
				results[i].Output = expr + " = ERR: " + err.Error() + inlineComment
				results[i].HasResult = true
				continue
			}
		}

		// Try network/IP evaluation
		if network.IsNetworkExpression(expr) {
			netResult, err := network.EvalNetwork(expr)
			if err == nil {
				results[i].Output = maybeFormat(i, expr) + " = " + netResult + inlineComment
				results[i].HasResult = true
				continue
			}
			// Fall through if network eval fails
		}

		// Try GeoIP lookup
		if network.IsGeoIPExpression(expr) {
			geoResult, err := network.EvalGeoIP(expr)
			if err == nil {
				results[i].Output = maybeFormat(i, expr) + " = " + geoResult + inlineComment
				results[i].HasResult = true
				continue
			}
			// Fall through if geoip eval fails
		}

		// Try "what is my ip" lookup
		if network.IsMyIPExpression(expr) {
			myIPResult, err := network.EvalMyIP()
			if err == nil {
				results[i].Output = maybeFormat(i, expr) + " =" + myIPResult + inlineComment
				results[i].HasResult = true
				continue
			}
			// Fall through if my ip eval fails
		}

		// Try color conversion
		if color.IsColorExpression(expr) {
			colorResult, err := color.EvalColor(expr)
			if err == nil {
				results[i].Output = maybeFormat(i, expr) + " = " + colorResult + inlineComment
				results[i].HasResult = true
				continue
			}
			// Fall through if color eval fails
		}

		// Try date/time evaluation (with reference support)
		if datetime.IsDateTimeExpression(expr) || strings.Contains(expr, "\\") {
			// Create resolver for line references
			resolver := func(n int) (string, bool) {
				idx := n - 1
				if idx < 0 || idx >= len(results) {
					return "", false
				}
				if results[idx].IsDateTime && results[idx].DateTimeStr != "" {
					return results[idx].DateTimeStr, true
				}
				return "", false
			}

			dtResult, err := datetime.EvalDateTimeWithRefs(expr, resolver)
			if err == nil {
				results[i].Output = maybeFormat(i, expr) + " = " + dtResult + inlineComment
				results[i].HasResult = true
				results[i].IsDateTime = true
				results[i].DateTimeStr = dtResult
				continue
			}
			// Fall through to numeric evaluation if datetime fails
		}

		isCurrency := strings.Contains(expr, "$") || eval.ExprReferencesCurrency(expr, currencyByLine)
		isComparison := isComparisonExpr(expr)

		val, err := eval.EvalExpr(expr, func(n int) (float64, error) {
			idx := n - 1
			if idx < 0 || idx >= len(values) {
				return 0, fmt.Errorf("bad reference \\\\%d", n)
			}
			if !haveRes[idx] {
				return 0, fmt.Errorf("unresolved reference \\\\%d", n)
			}
			return values[idx], nil
		})
		if err != nil {
			results[i].Output = maybeFormat(i, expr) + " = ERR" + inlineComment
			continue
		}

		values[i] = val
		haveRes[i] = true
		currencyByLine[i] = isCurrency

		var resultStr string
		if isComparison {
			resultStr = utils.FormatBoolResult(val)
		} else {
			resultStr = utils.FormatResult(isCurrency, val)
		}
		results[i].Output = maybeFormat(i, expr) + " = " + resultStr + inlineComment
		results[i].Value = val
		results[i].HasResult = true
		results[i].IsCurrency = isCurrency
	}

	return results
}

// BuildLineNumbers generates line number text for n lines.
func BuildLineNumbers(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		b.WriteString(fmt.Sprintf("%d\n", i))
	}
	return strings.TrimRight(b.String(), "\n")
}

// GetLineValues returns a map of line number (1-based) to formatted result string.
// This is used for replacing references with actual values when copying.
func GetLineValues(lines []string) map[int]string {
	results := EvalLines(lines, 0) // Format all lines when copying
	values := make(map[int]string)
	for i, r := range results {
		if r.HasResult {
			values[i+1] = utils.FormatResult(r.IsCurrency, r.Value)
		}
	}
	return values
}

// ReplaceRefsWithValues takes text and replaces all \n references with actual values.
func ReplaceRefsWithValues(text string) string {
	lines := strings.Split(text, "\n")
	values := GetLineValues(lines)
	return eval.ReplaceReferencesWithValues(text, values)
}

// FindDependentLines returns a list of line numbers (1-based) that reference the given line.
// It recursively finds all lines that depend on the changed line.
func FindDependentLines(lines []string, changedLine int) []int {
	dependents := make(map[int]bool)
	findDependentsRecursive(lines, changedLine, dependents)

	// Convert map to sorted slice
	result := make([]int, 0, len(dependents))
	for lineNum := range dependents {
		result = append(result, lineNum)
	}
	// Sort to ensure consistent order
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i] > result[j] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

// findDependentsRecursive finds all lines that reference targetLine and adds them to dependents map
func findDependentsRecursive(lines []string, targetLine int, dependents map[int]bool) {
	refPattern := regexp.MustCompile(`\\(\d+)`)

	for i, line := range lines {
		lineNum := i + 1 // 1-based
		if dependents[lineNum] {
			continue // Already processed
		}

		// Find all references in this line
		matches := refPattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			refNum, _ := strconv.Atoi(match[1])
			if refNum == targetLine {
				// This line references the target line
				dependents[lineNum] = true
				// Recursively find lines that depend on this line
				findDependentsRecursive(lines, lineNum, dependents)
				break
			}
		}
	}
}

// StripResult removes the result from a line, keeping the expression, '=' sign, and any inline comment.
// Example: "2 + 3 = 5 # my note" -> "2 + 3 = # my note"
// Example: "2 + 3 = 5" -> "2 + 3 ="
func StripResult(line string) string {
	eq := findResultEquals(line)
	if eq < 0 {
		return line // No '=' found, return as-is
	}

	// Get the part before and including '='
	beforeEq := line[:eq+1]

	// Check for inline comment after '='
	afterEq := line[eq+1:]
	hashIdx := strings.Index(afterEq, "#")
	if hashIdx >= 0 {
		// Has inline comment - keep it
		return beforeEq + " " + strings.TrimLeft(afterEq[hashIdx:], " ")
	}

	// No inline comment
	return beforeEq
}

// HasResult checks if a line has a result (something after '=' that's not just whitespace or comment)
func HasResult(line string) bool {
	eq := findResultEquals(line)
	if eq < 0 {
		return false
	}

	afterEq := strings.TrimSpace(line[eq+1:])
	// If it starts with # or is empty, no result
	if afterEq == "" || strings.HasPrefix(afterEq, "#") {
		return false
	}
	return true
}

// EvalSingleLine evaluates a single line given the context of all lines.
// Returns the evaluated line and whether it has a result.
func EvalSingleLine(lines []string, lineNum int) LineResult {
	results := EvalLines(lines, 0)
	if lineNum < 1 || lineNum > len(results) {
		return LineResult{}
	}
	return results[lineNum-1]
}

// EvalLinesSelective evaluates only the specified lines (1-based line numbers).
// Returns results for ALL lines, but only the specified ones are recalculated.
func EvalLinesSelective(lines []string, linesToEval []int) []LineResult {
	// For now, we still need to evaluate all lines to resolve dependencies correctly
	// But we can optimize by only formatting the specified lines
	return EvalLines(lines, 0)
}

// StripAllResults strips results from all expression lines and removes output lines (starting with '>').
// This prepares the document for a full recalculation.
func StripAllResults(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	for _, line := range lines {
		// Skip output lines
		if strings.HasPrefix(line, ">") {
			continue
		}
		// Strip result from expression lines
		if HasResult(line) {
			result = append(result, StripResult(line))
		} else {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

// StripAndEvalReferencingLines strips results from all lines that contain references
// and re-evaluates them. Returns the updated text.
func StripAndEvalReferencingLines(text string) string {
	lines := strings.Split(text, "\n")
	refPattern := regexp.MustCompile(`\\(\d+)`)

	// Find all lines with references and strip their results
	for i, line := range lines {
		if refPattern.MatchString(line) && HasResult(line) {
			lines[i] = StripResult(line)
		}
	}

	// Re-evaluate all lines
	results := EvalLines(lines, 0)

	// Build output
	output := make([]string, len(results))
	for i, r := range results {
		output[i] = r.Output
	}
	return strings.Join(output, "\n")
}
