package data

import (
	"strings"
	"testing"

	"smartcalc/internal/calc"
)

// TestSnippetsNoErrors verifies that all snippets evaluate without errors.
// Each snippet expression should produce a valid result, not "ERR".
func TestSnippetsNoErrors(t *testing.T) {
	categories := GetSnippetCategories()

	for _, category := range categories {
		for _, snippet := range category.Snippets {
			t.Run(category.Name+"/"+snippet.Name, func(t *testing.T) {
				// Split snippet content into lines
				lines := strings.Split(strings.TrimSuffix(snippet.Content, "\n"), "\n")

				results := calc.EvalLines(lines, 0)

				for i, result := range results {
					if strings.HasSuffix(result.Output, "ERR") {
						t.Errorf("Line %d (%q) produced error: %s", i+1, lines[i], result.Output)
					}
					// Verify that lines ending with = have a result
					if strings.HasSuffix(strings.TrimSpace(lines[i]), "=") && !result.HasResult {
						t.Errorf("Line %d (%q) should have result but HasResult=false", i+1, lines[i])
					}
				}
			})
		}
	}
}

// TestBasicMathSnippets tests the Basic Math category snippets individually
func TestBasicMathSnippets(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
	}{
		{
			name:  "Arithmetic",
			lines: []string{"10 + 20 * 3 ="},
		},
		{
			name:  "Currency",
			lines: []string{"$1,500.00 + $250.50 ="},
		},
		{
			name:  "Line Reference",
			lines: []string{"100 =", "\\1 * 2 ="},
		},
		{
			name:  "Scientific Functions",
			lines: []string{"sin(45) + cos(30) =", "sqrt(144) =", "abs(-50) ="},
		},
		{
			name:  "Complex Expression",
			lines: []string{"$1,000 x 12 - 15% + $500 ="},
		},
		{
			name:  "Comparison",
			lines: []string{"25 > 2.5 =", "100 >= 100 =", "5 != 3 ="},
		},
		{
			name:  "Base Conversion",
			lines: []string{"255 in hex =", "0xFF in dec =", "25 in bin =", "0b11001 in oct ="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := calc.EvalLines(tt.lines, 0)
			for i, result := range results {
				if strings.HasSuffix(result.Output, "ERR") {
					t.Errorf("Line %d (%q) produced error: %s", i+1, tt.lines[i], result.Output)
				}
				if !result.HasResult {
					t.Errorf("Line %d (%q) should have result", i+1, tt.lines[i])
				}
			}
		})
	}
}

// TestConstantsSnippets tests the Constants category snippets
func TestConstantsSnippets(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
	}{
		{
			name:  "Mathematical",
			lines: []string{"pi =", "e =", "phi =", "golden ratio ="},
		},
		{
			name:  "Physical",
			lines: []string{"speed of light =", "gravity =", "avogadro =", "planck ="},
		},
		{
			name:  "Value Lookup",
			lines: []string{"value of pi =", "value of speed of light ="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := calc.EvalLines(tt.lines, 0)
			for i, result := range results {
				if strings.HasSuffix(result.Output, "ERR") {
					t.Errorf("Line %d (%q) produced error: %s", i+1, tt.lines[i], result.Output)
				}
				if !result.HasResult {
					t.Errorf("Line %d (%q) should have result", i+1, tt.lines[i])
				}
			}
		})
	}
}

// TestDateTimeSnippets tests the Date & Time category snippets
func TestDateTimeSnippets(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
	}{
		{
			name:  "Current Time",
			lines: []string{"now =", "today ="},
		},
		{
			name:  "Time in City",
			lines: []string{"now in Seattle =", "now in New York =", "now in Kiev ="},
		},
		{
			name:  "Date Arithmetic",
			lines: []string{"today() =", "\\1 + 30 days =", "\\1 - 1 week ="},
		},
		{
			name:  "Date Difference",
			lines: []string{"19/01/22 - now ="},
		},
		{
			name:  "Duration Conversion",
			lines: []string{"861.5 hours in days =", "48 hours in days ="},
		},
		{
			name:  "Time Zone Conversion",
			lines: []string{"6:00 am Seattle in Kiev =", "11am Kiev in Seattle ="},
		},
		{
			name:  "Date Range",
			lines: []string{"Dec 6 till March 11 =", "Jan 1 until Dec 31 ="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := calc.EvalLines(tt.lines, 0)
			for i, result := range results {
				if strings.HasSuffix(result.Output, "ERR") {
					t.Errorf("Line %d (%q) produced error: %s", i+1, tt.lines[i], result.Output)
				}
				if !result.HasResult {
					t.Errorf("Line %d (%q) should have result", i+1, tt.lines[i])
				}
			}
		})
	}
}

