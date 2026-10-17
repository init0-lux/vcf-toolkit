package normalize

import (
	"testing"

	"github.com/init0-lux/vcf-toolkit/internal/model"
)

func TestNormalizeName(t *testing.T) {
	cfg := NameConfig{
		StripSuffix: true,
		OrgPatterns: []string{"Inc.", "Ltd.", "CSI VIT"},
		UseLLM:      false,
	}

	tests := []struct {
		raw      string
		expected model.NormalizedName
	}{
		{"John Doe Inc.", model.NormalizedName{FullName: "John Doe", FirstName: "John", LastName: "Doe", Organization: "Inc.", Normalized: true}},
		{"Rahul CSI VIT", model.NormalizedName{FullName: "Rahul", FirstName: "Rahul", Organization: "CSI VIT", Normalized: true}},
		{"   ", model.NormalizedName{Normalized: false}},
	}

	for _, test := range tests {
		result := NormalizeName(test.raw, cfg)
		if result != test.expected {
			t.Errorf("NormalizeName(%q) = %+v; want %+v", test.raw, result, test.expected)
		}
	}
}

func TestNormalizePhone_DefaultUS(t *testing.T) {
	cfg := PhoneConfig{
		DefaultCountry:   "+1",
		StrictValidation: true,
	}

	tests := []struct {
		name     string
		raw      string
		expected model.NormalizedPhone
	}{
		{
			name: "US local number formats to E.164",
			raw:  "(123) 456-7890",
			expected: model.NormalizedPhone{
				Raw: "(123) 456-7890", Normalized: "+11234567890",
				Valid: true, Country: "+1",
			},
		},
		{
			name: "US number without formatting",
			raw:  "2125550148",
			expected: model.NormalizedPhone{
				Raw: "2125550148", Normalized: "+12125550148",
				Valid: true, Country: "+1",
			},
		},
		{
			name: "UK number with + detects country +44",
			raw:  "+44 20 7946 0958",
			expected: model.NormalizedPhone{
				Raw: "+44 20 7946 0958", Normalized: "+442079460958",
				Valid: true, Country: "+44",
			},
		},
		{
			name: "India number with + detects country +91",
			raw:  "+91 98765 43210",
			expected: model.NormalizedPhone{
				Raw: "+91 98765 43210", Normalized: "+919876543210",
				Valid: true, Country: "+91",
			},
		},
		{
			name: "China number with + detects country +86",
			raw:  "+86 138 0013 8000",
			expected: model.NormalizedPhone{
				Raw: "+86 138 0013 8000", Normalized: "+8613800138000",
				Valid: true, Country: "+86",
			},
		},
		{
			name: "Germany number with + detects country +49",
			raw:  "+49 176 1234 5678",
			expected: model.NormalizedPhone{
				Raw: "+49 176 1234 5678", Normalized: "+4917612345678",
				Valid: true, Country: "+49",
			},
		},
		{
			name: "France number with + detects country +33",
			raw:  "+33 6 12 34 56 78",
			expected: model.NormalizedPhone{
				Raw: "+33 6 12 34 56 78", Normalized: "+33612345678",
				Valid: true, Country: "+33",
			},
		},
		{
			name: "Australia number with + detects country +61",
			raw:  "+61 2 9876 5432",
			expected: model.NormalizedPhone{
				Raw: "+61 2 9876 5432", Normalized: "+61298765432",
				Valid: true, Country: "+61",
			},
		},
		{
			name: "Japan number with + detects country +81",
			raw:  "+81 3 1234 5678",
			expected: model.NormalizedPhone{
				Raw: "+81 3 1234 5678", Normalized: "+81312345678",
				Valid: true, Country: "+81",
			},
		},
		{
			name: "Morocco number with 3-digit code +212",
			raw:  "+212 6 12 34 56 78",
			expected: model.NormalizedPhone{
				Raw: "+212 6 12 34 56 78", Normalized: "+212612345678",
				Valid: true, Country: "+212",
			},
		},
		{
			name: "invalid short number rejected",
			raw:  "123",
			expected: model.NormalizedPhone{
				Raw: "123", Valid: false,
			},
		},
		{
			name: "empty string rejected",
			raw:  "   ",
			expected: model.NormalizedPhone{
				Raw: "   ", Valid: false, Country: "",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizePhone(tc.raw, cfg)
			if result != tc.expected {
				t.Errorf("NormalizePhone(%q) =\n  %+v\nwant:\n  %+v", tc.raw, result, tc.expected)
			}
		})
	}
}

func TestNormalizePhone_DefaultIndia(t *testing.T) {
	cfg := PhoneConfig{
		DefaultCountry:   "+91",
		StrictValidation: true,
	}

	tests := []struct {
		name     string
		raw      string
		expected model.NormalizedPhone
	}{
		{
			name: "Indian local 10-digit gets +91 prepended",
			raw:  "9876543210",
			expected: model.NormalizedPhone{
				Raw: "9876543210", Normalized: "+919876543210",
				Valid: true, Country: "+91",
			},
		},
		{
			name: "Indian number with leading 0 strips 0 and prepends +91",
			raw:  "09876543210",
			expected: model.NormalizedPhone{
				Raw: "09876543210", Normalized: "+919876543210",
				Valid: true, Country: "+91",
			},
		},
		{
			name: "Indian number with + preserves +91",
			raw:  "+91 98765 43210",
			expected: model.NormalizedPhone{
				Raw: "+91 98765 43210", Normalized: "+919876543210",
				Valid: true, Country: "+91",
			},
		},
		{
			name: "Chinese number with + still detects +86",
			raw:  "+86 138 0013 8000",
			expected: model.NormalizedPhone{
				Raw: "+86 138 0013 8000", Normalized: "+8613800138000",
				Valid: true, Country: "+86",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizePhone(tc.raw, cfg)
			if result != tc.expected {
				t.Errorf("NormalizePhone(%q) =\n  %+v\nwant:\n  %+v", tc.raw, result, tc.expected)
			}
		})
	}
}

