package convert

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseFixture(t *testing.T, name string) *ParseResult {
	t.Helper()
	f, err := os.Open("../../testdata/" + name)
	require.NoError(t, err)
	defer f.Close()
	result, err := ParseCSV(f, Config{})
	require.NoError(t, err)
	return result
}

func TestParseCSV_HeadersVariantsFixture(t *testing.T) {
	result := parseFixture(t, "headers_variants.csv")
	require.Len(t, result.Contacts, 3)

	c := result.Contacts[0]
	assert.Equal(t, "Alice Johnson", c.Name)
	assert.Equal(t, []string{"+1-212-555-0100"}, c.Phones)
	assert.Equal(t, []string{"alice.j@company.com"}, c.Emails)
	assert.Equal(t, "Acme Corp", c.Organization)
}

func TestParseCSV_OutlookExport(t *testing.T) {
	result := parseFixture(t, "outlook.csv")
	require.Len(t, result.Contacts, 2)

	alice := result.Contacts[0]
	assert.Equal(t, "Alice Marie Johnson", alice.Name)
	assert.Equal(t, "Alice", alice.GivenName)
	assert.Equal(t, "Marie", alice.MiddleName)
	assert.Equal(t, "Johnson", alice.FamilyName)
	assert.Equal(t, []string{"alice.j@company.com", "alice@home.org"}, alice.Emails)
	// Fax, pager and assistant's phone are not the contact's own numbers.
	assert.Equal(t, []string{"+1 212 555 0100", "+1 917 555 0142"}, alice.Phones)
	assert.Equal(t, "Acme Corp", alice.Organization)

	bob := result.Contacts[1]
	assert.Equal(t, "Bob Smith", bob.Name)
	assert.Equal(t, []string{"+1 415 555 0199"}, bob.Phones)
	assert.Equal(t, "Globex Inc", bob.Organization)
}

func TestParseCSV_GoogleExport(t *testing.T) {
	result := parseFixture(t, "google_contacts.csv")
	require.Len(t, result.Contacts, 2)

	priya := result.Contacts[0]
	assert.Equal(t, "Priya Sharma", priya.Name)
	assert.Equal(t, "VIT", priya.Organization)
	assert.Equal(t, []string{"priya@gmail.com"}, priya.Emails)
	// " ::: " separates multiple values inside one Google cell.
	assert.Equal(t, []string{"+91 98765 43210", "+91 91234 56789", "+91 44 2345 6789"}, priya.Phones)

	rahul := result.Contacts[1]
	assert.Equal(t, "Rahul Kumar Verma", rahul.Name)
	assert.Equal(t, []string{"rahul.verma@example.in", "rahul@home.in"}, rahul.Emails)
	assert.Empty(t, rahul.Organization)
}

func TestParseCSV_GoogleLegacyExport(t *testing.T) {
	result := parseFixture(t, "google_contacts_legacy.csv")
	require.Len(t, result.Contacts, 1)

	c := result.Contacts[0]
	assert.Equal(t, "Jane Q Public", c.Name)
	assert.Equal(t, "Jane", c.GivenName)
	assert.Equal(t, "Q", c.MiddleName)
	assert.Equal(t, "Public", c.FamilyName)
	assert.Equal(t, []string{"jane@public.org"}, c.Emails)
	assert.Equal(t, []string{"+44 7700 900123"}, c.Phones)
	assert.Equal(t, "Public Ltd", c.Organization)
}

func TestParseCSV_FullNameColumnWinsOverParts(t *testing.T) {
	input := "Given Name,Family Name,Name\nJohn,Doe,Johnny Doe\n"
	result, err := ParseCSV(strings.NewReader(input), Config{})
	require.NoError(t, err)
	require.Len(t, result.Contacts, 1)
	assert.Equal(t, "Johnny Doe", result.Contacts[0].Name)
	assert.Equal(t, "John", result.Contacts[0].GivenName)
}

func TestParseCSV_DuplicateValuesInRow(t *testing.T) {
	input := "name,Mobile Phone,Primary Phone,email,Email 2\nAl,+1 212 555 0100,(+1) 212-555-0100,Al@X.com,al@x.com\n"
	result, err := ParseCSV(strings.NewReader(input), Config{})
	require.NoError(t, err)
	require.Len(t, result.Contacts, 1)
	assert.Equal(t, []string{"+1 212 555 0100"}, result.Contacts[0].Phones)
	assert.Equal(t, []string{"Al@X.com"}, result.Contacts[0].Emails)
}

func TestParseCSV_ByteOrderMark(t *testing.T) {
	// Excel and Outlook prefix CSVs with a BOM; overrides must still match the
	// first column.
	input := "\ufeffName,Email\nAcme,al@x.com\n"
	result, err := ParseCSV(strings.NewReader(input), Config{
		HeaderMapping: map[string]string{"Name": FieldOrg},
	})
	require.NoError(t, err)
	require.Len(t, result.Contacts, 1)
	assert.Equal(t, "Acme", result.Contacts[0].Organization)
	assert.Empty(t, result.Contacts[0].Name)
}

func TestParseCSV_InvalidHeaderMapping(t *testing.T) {
	input := "Col\nx\n"
	_, err := ParseCSV(strings.NewReader(input), Config{
		HeaderMapping: map[string]string{"Col": "telephone"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"telephone"`)
}

func TestParseCSV_IgnoreMapping(t *testing.T) {
	input := "Name,Team Name,Your Name\nTeam Rocket,Rocket,Jessie\n"
	result, err := ParseCSV(strings.NewReader(input), Config{
		HeaderMapping: map[string]string{"name": FieldIgnore},
	})
	require.NoError(t, err)
	require.Len(t, result.Contacts, 1)
	assert.Equal(t, "Jessie", result.Contacts[0].Name)
}

func TestParseCSV_VerboseListsIgnoredColumns(t *testing.T) {
	var stderr strings.Builder
	input := "name,notes\nAl,hi\n"
	_, err := ParseCSV(strings.NewReader(input), Config{Verbose: true, Stderr: &stderr})
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), `Ignored columns: "notes"`)
}