// TestNetworkSnippets tests the Network/IP category snippets
func TestNetworkSnippets(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
	}{
		{
			name:  "Subnet Info",
			lines: []string{"10.100.0.0/24 =", "subnet info 10.100.0.0/16 ="},
		},
		{
			name:  "Split to Subnets",
			lines: []string{"10.100.0.0/16 / 4 subnets ="},
		},
		{
			name:  "Split by Hosts",
			lines: []string{"10.100.0.0/28 / 16 hosts ="},
		},
		{
			name:  "Subnet Mask",
			lines: []string{"mask for /24 =", "wildcard for /24 ="},
		},
		{
			name:  "IP in Range",
			lines: []string{"is 10.100.0.50 in 10.100.0.0/24 =", "is 192.168.1.100 in 192.168.1.0/28 ="},
		},
		{
			name:  "Next Subnet",
			lines: []string{"next subnet after 10.100.0.0/24 ="},
		},
		{
			name:  "Broadcast Address",
			lines: []string{"broadcast for 10.100.0.0/24 ="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := calc.EvalLines(tt.lines, 0)
			for i, result := range results {
				if strings.HasSuffix(result.Output, "ERR") {
					t.Errorf("Line %d (%q) produced error: %s", i+1, tt.lines[i], result.Output)
				}
				if !result.HasResult {
					t.Errorf("Line %d (%q) should have result", i+1, tt.lines[i])
				}
			}
		})
	}
}

// TestUnitConversionSnippets tests the Unit Conversions category snippets
func TestUnitConversionSnippets(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
	}{
		{
			name:  "Length",
			lines: []string{"5 miles in km =", "100 cm to inches =", "10 feet to meters ="},
		},
		{
			name:  "Weight",
			lines: []string{"10 kg in lbs =", "5 oz to grams =", "100 grams to oz ="},
		},
		{
			name:  "Temperature",
			lines: []string{"100 f to c =", "25 celsius to fahrenheit =", "0 kelvin to c ="},
		},
		{
			name:  "Volume",
			lines: []string{"5 gallons in liters =", "2 cups to ml =", "1 liter to ml ="},
		},
		{
			name:  "Data",
			lines: []string{"500 mb in gb =", "1 tb to gb =", "1024 kb to mb ="},
		},
		{
			name:  "Speed",
			lines: []string{"60 mph to kph =", "100 kph to mph ="},
		},
		{
			name:  "Area",
			lines: []string{"1 acre to sqft =", "100 sqm to sqft =", "1 hectare to acres ="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := calc.EvalLines(tt.lines, 0)
			for i, result := range results {
				if strings.HasSuffix(result.Output, "ERR") {
					t.Errorf("Line %d (%q) produced error: %s", i+1, tt.lines[i], result.Output)
				}
				if !result.HasResult {
					t.Errorf("Line %d (%q) should have result", i+1, tt.lines[i])
				}
			}
		})
	}
}

// TestPercentageSnippets tests the Percentage category snippets
func TestPercentageSnippets(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
	}{
		{
			name:  "Basic Percentage",
			lines: []string{"$100 - 20% =", "$100 + 15% ="},
		},
		{
			name:  "What is X% of Y",
			lines: []string{"what is 15% of 200 =", "what is 25% of 80 ="},
		},
		{
			name:  "What Percent Is",
			lines: []string{"50 is what % of 200 =", "75 is what percent of 300 ="},
		},
		{
			name:  "Increase/Decrease",
			lines: []string{"increase 100 by 20% =", "decrease 500 by 15% ="},
		},
		{
			name:  "Percent Change",
			lines: []string{"percent change from 50 to 75 =", "percent change from 100 to 80 ="},
		},
		{
			name:  "Tip Calculator",
			lines: []string{"tip 20% on $85.50 =", "tip 15% on $100 ="},
		},
		{
			name:  "Split Bill",
			lines: []string{"$150 split 4 ways =", "$200 split 4 ways with 18% tip ="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := calc.EvalLines(tt.lines, 0)
			for i, result := range results {
				if strings.HasSuffix(result.Output, "ERR") {
					t.Errorf("Line %d (%q) produced error: %s", i+1, tt.lines[i], result.Output)
				}
				if !result.HasResult {
					t.Errorf("Line %d (%q) should have result", i+1, tt.lines[i])
				}
			}
		})
	}
}

