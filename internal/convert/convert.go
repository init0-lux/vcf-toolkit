// Package convert implements CSV to VCF conversion per RFC 6350 (vCard 4.0).
package convert

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/init0-lux/vcf-toolkit/internal/dedupe"
	"github.com/init0-lux/vcf-toolkit/internal/model"
	"github.com/init0-lux/vcf-toolkit/internal/normalize"
)

type Config struct {
	// DefaultCountry is the calling code (e.g. "+91") for phone numbers written
	// without one. When empty, such numbers are written to the vCard as typed.
	DefaultCountry string
	HeaderMapping  map[string]string
	Deduplicate    bool
	Verbose        bool
	Stderr         io.Writer
}

// Field identifiers accepted as values in Config.HeaderMapping.
const (
	FieldName       = "name"
	FieldGivenName  = "given_name"
	FieldMiddleName = "middle_name"
	FieldFamilyName = "family_name"
	FieldPhone      = "phone"
	FieldEmail      = "email"
	FieldOrg        = "org"
	// FieldIgnore drops a column that auto-detection would otherwise map.
	FieldIgnore = "ignore"
)

var knownFields = map[string]bool{
	FieldName: true, FieldGivenName: true, FieldMiddleName: true, FieldFamilyName: true,
	FieldPhone: true, FieldEmail: true, FieldOrg: true, FieldIgnore: true,
}

// ErrNoRecognizedColumns is returned by ParseCSV when no header maps to a
// contact field. Supply Config.HeaderMapping to map columns explicitly.
var ErrNoRecognizedColumns = errors.New("no contact columns recognized in CSV header")

func (cfg Config) logf(format string, args ...interface{}) {
	if cfg.Stderr != nil {
		fmt.Fprintf(cfg.Stderr, format+"\n", args...)
	}
}

type ParseResult struct {
	Contacts []model.Contact
	Errors   []model.ParseError
}

func ParseCSV(r io.Reader, cfg Config) (*ParseResult, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = false
	reader.LazyQuotes = true

	rawHeaders, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV headers: %w", err)
	}

	headers := make([]string, len(rawHeaders))
	for i, h := range rawHeaders {
		headers[i] = strings.TrimSpace(strings.TrimPrefix(h, "\ufeff"))
	}

	for h, f := range cfg.HeaderMapping {
		if !knownFields[f] {
			return nil, fmt.Errorf("invalid field %q for column %q: use one of name, given_name, middle_name, family_name, phone, email, org, ignore", f, h)
		}
	}

	mapping := buildHeaderMapping(headers, cfg.HeaderMapping)
	if len(mapping) == 0 {
		return nil, fmt.Errorf("%w (%s); map columns explicitly, e.g. --map \"%s=name\"",
			ErrNoRecognizedColumns, strings.Join(headers, ", "), headers[0])
	}

	if cfg.Verbose {
		cfg.logf("Detected header mapping:")
		var ignored []string
		for _, h := range headers {
			if f, ok := mapping[h]; ok {
				cfg.logf("  %q -> %s", h, f)
			} else {
				ignored = append(ignored, fmt.Sprintf("%q", h))
			}
		}
		if len(ignored) > 0 {
			cfg.logf("Ignored columns: %s", strings.Join(ignored, ", "))
		}
	}

	result := &ParseResult{}
	line := 2

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, model.ParseError{
				Row:     line,
				Message: fmt.Sprintf("read error: %v", err),
			})
			line++
			continue
		}

		c, errs := contactFromRow(record, headers, mapping)
		for i := range errs {
			errs[i].Row = line
		}
		if len(errs) > 0 {
			result.Errors = append(result.Errors, errs...)
		}

		if c.Name != "" || len(c.Phones) > 0 || len(c.Emails) > 0 || c.Organization != "" {
			c.Source.Row = line
			result.Contacts = append(result.Contacts, c)
		}

		line++
	}

	return result, nil
}

