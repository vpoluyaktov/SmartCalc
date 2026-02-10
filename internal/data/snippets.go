package data

// Snippet represents a menu snippet with a name and content
type Snippet struct {
	Name    string
	Content string
}

// SnippetCategory represents a category of snippets
type SnippetCategory struct {
	Name     string
	Snippets []Snippet
}

// GetSnippetCategories returns all snippet categories for the menu
func GetSnippetCategories() []SnippetCategory {
	return []SnippetCategory{
		{
			Name: "Basic Math",
			Snippets: []Snippet{
				{"Arithmetic", "10 + 20 * 3 =\n\n"},
				{"Currency", "$1,500.00 + $250.50 =\n\n"},
				{"Line Reference", "100 =\n\\1 * 2 =\n\n"},
				{"Scientific Functions", "sin(45) + cos(30) =\nsqrt(144) =\nabs(-50) =\n\n"},
				{"Complex Expression", "$1,000 x 12 - 15% + $500 =\n\n"},
				{"Comparison", "25 > 2.5 =\n100 >= 100 =\n5 != 3 =\n\n"},
				{"Base Conversion", "255 in hex =\n0xFF in dec =\n25 in bin =\n0b11001 in oct =\n\n"},
			},
		},
		{
			Name: "Constants",
			Snippets: []Snippet{
				{"Mathematical", "pi =\ne =\nphi =\ngolden ratio =\n\n"},
				{"Physical", "speed of light =\ngravity =\navogadro =\nplanck =\n\n"},
				{"Value Lookup", "value of pi =\nvalue of speed of light =\n\n"},
			},
		},
		{
			Name: "Date & Time",
			Snippets: []Snippet{
				{"Current Time", "now =\ntoday =\n\n"},
				{"Time in City", "now in Seattle =\nnow in New York =\nnow in Kiev =\n\n"},
				{"Date Arithmetic", "today() =\n\\1 + 30 days =\n\\1 - 1 week =\n\n"},
				{"Date Difference", "19/01/22 - now =\n\n"},
				{"Duration Conversion", "861.5 hours in days =\n48 hours in days =\n\n"},
				{"Time Zone Conversion", "6:00 am Seattle in Kiev =\n11am Kiev in Seattle =\n\n"},
				{"Date Range", "Dec 6 till March 11 =\nJan 1 until Dec 31 =\n\n"},
			},
		},
		{
			Name: "Network/IP",
			Snippets: []Snippet{
				{"Subnet Info", "10.100.0.0/24 =\n\nsubnet info 10.100.0.0/16 =\n\n"},
				{"Split to Subnets", "10.100.0.0/16 / 4 subnets =\n\n"},
				{"Split by Hosts", "10.100.0.0/16 / 4096 hosts =\n\n"},
				{"Subnet Mask", "mask for /24 =\nwildcard for /24 =\n\n"},
				{"IP in Range", "is 10.100.0.50 in 10.100.0.0/24 =\nis 192.168.1.100 in 192.168.1.0/28 =\n\n"},
				{"Next Subnet", "next subnet after 10.100.0.0/24 =\n\n"},
				{"Broadcast Address", "broadcast for 10.100.0.0/24 =\n\n"},
			},
		},
		{
			Name: "Unit Conversions",
			Snippets: []Snippet{
				{"Length", "5 miles in km =\n100 cm to inches =\n10 feet to meters =\n\n"},
				{"Weight", "10 kg in lbs =\n5 oz to grams =\n100 grams to oz =\n\n"},
				{"Temperature", "100 f to c =\n25 celsius to fahrenheit =\n0 kelvin to c =\n\n"},
				{"Volume", "5 gallons in liters =\n2 cups to ml =\n1 liter to ml =\n\n"},
				{"Data (SI)", "# SI units (base 1000): KB, MB, GB, TB\n1234567 bytes to mb =\n500 mb in gb =\n1 tb to gb =\n\n"},
				{"Data (IEC)", "# IEC units (base 1024): KiB, MiB, GiB, TiB\n1234567 bytes to mib =\n1024 mib to gib =\n1 tib to gib =\n\n"},
				{"Speed", "60 mph to kph =\n100 kph to mph =\n\n"},
				{"Area", "1 acre to sqft =\n100 sqm to sqft =\n1 hectare to acres =\n\n"},
			},
		},
		{
			Name: "Percentage",
			Snippets: []Snippet{
				{"Basic Percentage", "$100 - 20% =\n$100 + 15% =\n\n"},
				{"What is X% of Y", "what is 15% of 200 =\nwhat is 25% of 80 =\n\n"},
				{"What Percent Is", "50 is what % of 200 =\n75 is what percent of 300 =\n\n"},
				{"Increase/Decrease", "increase 100 by 20% =\ndecrease 500 by 15% =\n\n"},
				{"Percent Change", "percent change from 50 to 75 =\npercent change from 100 to 80 =\n\n"},
				{"Tip Calculator", "tip 20% on $85.50 =\ntip 15% on $100 =\n\n"},
				{"Split Bill", "$150 split 4 ways =\n$200 split 4 ways with 18% tip =\n\n"},
			},
		},
		{
			Name: "Financial",
			Snippets: []Snippet{
				{"Loan Payment", "loan $250000 at 6.5% for 30 years =\n\nloan $50000 at 4% for 5 years =\n\n"},
				{"Mortgage", "mortgage $350000 at 7% for 30 years =\n\n"},
				{"Mortgage Pay Schedule", "mortgage $100000 at 5% for 1 year pay schedule =\n\n"},
				{"Mortgage Extra Payment", "mortgage $350000 at 7% for 30 years extra payment $500 =\n\n"},
				{"Compound Interest", "$10000 at 5% for 10 years compounded monthly =\n\ncompound interest $5000 at 7% for 5 years =\n\n"},
				{"Simple Interest", "simple interest $5000 at 3% for 2 years =\n\n"},
				{"Investment Growth", "invest $1000 at 7% for 20 years =\n\ninvest $5000 at 10% for 10 years =\n\n"},
			},
		},
		{
			Name: "Statistics",
			Snippets: []Snippet{
				{"Average/Mean", "avg(10, 20, 30, 40) =\nmean(1, 2, 3, 4, 5) =\n\n"},
				{"Median", "median(1, 2, 3, 4, 5) =\nmedian(1, 2, 3, 4, 100) =\n\n"},
				{"Sum & Count", "sum(10, 20, 30) =\ncount(1, 2, 3, 4, 5) =\n\n"},
				{"Min & Max", "min(10, 5, 20, 3) =\nmax(10, 5, 20, 3) =\n\n"},
				{"Standard Deviation", "stddev(2, 4, 4, 4, 5, 5, 7, 9) =\n\n"},
				{"Variance", "variance(2, 4, 4, 4, 5, 5, 7, 9) =\n\n"},
				{"Range", "range(1, 5, 10, 3) =\n\n"},
			},
		},
		{
			Name: "Programmer",
			Snippets: []Snippet{
				{"Bitwise AND/OR/XOR", "0xFF AND 0x0F =\n0xF0 OR 0x0F =\n0xFF XOR 0x0F =\n\n"},
				{"Bit Shifts", "1 << 8 =\n256 >> 4 =\n0xFF << 4 =\n\n"},
				{"ASCII/Char", "ascii A =\nascii a =\nchar 65 =\nchar 0x41 =\n\n"},
				{"ASCII Table", "ascii table =\n\n"},
				{"UUID Generation", "uuid =\n\n"},
				{"Hash Functions", "md5 hello =\nsha256 hello =\nsha1 test =\n\n"},
				{"Base64 Encode/Decode", "base64 encode hello world =\nbase64 decode SGVsbG8gd29ybGQ= =\n\n"},
				{"Random Number", "random 1 to 100 =\nrandom 1-1000 =\n\n"},
				{"Password Generator", "pwgen =\n\npwgen -c 20 =\n\npwgen -h =\n\npwgen -c 12 -h =\n\n"},
			},
		},
		{
			Name: "Regex Tester",
			Snippets: []Snippet{
				{"Basic Match", "regex /hello/ test \"hello world\" =\n\nregex /\\d+/ test \"abc123def\" =\n\n"},
				{"No Match", "regex /xyz/ test \"hello world\" =\n\n"},
				{"Multiple Matches", "regex /\\d+/ test \"a1b2c3d4\" =\n\n"},
				{"Capture Groups", "regex /(\\w+)@(\\w+)\\.(\\w+)/ test \"email: user@example.com\" =\n\n"},
				{"Word Boundary", "regex /\\bword\\b/ test \"a word here\" =\n\n"},
				{"Case Insensitive", "regex /(?i)hello/ test \"HELLO World\" =\n\n"},
				{"Phone Number", "regex /(\\d{3})-(\\d{3})-(\\d{4})/ test \"Call 555-123-4567 today\" =\n\n"},
				{"URL Pattern", "regex /https?:\\/\\/[\\w.-]+/ test \"Visit https://example.com for more\" =\n\n"},
			},
		},
		{
			Name: "Unix Permissions",
			Snippets: []Snippet{
				{"Chmod Octal to Symbolic", "chmod 755 =\nchmod 644 =\nchmod 777 =\nchmod 600 =\n\n"},
				{"Chmod Symbolic to Octal", "chmod rwxr-xr-x =\nchmod rw-r--r-- =\nchmod rwx r-x r-x =\n\n"},
				{"Special Bits (setuid/setgid/sticky)", "chmod 4755 =\nchmod 2755 =\nchmod 1777 =\nchmod 7755 =\n\n"},
				{"Umask Calculator", "umask 022 =\numask 077 =\numask 027 =\numask 002 =\n\n"},
				{"Permission Conversions", "755 to symbolic =\nrwxr-xr-x to octal =\npermission 644 =\n\n"},
			},
		},
		{
			Name: "Crontab Explanation",
			Snippets: []Snippet{
				{"Every N Minutes", "cron */5 * * * * =\ncron */15 * * * * =\n\n"},
				{"Specific Time", "cron 0 0 * * * =\ncron 30 2 * * * =\ncron 0 9 * * * =\n\n"},
				{"Weekdays Only", "cron 30 2 * * 1-5 =\ncron 0 9 * * 1,5 =\n\n"},
				{"Monthly/Yearly", "cron 0 0 1 * * =\ncron 0 0 1 1 * =\n\n"},
				{"Short Aliases", "cron @daily =\ncron @hourly =\ncron @weekly =\ncron @monthly =\ncron @yearly =\n\n"},
			},
		},
		{
			Name: "HTTP Status Codes",
			Snippets: []Snippet{
				{"Common Success", "http code 200 =\nhttp code 201 =\nhttp code 204 =\n\n"},
				{"Redirections", "http code 301 =\nhttp code 302 =\nhttp code 304 =\nhttp code 307 =\n\n"},
				{"Client Errors", "http code 400 =\nhttp code 401 =\nhttp code 403 =\nhttp code 404 =\nhttp code 429 =\n\n"},
				{"Server Errors", "http code 500 =\nhttp code 502 =\nhttp code 503 =\nhttp code 504 =\n\n"},
				{"Fun & Uncommon", "http code 418 =\nhttp code 451 =\nhttp code 103 =\nhttp code 426 =\n\n"},
			},
		},
		{
			Name: "JWT Decoder",
			Snippets: []Snippet{
				{"Decode JWT Token", "jwt decode eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c =\n\n"},
				{"JWT with Expiration", "jwt eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiZXhwIjoxNzM1Njg5NjAwfQ.signature =\n\n"},
			},
		},
		{
			Name: "SSL Certificate",
			Snippets: []Snippet{
				{"Decode Certificate (Google)", "cert decode https://google.com =\n\n"},
				{"Decode Certificate (GitHub)", "ssl decode https://github.com =\n\n"},
				{"Decode Certificate (Custom)", "cert decode example.com =\n\n"},
				{"Test Certificate (Expired)", "cert test https://expired.badssl.com =\n\n"},
				{"Test Certificate (Self-Signed)", "ssl test https://self-signed.badssl.com =\n\n"},
			},
		},
		{
			Name: "Networking Utilities",
			Snippets: []Snippet{
				{"DNS Lookup", "# DNS lookup (aliases: dig, nslookup, dns, lookup, resolve)\ndig google.com =\n\n"},
				{"WHOIS Lookup", "# Domain registration info\nwhois google.com =\n\n"},
				{"IP Geolocation", "# IP geolocation (aliases: geoip, ip location, ip lookup, locate ip, where is)\ngeoip 8.8.8.8 =\n\nip lookup 1.1.1.1 =\n\n"},
				{"My IP Address", "# Get your public IP address\nwhat is my ip =\nmy ip =\n\n"},
			},
		},
		{
			Name: "Color Conversions",
			Snippets: []Snippet{
				{"Hex to RGB", "#FF5733 to rgb =\n#FFF to rgb =\n\n"},
				{"Hex to HSL", "#FF5733 to hsl =\n#00FF00 to hsl =\n\n"},
				{"RGB to Hex", "rgb(255, 87, 51) to hex =\nrgb(0, 128, 255) to hex =\n\n"},
				{"RGB to HSL", "rgb(255, 0, 0) to hsl =\nrgb(0, 255, 0) to hsl =\n\n"},
				{"HSL to RGB", "hsl(0, 100%, 50%) to rgb =\nhsl(120, 100%, 50%) to rgb =\n\n"},
				{"HSL to Hex", "hsl(240, 100%, 50%) to hex =\nhsl(60, 100%, 50%) to hex =\n\n"},
			},
		},
		{
			Name: "Electrical/Radio",
			Snippets: []Snippet{
				{"Ohm's Law Calculator", "12v 2a =\n\n24v 100ohm =\n\n100w 50ohm =\n\n"},
				{"Power/dBm Conversion", "30 dbm to watts =\n1 watt to dbm =\n100 mw to dbm =\n\n"},
				{"Decibel Conversion", "3 db to times =\n6 db to times voltage =\n2 times to db =\n\n"},
				{"Frequency to Wavelength", "14.2 MHz to meters =\n146 MHz to m =\n440 MHz to meters =\n\n"},
				{"Wavelength to Frequency", "2 m to MHz =\n70 cm to MHz =\n20 meters to MHz =\n\n"},
				{"Dipole Antenna", "dipole for 14.2 MHz =\n\ndipole for 146 MHz =\n\n"},
				{"Quarter Wave Vertical", "quarter wave for 14.2 MHz =\n\n\n1/4 wave 146 MHz =\n\n"},
				{"Yagi Antenna", "yagi for 144 MHz =\n\nyagi for 14.2 MHz =\n\n"},
				{"SWR Calculator", "swr 1.5 =\n\nswr 50 75 =\n\n\n"},
				{"Band Information", "radio band 14.2 MHz =\n\nradio band 146 MHz =\n\n20m band =\n\n"},
				{"Velocity Factor", "10m vf=0.66 =\n2m cable vf 0.82 =\n\n"},
			},
		},
		{
			Name: "Cooking Conversions",
			Snippets: []Snippet{
				{"Volume Conversions", "2 cups to tbsp =\n1 cup to ml =\n3 tbsp to tsp =\n1 pint to cups =\n\n"},
				{"Butter Measurements", "1 stick butter =\n2 sticks to grams =\n1 cup butter to grams =\n\n"},
				{"Flour Conversions", "1 cup flour to grams =\n250g flour to cups =\n2 tbsp flour to grams =\n\n"},
				{"Sugar Conversions", "1 cup sugar to grams =\n1 cup brown sugar to grams =\n100g sugar to cups =\n\n"},
				{"Common Ingredients", "1 cup honey to grams =\n1 cup rice to grams =\n1 cup oats to grams =\n\n"},
				{"Oven Temperatures", "350 f to c =\n180 c to f =\ngas mark 4 =\ngas mark 6 to f =\n\n"},
			},
		},
		{
			Name: "Man-Hour Calculations",
			Snippets: []Snippet{
				{"Business Time", "# Business time: 8h/day, 40h/week, 160h/month\n248 man-hours / 3 men in business weeks =\n160 man-hours / 2 men in business days =\n320 man-hours / 2 men in business months =\n\n"},
				{"Calendar Time", "# Calendar time: 24h/day, 168h/week, 720h/month\n248 man-hours / 3 men in weeks =\n72 man-hours / 3 men in days =\n1440 man-hours / 2 men in months =\n\n"},
				{"Project Estimation", "# How long will 500 man-hours take with 4 developers?\n500 man-hours / 4 men in business weeks =\n500 man-hours / 4 men in business days =\n\n"},
			},
		},
		{
			Name: "Hourly Cost Calculations",
			Snippets: []Snippet{
				{"Weekly Cost", "# Calculate total cost for hourly rate over time\n$35 per hour in week =\n$50 per hour in 2 weeks =\n\n"},
				{"Monthly Cost", "# Monthly and multi-month costs\n$45 per hour in month =\n$45 per hour in 5 months =\n\n"},
				{"Yearly Cost", "# Annual cost calculations\n$100 per hour in year =\n$100 per hour in 2 years =\n\n"},
				{"Cents per Hour", "# Small rates add up!\n25 cents per hour in year =\n50 cents per hour in 2 years =\n\n"},
			},
		},
	}
}