// TestFinancialSnippets tests the Financial category snippets
func TestFinancialSnippets(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
	}{
		{
			name:  "Loan Payment",
			lines: []string{"loan $250000 at 6.5% for 30 years =", "loan $50000 at 4% for 5 years ="},
		},
		{
			name:  "Mortgage",
			lines: []string{"mortgage $350000 at 7% for 30 years ="},
		},
		{
			name:  "Compound Interest",
			lines: []string{"$10000 at 5% for 10 years compounded monthly =", "compound interest $5000 at 7% for 5 years ="},
		},
		{
			name:  "Simple Interest",
			lines: []string{"simple interest $5000 at 3% for 2 years ="},
		},
		{
			name:  "Investment Growth",
			lines: []string{"invest $1000 at 7% for 20 years =", "invest $5000 at 10% for 10 years ="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := calc.EvalLines(tt.lines, 0)
			for i, result := range results {
				if strings.HasSuffix(result.Output, "ERR") {
					t.Errorf("Line %d (%q) produced error: %s", i+1, tt.lines[i], result.Output)
				}
				if !result.HasResult {
					t.Errorf("Line %d (%q) should have result", i+1, tt.lines[i])
				}
			}
		})
	}
}

// TestStatisticsSnippets tests the Statistics category snippets
func TestStatisticsSnippets(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
	}{
		{
			name:  "Average/Mean",
			lines: []string{"avg(10, 20, 30, 40) =", "mean(1, 2, 3, 4, 5) ="},
		},
		{
			name:  "Median",
			lines: []string{"median(1, 2, 3, 4, 5) =", "median(1, 2, 3, 4, 100) ="},
		},
		{
			name:  "Sum & Count",
			lines: []string{"sum(10, 20, 30) =", "count(1, 2, 3, 4, 5) ="},
		},
		{
			name:  "Min & Max",
			lines: []string{"min(10, 5, 20, 3) =", "max(10, 5, 20, 3) ="},
		},
		{
			name:  "Standard Deviation",
			lines: []string{"stddev(2, 4, 4, 4, 5, 5, 7, 9) ="},
		},
		{
			name:  "Variance",
			lines: []string{"variance(2, 4, 4, 4, 5, 5, 7, 9) ="},
		},
		{
			name:  "Range",
			lines: []string{"range(1, 5, 10, 3) ="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := calc.EvalLines(tt.lines, 0)
			for i, result := range results {
				if strings.HasSuffix(result.Output, "ERR") {
					t.Errorf("Line %d (%q) produced error: %s", i+1, tt.lines[i], result.Output)
				}
				if !result.HasResult {
					t.Errorf("Line %d (%q) should have result", i+1, tt.lines[i])
				}
			}
		})
	}
}

// TestProgrammerSnippets tests the Programmer category snippets
func TestProgrammerSnippets(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
	}{
		{
			name:  "Bitwise AND/OR/XOR",
			lines: []string{"0xFF AND 0x0F =", "0xF0 OR 0x0F =", "0xFF XOR 0x0F ="},
		},
		{
			name:  "Bit Shifts",
			lines: []string{"1 << 8 =", "256 >> 4 =", "0xFF << 4 ="},
		},
		{
			name:  "ASCII/Char",
			lines: []string{"ascii A =", "ascii a =", "char 65 =", "char 0x41 ="},
		},
		{
			name:  "ASCII Table",
			lines: []string{"ascii table ="},
		},
		{
			name:  "UUID Generation",
			lines: []string{"uuid ="},
		},
		{
			name:  "Hash Functions",
			lines: []string{"md5 hello =", "sha256 hello =", "sha1 test ="},
		},
		{
			name:  "Random Number",
			lines: []string{"random 1 to 100 =", "random 1-1000 ="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := calc.EvalLines(tt.lines, 0)
			for i, result := range results {
				if strings.HasSuffix(result.Output, "ERR") {
					t.Errorf("Line %d (%q) produced error: %s", i+1, tt.lines[i], result.Output)
				}
				if !result.HasResult {
					t.Errorf("Line %d (%q) should have result", i+1, tt.lines[i])
				}
			}
		})
	}
}

