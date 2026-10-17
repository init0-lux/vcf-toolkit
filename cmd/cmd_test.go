package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// run executes the CLI with args, reading input from a temp CSV and returning
// the written output file and stderr.
func run(t *testing.T, csv string, args ...string) (out, stderr string, err error) {
	t.Helper()

	// Flags bind package-level variables that persist between Execute calls.
	outputFile, verbose, jsonOutput, dryRun, dedupeFlag, stripAliases = "", false, false, false, false, false
	headerMap, defaultCountry = nil, ""

	dir := t.TempDir()
	in := filepath.Join(dir, "in.csv")
	outPath := filepath.Join(dir, "out")
	require.NoError(t, os.WriteFile(in, []byte(csv), 0o600))

	var errBuf bytes.Buffer
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs(append([]string{args[0], in, "--out", outPath}, args[1:]...))
	err = rootCmd.Execute()

	b, _ := os.ReadFile(outPath)
	return string(b), errBuf.String(), err
}

func TestConvert_CountryFlag(t *testing.T) {
	out, stderr, err := run(t, "name,phone\nPriya,98765 43210\n", "convert", "--country", "+91")
	require.NoError(t, err)
	assert.Contains(t, out, "TEL;VALUE=uri:tel:+919876543210\r\n")
	assert.Empty(t, stderr)
}

func TestConvert_WarnsAboutLocalPhones(t *testing.T) {
	out, stderr, err := run(t, "name,phone\nPriya,98765 43210\n", "convert")
	require.NoError(t, err)
	assert.Contains(t, out, "TEL;VALUE=text:98765 43210\r\n")
	assert.Contains(t, stderr, "1 phone number(s) have no country code")
}

func TestConvert_InvalidCountry(t *testing.T) {
	_, _, err := run(t, "name,phone\nPriya,98765 43210\n", "convert", "--country", "IN")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid --country "IN"`)
}

func TestConvert_UnrecognizedColumnsSuggestMap(t *testing.T) {
	_, _, err := run(t, "Who,Digits\nAl,+12125550100\n", "convert")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--map")
}

func TestConvert_MapFlag(t *testing.T) {
	out, _, err := run(t, "Who,Digits\nAl,+12125550100\n", "convert", "--map", "Who=name", "--map", "Digits=phone")
	require.NoError(t, err)
	assert.Contains(t, out, "FN:Al\r\n")
	assert.Contains(t, out, "TEL;VALUE=uri:tel:+12125550100\r\n")
}

func TestNormalize_StripAliasesIsOptIn(t *testing.T) {
	csv := "name,email\nAl,Al+news@X.com\n"

	out, _, err := run(t, csv, "normalize")
	require.NoError(t, err)
	assert.Contains(t, out, "al+news@x.com")

	out, _, err = run(t, csv, "normalize", "--strip-aliases")
	require.NoError(t, err)
	assert.Contains(t, out, "al@x.com")
}

func TestDedupe_SharedPhoneDifferentPeople(t *testing.T) {
	csv := "name,phone,org\nFront Desk,+12125550100,Acme\nJohn Doe,+12125550100,Acme\nJohn Doe,+12125550100,Acme\n"
	out, _, err := run(t, csv, "dedupe")
	require.NoError(t, err)
	assert.Contains(t, out, "Front Desk")
	assert.Equal(t, 3, bytes.Count([]byte(out), []byte("\n")), "header plus two contacts:\n%s", out)
}
