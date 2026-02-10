package crontab

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Short cron aliases mapped to their 5-field equivalents
var cronAliases = map[string]string{
	"@yearly":   "0 0 1 1 *",
	"@annually": "0 0 1 1 *",
	"@monthly":  "0 0 1 * *",
	"@weekly":   "0 0 * * 0",
	"@daily":    "0 0 * * *",
	"@midnight": "0 0 * * *",
	"@hourly":   "0 * * * *",
}

// Day of week names
var dowNames = map[int]string{
	0: "Sunday", 1: "Monday", 2: "Tuesday", 3: "Wednesday",
	4: "Thursday", 5: "Friday", 6: "Saturday", 7: "Sunday",
}

// Month names
var monthNames = map[int]string{
	1: "January", 2: "February", 3: "March", 4: "April",
	5: "May", 6: "June", 7: "July", 8: "August",
	9: "September", 10: "October", 11: "November", 12: "December",
}

// IsCrontabExpression checks if an expression looks like a crontab expression
func IsCrontabExpression(expr string) bool {
	exprLower := strings.ToLower(strings.TrimSpace(expr))

	// Check for "cron" or "crontab" prefix
	if strings.HasPrefix(exprLower, "cron ") || strings.HasPrefix(exprLower, "crontab ") {
		return true
	}

	return false
}

// EvalCrontab evaluates a crontab expression and returns a human-readable explanation
func EvalCrontab(expr string) (string, error) {
	expr = strings.TrimSpace(expr)

	// Strip "cron " or "crontab " prefix (case-insensitive)
	cronExpr := expr
	exprLower := strings.ToLower(expr)
	if strings.HasPrefix(exprLower, "crontab ") {
		cronExpr = strings.TrimSpace(expr[8:])
	} else if strings.HasPrefix(exprLower, "cron ") {
		cronExpr = strings.TrimSpace(expr[5:])
	}

	// Check for short aliases
	aliasLower := strings.ToLower(cronExpr)
	if expanded, ok := cronAliases[aliasLower]; ok {
		desc, err := describeCron(expanded)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s (%s)", desc, expanded), nil
	}

	// Parse standard 5-field cron expression
	fields := strings.Fields(cronExpr)
	if len(fields) != 5 {
		return "", fmt.Errorf("invalid cron expression: expected 5 fields (minute hour day month weekday), got %d", len(fields))
	}

	// Validate fields
	if err := validateCronFields(fields); err != nil {
		return "", err
	}

	return describeCron(cronExpr)
}

// validateCronFields validates the 5 cron fields
func validateCronFields(fields []string) error {
	type fieldSpec struct {
		name string
		min  int
		max  int
	}
	specs := []fieldSpec{
		{"minute", 0, 59},
		{"hour", 0, 23},
		{"day of month", 1, 31},
		{"month", 1, 12},
		{"day of week", 0, 7},
	}

	for i, field := range fields {
		if err := validateField(field, specs[i].name, specs[i].min, specs[i].max); err != nil {
			return err
		}
	}
	return nil
}

// validateField validates a single cron field
func validateField(field, name string, min, max int) error {
	// Handle comma-separated values
	parts := strings.Split(field, ",")
	for _, part := range parts {
		if err := validateFieldPart(part, name, min, max); err != nil {
			return err
		}
	}
	return nil
}

// validateFieldPart validates a single part of a cron field (handles *, ranges, steps)
func validateFieldPart(part, name string, min, max int) error {
	// Wildcard
	if part == "*" {
		return nil
	}

	// Step values: */5 or 1-30/5
	if strings.Contains(part, "/") {
		stepParts := strings.SplitN(part, "/", 2)
		if stepParts[0] != "*" {
			if err := validateFieldPart(stepParts[0], name, min, max); err != nil {
				return err
			}
		}
		step, err := strconv.Atoi(stepParts[1])
		if err != nil || step < 1 {
			return fmt.Errorf("invalid step value in %s: %s", name, part)
		}
		return nil
	}

	// Range: 1-5
	if strings.Contains(part, "-") {
		rangeParts := strings.SplitN(part, "-", 2)
		lo, err1 := strconv.Atoi(rangeParts[0])
		hi, err2 := strconv.Atoi(rangeParts[1])
		if err1 != nil || err2 != nil {
			return fmt.Errorf("invalid range in %s: %s", name, part)
		}
		if lo < min || lo > max || hi < min || hi > max {
			return fmt.Errorf("value out of range in %s: %s (allowed %d-%d)", name, part, min, max)
		}
		return nil
	}

	// Single value
	val, err := strconv.Atoi(part)
	if err != nil {
		return fmt.Errorf("invalid value in %s: %s", name, part)
	}
	if val < min || val > max {
		return fmt.Errorf("value out of range in %s: %d (allowed %d-%d)", name, val, min, max)
	}
	return nil
}

// describeCron generates a human-readable description of a 5-field cron expression
func describeCron(cronExpr string) (string, error) {
	fields := strings.Fields(cronExpr)
	if len(fields) != 5 {
		return "", fmt.Errorf("invalid cron expression")
	}

	minute := fields[0]
	hour := fields[1]
	dom := fields[2]
	month := fields[3]
	dow := fields[4]

	var parts []string

	// Describe time
	timePart := describeTime(minute, hour)
	parts = append(parts, timePart)

	// Describe day of month
	if dom != "*" {
		parts = append(parts, "on "+describeDayOfMonth(dom))
	}

	// Describe month
	if month != "*" {
		parts = append(parts, "in "+describeMonth(month))
	}

	// Describe day of week
	if dow != "*" {
		parts = append(parts, "on "+describeDayOfWeek(dow))
	}

	return strings.Join(parts, ", "), nil
}

