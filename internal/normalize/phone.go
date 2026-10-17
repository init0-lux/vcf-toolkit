package normalize

import (
	"strings"

	"github.com/init0-lux/vcf-toolkit/internal/model"
)

type PhoneConfig struct {
	DefaultCountry   string
	StrictValidation bool
}

// NormalizePhone converts a phone number to E.164 when its country is known:
// either written with "+" or "00", or supplied via cfg.DefaultCountry. Numbers
// without a known country are never given one; in non-strict mode they are
// returned as plain digits with an empty Country.
func NormalizePhone(raw string, cfg PhoneConfig) model.NormalizedPhone {
	orig := raw
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return model.NormalizedPhone{Raw: orig, Valid: false}
	}

	normalized, country, valid := normalizePhone(raw, cfg)
	return model.NormalizedPhone{
		Raw:        orig,
		Normalized: normalized,
		Valid:      valid,
		Country:    country,
	}
}

// NormalizeCountryCode returns the "+<digits>" form of a calling code given as
// "+91" or "91", and whether it is a known code.
func NormalizeCountryCode(s string) (string, bool) {
	d := strings.TrimPrefix(strings.TrimSpace(s), "+")
	if d == "" {
		return "", false
	}
	for _, r := range d {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	code, ok := countryCodes[d]
	return code, ok
}

// nationalLengths pins the national number length for countries where it is
// fixed, so a mistyped number is rejected rather than prefixed.
var nationalLengths = map[string]int{"+1": 10, "+91": 10}

func normalizePhone(raw string, cfg PhoneConfig) (normalized, country string, valid bool) {
	digits := allDigits(raw)

	international := strings.HasPrefix(raw, "+")
	if !international && strings.HasPrefix(digits, "00") {
		international = true
		digits = digits[2:]
	}
	if international {
		if len(digits) < 8 || len(digits) > 15 || digits[0] == '0' {
			return "", "", false
		}
		e164 := "+" + digits
		return e164, extractCountryCode(e164), true
	}

	// A leading 0 is a national trunk prefix.
	hasTrunk := strings.HasPrefix(digits, "0")
	national := strings.TrimLeft(digits, "0")

	if cc, ok := NormalizeCountryCode(cfg.DefaultCountry); ok {
		if n, ok := nationalNumber(national, cc, hasTrunk); ok {
			return cc + n, cc, true
		}
	}

	if cfg.StrictValidation || len(national) < 7 || len(national) > 15 {
		return "", "", false
	}
	return national, "", true
}

// nationalNumber validates a number written without a country code against cc,
// dropping the country digits when they were typed without a "+".
func nationalNumber(national, cc string, hasTrunk bool) (string, bool) {
	ccDigits := cc[1:]
	typedCC := !hasTrunk && strings.HasPrefix(national, ccDigits)

	if want, fixed := nationalLengths[cc]; fixed {
		switch {
		case len(national) == want:
			return national, true
		case typedCC && len(national)-len(ccDigits) == want:
			return national[len(ccDigits):], true
		}
		return "", false
	}

	if typedCC && len(national)-len(ccDigits) >= 9 {
		national = national[len(ccDigits):]
	}
	if len(national) < 7 || len(ccDigits)+len(national) > 15 {
		return "", false
	}
	return national, true
}

func allDigits(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// PhoneDigits strips non-numeric characters and removes leading zeros.
// Returns a comparable digit string suitable for deduplication lookups.
func PhoneDigits(s string) string {
	return strings.TrimLeft(allDigits(s), "0")
}

func extractCountryCode(e164 string) string {
	if !strings.HasPrefix(e164, "+") {
		return ""
	}
	digits := e164[1:]
	for i := 3; i >= 1; i-- {
		if len(digits) >= i {
			prefix := digits[:i]
			if code, ok := countryCodes[prefix]; ok {
				return code
			}
		}
	}
	return ""
}

var countryCodes = map[string]string{
	"1":  "+1",
	"20": "+20", "27": "+27", "30": "+30", "31": "+31",
	"32": "+32", "33": "+33", "34": "+34", "36": "+36",
	"39": "+39", "40": "+40", "41": "+41", "43": "+43",
	"44": "+44", "45": "+45", "46": "+46", "47": "+47",
	"48": "+48", "49": "+49", "51": "+51", "52": "+52",
	"53": "+53", "54": "+54", "55": "+55", "56": "+56",
	"57": "+57", "58": "+58", "60": "+60", "61": "+61",
	"62": "+62", "63": "+63", "64": "+64", "65": "+65",
	"66": "+66", "81": "+81", "82": "+82", "84": "+84",
	"86": "+86", "90": "+90", "91": "+91", "92": "+92",
	"93": "+93", "94": "+94", "95": "+95", "98": "+98",
	"212": "+212", "213": "+213", "216": "+216", "218": "+218",
	"220": "+220", "221": "+221", "222": "+222", "223": "+223",
	"224": "+224", "225": "+225", "226": "+226", "227": "+227",
	"228": "+228", "229": "+229", "230": "+230", "231": "+231",
	"232": "+232", "233": "+233", "234": "+234", "235": "+235",
	"236": "+236", "237": "+237", "238": "+238", "239": "+239",
	"240": "+240", "241": "+241", "242": "+242", "243": "+243",
	"244": "+244", "245": "+245", "246": "+246", "247": "+247",
	"248": "+248", "249": "+249", "250": "+250", "251": "+251",
	"252": "+252", "253": "+253", "254": "+254", "255": "+255",
	"256": "+256", "257": "+257", "258": "+258", "260": "+260",
	"261": "+261", "262": "+262", "263": "+263", "264": "+264",
	"265": "+265", "266": "+266", "267": "+267", "268": "+268",
	"269": "+269", "290": "+290", "291": "+291", "297": "+297",
	"298": "+298", "299": "+299", "350": "+350", "351": "+351",
	"352": "+352", "353": "+353", "354": "+354", "355": "+355",
	"356": "+356", "357": "+357", "358": "+358", "359": "+359",
	"370": "+370", "371": "+371", "372": "+372", "373": "+373",
	"374": "+374", "375": "+375", "376": "+376", "377": "+377",
	"378": "+378", "379": "+379", "380": "+380", "381": "+381",
	"382": "+382", "385": "+385", "386": "+386", "387": "+387",
	"389": "+389", "420": "+420", "421": "+421", "423": "+423",
	"500": "+500", "501": "+501", "502": "+502", "503": "+503",
	"504": "+504", "505": "+505", "506": "+506", "507": "+507",
	"508": "+508", "509": "+509", "590": "+590", "591": "+591",
	"592": "+592", "593": "+593", "594": "+594", "595": "+595",
	"596": "+596", "597": "+597", "598": "+598", "599": "+599",
	"670": "+670", "672": "+672", "673": "+673", "674": "+674",
	"675": "+675", "676": "+676", "677": "+677", "678": "+678",
	"679": "+679", "680": "+680", "681": "+681", "682": "+682",
	"683": "+683", "684": "+684", "685": "+685", "686": "+686",
	"687": "+687", "688": "+688", "689": "+689", "690": "+690",
	"691": "+691", "692": "+692", "850": "+850", "852": "+852",
	"853": "+853", "855": "+855", "856": "+856", "880": "+880",
	"886": "+886", "960": "+960", "961": "+961", "962": "+962",
	"963": "+963", "964": "+964", "965": "+965", "966": "+966",
	"967": "+967", "968": "+968", "970": "+970", "971": "+971",
	"972": "+972", "973": "+973", "974": "+974", "975": "+975",
	"976": "+976", "977": "+977", "992": "+992", "993": "+993",
	"994": "+994", "995": "+995", "996": "+996", "998": "+998",
}