func contactFromRow(row []string, headers []string, mapping map[string]string) (model.Contact, []model.ParseError) {
	var c model.Contact
	var errs []model.ParseError
	seenPhones := make(map[string]bool)
	seenEmails := make(map[string]bool)

	setOnce := func(dst *string, val string) {
		if *dst == "" {
			*dst = val
		}
	}

	for i, header := range headers {
		if i >= len(row) {
			continue
		}
		val := strings.TrimSpace(row[i])
		if val == "" {
			continue
		}

		field, ok := mapping[header]
		if !ok {
			continue
		}

		switch field {
		case FieldName:
			setOnce(&c.Name, val)
		case FieldGivenName:
			setOnce(&c.GivenName, val)
		case FieldMiddleName:
			setOnce(&c.MiddleName, val)
		case FieldFamilyName:
			setOnce(&c.FamilyName, val)
		case FieldPhone:
			for _, p := range splitMultiValue(val) {
				key := normalize.PhoneDigits(p)
				if key == "" {
					key = p
				}
				if !seenPhones[key] {
					seenPhones[key] = true
					c.Phones = append(c.Phones, p)
				}
			}
		case FieldEmail:
			for _, e := range splitMultiValue(val) {
				if key := strings.ToLower(e); !seenEmails[key] {
					seenEmails[key] = true
					c.Emails = append(c.Emails, e)
				}
			}
		case FieldOrg:
			setOnce(&c.Organization, val)
		}
	}

	if c.Name == "" {
		c.Name = joinNonEmpty(c.GivenName, c.MiddleName, c.FamilyName)
	}

	if c.Name == "" && len(c.Phones) == 0 && len(c.Emails) == 0 && c.Organization == "" {
		errs = append(errs, model.ParseError{
			Field:   "row",
			Message: "no recognizable contact fields in row",
		})
	}

	return c, errs
}

// splitMultiValue splits Google Contacts cells that hold several values
// separated by " ::: ".
func splitMultiValue(val string) []string {
	var out []string
	for _, part := range strings.Split(val, ":::") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func joinNonEmpty(parts ...string) string {
	var kept []string
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, " ")
}

func buildHeaderMapping(headers []string, override map[string]string) map[string]string {
	mapping := make(map[string]string, len(headers))

	normalizedOverride := make(map[string]string, len(override))
	for k, v := range override {
		normalizedOverride[strings.ToLower(strings.TrimSpace(k))] = v
	}

	for _, header := range headers {
		lower := strings.ToLower(header)

		if field, ok := normalizedOverride[lower]; ok {
			if field != FieldIgnore {
				mapping[header] = field
			}
			continue
		}

		if field := detectField(lower); field != "" {
			mapping[header] = field
		}
	}

	return mapping
}

var (
	headerWordRe  = regexp.MustCompile(`\p{L}+`)
	headerEmailRe = regexp.MustCompile(`\be[\s_\-]?mail`)
)

// Columns that describe a value (its label or type) or belong to someone
// other than the contact. Exports from Google and Outlook place these next to
// the real value columns.
var skipHeaderWords = wordSet(
	"label", "type", "title", "prefix", "suffix", "phonetic", "yomi", "nickname",
	"fax", "pager", "department", "assistant", "manager", "spouse", "emergency",
	"parent", "guardian", "father", "mother",
)

var (
	emailWords      = wordSet("email", "emails")
	phoneWords      = wordSet("phone", "phones", "telephone", "tel", "mobile", "cell", "cellphone", "whatsapp")
	phoneJoined     = wordSet("phonenumber", "phoneno", "mobilenumber", "mobileno", "contactnumber", "contactno", "cellnumber")
	orgWords        = wordSet("org", "organization", "organisation", "company", "employer", "affiliation", "college", "university", "institution")
	givenWords      = wordSet("first", "given")
	middleWords     = wordSet("middle", "additional")
	familyWords     = wordSet("last", "family", "surname")
	nameWord        = wordSet("name")
	nameJoined      = wordSet("fullname", "displayname", "contactname")
	givenJoined     = wordSet("firstname", "givenname", "fname")
	middleJoined    = wordSet("middlename", "additionalname")
	familyJoined    = wordSet("lastname", "familyname", "lname")
	fullNameContext = wordSet(
		"name", "full", "display", "contact", "your", "student", "participant", "member",
		"person", "candidate", "applicant", "attendee", "of", "the", "s",
	)
)