// describeTime describes the minute and hour fields
func describeTime(minute, hour string) string {
	// Every minute
	if minute == "*" && hour == "*" {
		return "Every minute"
	}

	// Every N minutes
	if strings.HasPrefix(minute, "*/") && hour == "*" {
		step := minute[2:]
		return fmt.Sprintf("Every %s minutes", step)
	}

	// Every hour at specific minute
	if hour == "*" {
		return fmt.Sprintf("At minute %s of every hour", describeField(minute))
	}

	// Every N hours
	if minute == "0" && strings.HasPrefix(hour, "*/") {
		step := hour[2:]
		return fmt.Sprintf("Every %s hours", step)
	}

	// Specific time
	if !strings.ContainsAny(minute, "*/,-") && !strings.ContainsAny(hour, "*/,-") {
		h, _ := strconv.Atoi(hour)
		m, _ := strconv.Atoi(minute)
		return fmt.Sprintf("At %02d:%02d", h, m)
	}

	// Hour range or list with specific minute
	if !strings.ContainsAny(minute, "*/,-") {
		return fmt.Sprintf("At minute %s, during hour %s", minute, describeField(hour))
	}

	// Complex time
	return fmt.Sprintf("At minute %s, during hour %s", describeField(minute), describeField(hour))
}

// describeField describes a generic cron field value
func describeField(field string) string {
	if field == "*" {
		return "every"
	}

	// Step
	if strings.HasPrefix(field, "*/") {
		return "every " + field[2:]
	}

	// Range with step
	if strings.Contains(field, "/") {
		stepParts := strings.SplitN(field, "/", 2)
		return fmt.Sprintf("every %s from %s", stepParts[1], stepParts[0])
	}

	// Range
	if strings.Contains(field, "-") && !strings.Contains(field, ",") {
		return field
	}

	// List
	if strings.Contains(field, ",") {
		return field
	}

	return field
}

// describeDayOfMonth describes the day-of-month field
func describeDayOfMonth(dom string) string {
	if strings.Contains(dom, ",") {
		return "days " + dom
	}
	if strings.Contains(dom, "-") {
		return "days " + dom
	}
	if strings.Contains(dom, "/") {
		if strings.HasPrefix(dom, "*/") {
			return fmt.Sprintf("every %s days", dom[2:])
		}
		parts := strings.SplitN(dom, "/", 2)
		return fmt.Sprintf("every %s days starting on day %s", parts[1], parts[0])
	}
	return "day " + dom
}

// describeMonth describes the month field
func describeMonth(month string) string {
	if strings.Contains(month, ",") {
		months := strings.Split(month, ",")
		names := make([]string, len(months))
		for i, m := range months {
			if n, err := strconv.Atoi(m); err == nil {
				if name, ok := monthNames[n]; ok {
					names[i] = name
					continue
				}
			}
			names[i] = m
		}
		return strings.Join(names, ", ")
	}
	if strings.Contains(month, "-") {
		rangeParts := strings.SplitN(month, "-", 2)
		lo, err1 := strconv.Atoi(rangeParts[0])
		hi, err2 := strconv.Atoi(rangeParts[1])
		if err1 == nil && err2 == nil {
			loName, ok1 := monthNames[lo]
			hiName, ok2 := monthNames[hi]
			if ok1 && ok2 {
				return loName + " through " + hiName
			}
		}
		return month
	}
	if n, err := strconv.Atoi(month); err == nil {
		if name, ok := monthNames[n]; ok {
			return name
		}
	}
	return month
}

// describeDayOfWeek describes the day-of-week field
func describeDayOfWeek(dow string) string {
	if strings.Contains(dow, ",") {
		days := strings.Split(dow, ",")
		names := make([]string, len(days))
		for i, d := range days {
			if n, err := strconv.Atoi(d); err == nil {
				if name, ok := dowNames[n]; ok {
					names[i] = name
					continue
				}
			}
			names[i] = d
		}
		return strings.Join(names, ", ")
	}
	if strings.Contains(dow, "-") {
		rangeParts := strings.SplitN(dow, "-", 2)
		lo, err1 := strconv.Atoi(rangeParts[0])
		hi, err2 := strconv.Atoi(rangeParts[1])
		if err1 == nil && err2 == nil {
			loName, ok1 := dowNames[lo]
			hiName, ok2 := dowNames[hi]
			if ok1 && ok2 {
				return loName + " through " + hiName
			}
		}
		return dow
	}
	if strings.HasPrefix(dow, "*/") {
		return fmt.Sprintf("every %s days of the week", dow[2:])
	}
	if n, err := strconv.Atoi(dow); err == nil {
		if name, ok := dowNames[n]; ok {
			return name
		}
	}
	return dow
}

// cronFieldPattern matches valid cron field characters
var cronFieldPattern = regexp.MustCompile(`^[\d\*\/\,\-]+$`)