func TestNormalizePhone_NonStrict(t *testing.T) {
	cfg := PhoneConfig{
		DefaultCountry:   "",
		StrictValidation: false,
	}

	tests := []struct {
		name     string
		raw      string
		expected model.NormalizedPhone
	}{
		{
			name: "non-strict returns cleaned digits without country",
			raw:  "1234567890",
			expected: model.NormalizedPhone{
				Raw: "1234567890", Normalized: "1234567890",
				Valid: true, Country: "",
			},
		},
		{
			name: "non-strict with formatting stripped",
			raw:  "+44 20 7946 0958",
			expected: model.NormalizedPhone{
				Raw: "+44 20 7946 0958", Normalized: "+442079460958",
				Valid: true, Country: "+44",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizePhone(tc.raw, cfg)
			if result != tc.expected {
				t.Errorf("NormalizePhone(%q) =\n  %+v\nwant:\n  %+v", tc.raw, result, tc.expected)
			}
		})
	}
}

func TestNormalizeEmail(t *testing.T) {
	cfg := EmailConfig{
		StripAliases:     true,
		StrictValidation: true,
	}

	tests := []struct {
		raw      string
		expected model.NormalizedEmail
	}{
		{"User+tag@example.com", model.NormalizedEmail{Raw: "User+tag@example.com", Normalized: "user@example.com", Valid: true}},
		{"invalid-email", model.NormalizedEmail{Raw: "invalid-email", Valid: false}},
		{"   ", model.NormalizedEmail{Raw: "   ", Valid: false}},
	}

	for _, test := range tests {
		result := NormalizeEmail(test.raw, cfg)
		if result != test.expected {
			t.Errorf("NormalizeEmail(%q) = %+v; want %+v", test.raw, result, test.expected)
		}
	}
}

func TestNormalizePhone_CountryHandling(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		country string
		strict  bool
		want    string
		valid   bool
		cc      string
	}{
		// Without a default country a local number never gains a "+".
		{"local US number, no country", "(212) 555-0101", "", false, "2125550101", true, ""},
		{"local Indian number, no country", "98765 43210", "", false, "9876543210", true, ""},
		{"local number, no country, strict", "(212) 555-0101", "", true, "", false, ""},
		{"too short, no country", "555-01", "", false, "", false, ""},

		{"US local with +1", "(212) 555-0101", "+1", true, "+12125550101", true, "+1"},
		{"US with country digits but no plus", "1 212 555 0101", "+1", true, "+12125550101", true, "+1"},
		{"US 7-digit with +1 is rejected", "555-0199", "+1", true, "", false, ""},
		{"US 7-digit with +1, non-strict keeps digits", "555-0199", "+1", false, "5550199", true, ""},
		{"country code without plus accepted", "(212) 555-0101", "1", true, "+12125550101", true, "+1"},

		{"Indian local with +91", "98765 43210", "+91", true, "+919876543210", true, "+91"},
		{"Indian with 91 but no plus", "919876543210", "+91", true, "+919876543210", true, "+91"},
		{"Indian with trunk 0", "098765 43210", "+91", true, "+919876543210", true, "+91"},

		{"UK national with trunk 0", "020 7946 0958", "+44", true, "+442079460958", true, "+44"},
		{"UK with 44 but no plus", "44 20 7946 0958", "+44", true, "+442079460958", true, "+44"},
		{"French national", "06 12 34 56 78", "+33", true, "+33612345678", true, "+33"},
		// A trunk 0 marks a national number, so a leading 49 is an area code.
		{"German area code starting with 49", "0491 12345678", "+49", true, "+4949112345678", true, "+49"},

		{"00 international prefix", "0044 20 7946 0958", "+91", true, "+442079460958", true, "+44"},
		{"00 international prefix, no country", "0044 20 7946 0958", "", false, "+442079460958", true, "+44"},
		{"unknown default country is ignored", "(212) 555-0101", "+999", false, "2125550101", true, ""},
		// The trunk 0 is part of a local number; keep it when no country applies.
		{"UK local, no country, keeps trunk 0", "020 7946 0958", "", false, "02079460958", true, ""},
		{"parenthesized plus is international", "(+44) 20 7946 0958", "+33", true, "+442079460958", true, "+44"},
		{"plus after label is international", "Tel: +44 20 7946 0958", "+33", true, "+442079460958", true, "+44"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizePhone(tc.raw, PhoneConfig{DefaultCountry: tc.country, StrictValidation: tc.strict})
			want := model.NormalizedPhone{Raw: tc.raw, Normalized: tc.want, Valid: tc.valid, Country: tc.cc}
			if got != want {
				t.Errorf("NormalizePhone(%q, %q) =\n  %+v\nwant:\n  %+v", tc.raw, tc.country, got, want)
			}
		})
	}
}

func TestNormalizeCountryCode(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"+91", "+91", true},
		{"91", "+91", true},
		{" +1 ", "+1", true},
		{"+44", "+44", true},
		{"+212", "+212", true},
		{"IN", "", false},
		{"+999", "", false},
		{"+9a", "", false},
		{"", "", false},
	}
	for _, tc := range tests {
		got, ok := NormalizeCountryCode(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("NormalizeCountryCode(%q) = %q, %v; want %q, %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
