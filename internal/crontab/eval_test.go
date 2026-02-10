package crontab

import (
	"strings"
	"testing"
)

func TestIsCrontabExpression(t *testing.T) {
	tests := []struct {
		expr     string
		expected bool
	}{
		// Valid crontab expressions
		{"cron */5 * * * *", true},
		{"cron 0 0 * * *", true},
		{"cron 30 2 * * 1-5", true},
		{"crontab 0 0 1 1 *", true},
		{"crontab */15 * * * *", true},
		{"CRON 0 0 * * *", true},
		{"CRONTAB 0 0 * * *", true},
		{"cron @daily", true},
		{"cron @hourly", true},
		{"crontab @weekly", true},

		// Not crontab expressions
		{"0 0 * * *", false},
		{"*/5 * * * *", false},
		{"hello world", false},
		{"2 + 2", false},
		{"cron", false},
		{"chmod 755", false},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result := IsCrontabExpression(tt.expr)
			if result != tt.expected {
				t.Errorf("IsCrontabExpression(%q) = %v, want %v", tt.expr, result, tt.expected)
			}
		})
	}
}

func TestEvalCrontabAliases(t *testing.T) {
	tests := []struct {
		expr     string
		contains []string
	}{
		{"cron @yearly", []string{"At 00:00", "day 1", "January", "0 0 1 1 *"}},
		{"cron @annually", []string{"At 00:00", "day 1", "January", "0 0 1 1 *"}},
		{"cron @monthly", []string{"At 00:00", "day 1", "0 0 1 * *"}},
		{"cron @weekly", []string{"At 00:00", "Sunday", "0 0 * * 0"}},
		{"cron @daily", []string{"At 00:00", "0 0 * * *"}},
		{"cron @midnight", []string{"At 00:00", "0 0 * * *"}},
		{"cron @hourly", []string{"minute 0", "0 * * * *"}},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvalCrontab(tt.expr)
			if err != nil {
				t.Errorf("EvalCrontab(%q) returned error: %v", tt.expr, err)
				return
			}
			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("EvalCrontab(%q) = %q, want to contain %q", tt.expr, result, s)
				}
			}
		})
	}
}

func TestEvalCrontabStandard(t *testing.T) {
	tests := []struct {
		expr     string
		contains []string
	}{
		// Every 5 minutes
		{"cron */5 * * * *", []string{"Every 5 minutes"}},
		// Every 15 minutes
		{"cron */15 * * * *", []string{"Every 15 minutes"}},
		// Every minute
		{"cron * * * * *", []string{"Every minute"}},
		// At midnight
		{"cron 0 0 * * *", []string{"At 00:00"}},
		// At 2:30 AM
		{"cron 30 2 * * *", []string{"At 02:30"}},
		// At 14:00 (2 PM)
		{"cron 0 14 * * *", []string{"At 14:00"}},
		// At 2:30 AM on weekdays
		{"cron 30 2 * * 1-5", []string{"At 02:30", "Monday through Friday"}},
		// At midnight on the 1st of every month
		{"cron 0 0 1 * *", []string{"At 00:00", "day 1"}},
		// At midnight on Jan 1st
		{"cron 0 0 1 1 *", []string{"At 00:00", "day 1", "January"}},
		// Every hour
		{"cron 0 * * * *", []string{"minute 0", "every hour"}},
		// Every 2 hours
		{"cron 0 */2 * * *", []string{"Every 2 hours"}},
		// On Sundays
		{"cron 0 0 * * 0", []string{"At 00:00", "Sunday"}},
		// On Monday and Friday
		{"cron 0 9 * * 1,5", []string{"At 09:00", "Monday", "Friday"}},
		// In June
		{"cron 0 0 * 6 *", []string{"At 00:00", "June"}},
		// Multiple months
		{"cron 0 0 * 1,6 *", []string{"At 00:00", "January", "June"}},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvalCrontab(tt.expr)
			if err != nil {
				t.Errorf("EvalCrontab(%q) returned error: %v", tt.expr, err)
				return
			}
			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("EvalCrontab(%q) = %q, want to contain %q", tt.expr, result, s)
				}
			}
		})
	}
}

func TestEvalCrontabWithCrontabPrefix(t *testing.T) {
	result, err := EvalCrontab("crontab */5 * * * *")
	if err != nil {
		t.Errorf("EvalCrontab with 'crontab' prefix returned error: %v", err)
		return
	}
	if !strings.Contains(result, "Every 5 minutes") {
		t.Errorf("EvalCrontab('crontab */5 * * * *') = %q, want to contain 'Every 5 minutes'", result)
	}
}

func TestEvalCrontabInvalid(t *testing.T) {
	tests := []string{
		"cron 60 * * * *",    // minute out of range
		"cron * 24 * * *",    // hour out of range
		"cron * * 32 * *",    // day out of range
		"cron * * * 13 *",    // month out of range
		"cron * * * * 8",     // dow out of range
		"cron * * *",         // too few fields
		"cron * * * * * *",   // too many fields
		"cron abc * * * *",   // invalid value
	}

	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			_, err := EvalCrontab(expr)
			if err == nil {
				t.Errorf("EvalCrontab(%q) should return error", expr)
			}
		})
	}
}

func TestValidateField(t *testing.T) {
	tests := []struct {
		field string
		name  string
		min   int
		max   int
		valid bool
	}{
		{"*", "minute", 0, 59, true},
		{"*/5", "minute", 0, 59, true},
		{"0", "minute", 0, 59, true},
		{"59", "minute", 0, 59, true},
		{"60", "minute", 0, 59, false},
		{"0-30", "minute", 0, 59, true},
		{"0-59/5", "minute", 0, 59, true},
		{"1,15,30", "minute", 0, 59, true},
		{"0-23", "hour", 0, 23, true},
		{"24", "hour", 0, 23, false},
		{"1-31", "day of month", 1, 31, true},
		{"0", "day of month", 1, 31, false},
		{"1-12", "month", 1, 12, true},
		{"0-7", "day of week", 0, 7, true},
	}

	for _, tt := range tests {
		t.Run(tt.field+"_"+tt.name, func(t *testing.T) {
			err := validateField(tt.field, tt.name, tt.min, tt.max)
			if tt.valid && err != nil {
				t.Errorf("validateField(%q, %q, %d, %d) returned unexpected error: %v", tt.field, tt.name, tt.min, tt.max, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("validateField(%q, %q, %d, %d) should return error", tt.field, tt.name, tt.min, tt.max)
			}
		})
	}
}