func wordSet(words ...string) map[string]bool {
	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[w] = true
	}
	return m
}

// detectField maps a CSV header to a contact field by its words, so variants
// like "E-mail 2 Address", "Mobile Phone" or "Phone 1 - Value" are recognized.
func detectField(header string) string {
	h := headerEmailRe.ReplaceAllString(strings.ToLower(header), "email")
	words := headerWordRe.FindAllString(h, -1)
	if len(words) == 0 {
		return ""
	}
	joined := strings.Join(words, "")

	has := func(set map[string]bool) bool {
		for _, w := range words {
			if set[w] {
				return true
			}
		}
		return false
	}
	only := func(set map[string]bool) bool {
		for _, w := range words {
			if !set[w] {
				return false
			}
		}
		return true
	}
	hasName := has(nameWord)
	namePart := func(set, joinedSet map[string]bool) bool {
		return joinedSet[joined] || (has(set) && (hasName || len(words) == 1))
	}

	switch {
	case has(skipHeaderWords):
		return ""
	case (has(emailWords) || strings.HasPrefix(joined, "email") || joined == "mail") && !hasName:
		return FieldEmail
	case has(phoneWords) || phoneJoined[joined]:
		return FieldPhone
	case has(orgWords):
		return FieldOrg
	case namePart(givenWords, givenJoined):
		return FieldGivenName
	case namePart(middleWords, middleJoined):
		return FieldMiddleName
	case namePart(familyWords, familyJoined):
		return FieldFamilyName
	case nameJoined[joined] || (hasName && only(fullNameContext)):
		return FieldName
	}

	return ""
}

// VCFOptions controls how a contact is rendered as a vCard.
type VCFOptions struct {
	// DefaultCountry is the calling code (e.g. "+91") for phone numbers written
	// without one. When empty, such numbers are written as typed.
	DefaultCountry string
}

func ContactToVCF(c model.Contact) string {
	return ContactToVCFWithOptions(c, VCFOptions{})
}

func ContactToVCFWithOptions(c model.Contact, opts VCFOptions) string {
	var b strings.Builder

	b.WriteString("BEGIN:VCARD\r\n")
	b.WriteString("VERSION:4.0\r\n")
	b.WriteString("PRODID:-//vcf-toolkit//EN\r\n")

	fn := c.Name
	if fn == "" {
		fn = "Unknown"
	}
	b.WriteString(fmt.Sprintf("FN:%s\r\n", escapeVCF(fn)))
	b.WriteString(fmt.Sprintf("N:%s\r\n", structuredName(c)))

	if c.Organization != "" {
		b.WriteString(fmt.Sprintf("ORG:%s\r\n", escapeVCF(c.Organization)))
	}

	for _, phone := range c.Phones {
		if tel, ok := internationalPhone(phone, opts.DefaultCountry); ok {
			b.WriteString(fmt.Sprintf("TEL;VALUE=uri:tel:%s\r\n", tel))
		} else {
			b.WriteString(fmt.Sprintf("TEL;VALUE=text:%s\r\n", escapeVCF(strings.TrimSpace(phone))))
		}
	}

	// Emails are written as given; use NormalizeContact to normalize them first.
	for _, email := range c.Emails {
		if email = strings.TrimSpace(email); email != "" {
			b.WriteString(fmt.Sprintf("EMAIL:%s\r\n", escapeVCF(email)))
		}
	}

	b.WriteString("END:VCARD\r\n")
	return b.String()
}

// internationalPhone returns the E.164 form of phone when its country is known.
func internationalPhone(phone, defaultCountry string) (string, bool) {
	n := normalize.NormalizePhone(phone, normalize.PhoneConfig{DefaultCountry: defaultCountry})
	return n.Normalized, n.Valid && strings.HasPrefix(n.Normalized, "+")
}

// writeVCards writes contacts as vCards and returns how many phone numbers had
// no known country and were written as typed.
func writeVCards(w io.Writer, contacts []model.Contact, opts VCFOptions) (localPhones int, err error) {
	for _, c := range contacts {
		for _, p := range c.Phones {
			if _, ok := internationalPhone(p, opts.DefaultCountry); !ok {
				localPhones++
			}
		}
		if _, err := io.WriteString(w, ContactToVCFWithOptions(c, opts)); err != nil {
			return localPhones, fmt.Errorf("writing VCF: %w", err)
		}
	}
	return localPhones, nil
}