// TestRegexSnippets tests the Regex Tester category snippets
func TestRegexSnippets(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
	}{
		{
			name:  "Basic Match",
			lines: []string{`regex /hello/ test "hello world" =`, `regex /\d+/ test "abc123def" =`},
		},
		{
			name:  "No Match",
			lines: []string{`regex /xyz/ test "hello world" =`},
		},
		{
			name:  "Multiple Matches",
			lines: []string{`regex /\d+/ test "a1b2c3d4" =`},
		},
		{
			name:  "Capture Groups",
			lines: []string{`regex /(\w+)@(\w+)\.(\w+)/ test "email: user@example.com" =`},
		},
		{
			name:  "Word Boundary",
			lines: []string{`regex /\bword\b/ test "a word here" =`},
		},
		{
			name:  "Case Insensitive",
			lines: []string{`regex /(?i)hello/ test "HELLO World" =`},
		},
		{
			name:  "Phone Number",
			lines: []string{`regex /(\d{3})-(\d{3})-(\d{4})/ test "Call 555-123-4567 today" =`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := calc.EvalLines(tt.lines, 0)
			for i, result := range results {
				if strings.HasSuffix(result.Output, "ERR") {
					t.Errorf("Line %d (%q) produced error: %s", i+1, tt.lines[i], result.Output)
				}
				if !result.HasResult {
					t.Errorf("Line %d (%q) should have result", i+1, tt.lines[i])
				}
			}
		})
	}
}

// TestJWTSnippets tests the JWT Decoder category snippets
func TestJWTSnippets(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
	}{
		{
			name:  "Decode JWT Token",
			lines: []string{"jwt decode eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c ="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := calc.EvalLines(tt.lines, 0)
			for i, result := range results {
				if strings.HasSuffix(result.Output, "ERR") {
					t.Errorf("Line %d (%q) produced error: %s", i+1, tt.lines[i][:50], result.Output)
				}
				if !result.HasResult {
					t.Errorf("Line %d should have result", i+1)
				}
			}
		})
	}
}

// TestUnixPermissionsSnippets tests the Unix Permissions category snippets
func TestUnixPermissionsSnippets(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
	}{
		{
			name:  "Chmod Octal to Symbolic",
			lines: []string{"chmod 755 =", "chmod 644 =", "chmod 777 =", "chmod 600 ="},
		},
		{
			name:  "Chmod Symbolic to Octal",
			lines: []string{"chmod rwxr-xr-x =", "chmod rw-r--r-- ="},
		},
		{
			name:  "Special Bits",
			lines: []string{"chmod 4755 =", "chmod 2755 =", "chmod 1777 ="},
		},
		{
			name:  "Umask Calculator",
			lines: []string{"umask 022 =", "umask 077 =", "umask 027 ="},
		},
		{
			name:  "Permission Conversions",
			lines: []string{"755 to symbolic =", "rwxr-xr-x to octal =", "permission 644 ="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := calc.EvalLines(tt.lines, 0)
			for i, result := range results {
				if strings.HasSuffix(result.Output, "ERR") {
					t.Errorf("Line %d (%q) produced error: %s", i+1, tt.lines[i], result.Output)
				}
				if !result.HasResult {
					t.Errorf("Line %d (%q) should have result", i+1, tt.lines[i])
				}
			}
		})
	}
}

// TestAllSnippetCategoriesExist verifies the expected categories are present
func TestAllSnippetCategoriesExist(t *testing.T) {
	expectedCategories := []string{
		"Basic Math",
		"Constants",
		"Date & Time",
		"Network/IP",
		"Unit Conversions",
		"Percentage",
		"Financial",
		"Statistics",
		"Programmer",
		"Regex Tester",
		"Unix Permissions",
		"Crontab Explanation",
		"HTTP Status Codes",
		"JWT Decoder",
		"SSL Certificate",
		"Networking Utilities",
		"Color Conversions",
		"Electrical/Radio",
		"Cooking Conversions",
		"Man-Hour Calculations",
		"Hourly Cost Calculations",
	}

	categories := GetSnippetCategories()
	categoryNames := make(map[string]bool)
	for _, cat := range categories {
		categoryNames[cat.Name] = true
	}

	for _, expected := range expectedCategories {
		if !categoryNames[expected] {
			t.Errorf("Expected category %q not found", expected)
		}
	}

	if len(categories) != len(expectedCategories) {
		t.Errorf("Expected %d categories, got %d", len(expectedCategories), len(categories))
	}
}

// TestSnippetStructure verifies each snippet has non-empty name and content
func TestSnippetStructure(t *testing.T) {
	categories := GetSnippetCategories()

	for _, category := range categories {
		if category.Name == "" {
			t.Error("Found category with empty name")
		}
		if len(category.Snippets) == 0 {
			t.Errorf("Category %q has no snippets", category.Name)
		}

		for _, snippet := range category.Snippets {
			if snippet.Name == "" {
				t.Errorf("Category %q has snippet with empty name", category.Name)
			}
			if snippet.Content == "" {
				t.Errorf("Category %q, snippet %q has empty content", category.Name, snippet.Name)
			}
			// Each snippet should contain at least one expression ending with =
			if !strings.Contains(snippet.Content, "=") {
				t.Errorf("Category %q, snippet %q has no expression (missing '=')", category.Name, snippet.Name)
			}
		}
	}
}