func validateCountry(country string) error {
	if country == "" {
		return nil
	}
	if _, ok := normalize.NormalizeCountryCode(country); !ok {
		return fmt.Errorf("invalid default country %q: use a calling code such as +91 or +1", country)
	}
	return nil
}

var (
	nameSuffixes = wordSet("jr", "sr", "ii", "iii", "iv", "phd", "md", "esq")
	companyWords = wordSet("inc", "ltd", "llc", "llp", "corp", "co", "plc", "gmbh")
)

// structuredName builds the vCard N value (family;given;additional;prefix;suffix).
// Separate name columns win; otherwise Name is split, understanding the
// "Family, Given" and "Given Family, Jr." forms.
func structuredName(c model.Contact) string {
	if c.GivenName != "" || c.MiddleName != "" || c.FamilyName != "" {
		return nValue(c.FamilyName, c.GivenName, c.MiddleName, "")
	}

	fullName := strings.TrimSpace(c.Name)
	if fullName == "" {
		return ";;;;"
	}

	var suffix string
	if before, after, ok := strings.Cut(fullName, ","); ok && !strings.Contains(after, ",") {
		before, after = strings.TrimSpace(before), strings.TrimSpace(after)
		word := strings.ToLower(strings.Trim(after, "."))
		switch {
		case before == "" || after == "":
		case companyWords[word]:
			return nValue("", fullName, "", "")
		case nameSuffixes[word]:
			fullName, suffix = before, after
		default:
			return nValue(before, after, "", "")
		}
	}

	parts := strings.Fields(fullName)
	if len(parts) == 1 {
		return nValue("", parts[0], "", suffix)
	}
	middle := strings.Join(parts[1:len(parts)-1], " ")
	return nValue(parts[len(parts)-1], parts[0], middle, suffix)
}

func nValue(family, given, middle, suffix string) string {
	return fmt.Sprintf("%s;%s;%s;;%s", escapeVCF(family), escapeVCF(given), escapeVCF(middle), escapeVCF(suffix))
}

func escapeVCF(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\r\n", "\\n")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\n")
	return s
}

func ConvertCSVToVCF(r io.Reader, w io.Writer, cfg Config) (*ConversionSummary, error) {
	if err := validateCountry(cfg.DefaultCountry); err != nil {
		return nil, err
	}

	result, err := ParseCSV(r, cfg)
	if err != nil {
		return nil, fmt.Errorf("parsing CSV: %w", err)
	}

	if cfg.Verbose {
		cfg.logf("Parsed %d contacts, %d parse errors", len(result.Contacts), len(result.Errors))
	}

	contacts := result.Contacts
	if cfg.Deduplicate && len(contacts) > 0 {
		cfg.logf("Deduplicating %d contacts...", len(contacts))
		contacts = dedupe.Representatives(dedupe.Deduplicate(contacts, dedupe.DefaultConfig()))

		cfg.logf("After deduplication: %d contacts (removed %d duplicates)",
			len(contacts), len(result.Contacts)-len(contacts))
	}

	localPhones, err := writeVCards(w, contacts, VCFOptions{DefaultCountry: cfg.DefaultCountry})
	if err != nil {
		return nil, err
	}

	return &ConversionSummary{
		InputRows:         len(result.Contacts),
		OutputVCards:      len(contacts),
		ParseErrors:       result.Errors,
		Deduplicated:      cfg.Deduplicate,
		InputContacts:     len(result.Contacts),
		DuplicatesRemoved: len(result.Contacts) - len(contacts),
		LocalPhones:       localPhones,
	}, nil
}

type ConversionSummary struct {
	InputRows         int
	OutputVCards      int
	ParseErrors       []model.ParseError
	Deduplicated      bool
	InputContacts     int
	DuplicatesRemoved int
	// LocalPhones counts phone numbers written as typed because their country
	// was unknown. Set DefaultCountry to convert them to international format.
	LocalPhones int
}
