package cmd

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"strings"

	"github.com/init0-lux/vcf-toolkit/internal/convert"
	"github.com/init0-lux/vcf-toolkit/internal/normalize"
	"github.com/spf13/cobra"
)

var normalizeCmd = &cobra.Command{
	Use:   "normalize [file]",
	Short: "Normalize contact data (names, phones, emails)",
	Long: `Normalize contact data from a CSV file or stdin.

Reads a CSV with columns for name, phone, email, and org headers
and outputs normalized data in CSV or JSON format.

Examples:
  vcf-toolkit normalize contacts.csv
  cat contacts.csv | vcf-toolkit normalize
  vcf-toolkit normalize contacts.csv --out clean.csv
  vcf-toolkit normalize contacts.csv --json
  vcf-toolkit normalize contacts.csv --country +91
`,
	Args: cobra.MaximumNArgs(1),
	RunE: runNormalize,
}

func init() {
	rootCmd.AddCommand(normalizeCmd)
	normalizeCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "output as JSON")
	addCSVFlags(normalizeCmd)
	addCountryFlag(normalizeCmd)
}

func runNormalize(cmd *cobra.Command, args []string) error {
	country, err := resolveCountry()
	if err != nil {
		return err
	}

	r, err := openInput(args)
	if err != nil {
		return err
	}
	defer r.Close()

	w, err := openOutput()
	if err != nil {
		return err
	}
	defer w.Close()

	result, err := parseCSVInput(cmd, r)
	if err != nil {
		return err
	}

	var contacts []normalizedContact
	localPhones := 0
	for _, c := range result.Contacts {
		n := convert.NormalizeContact(c, convert.NormalizeVCFConfig{
			Phone: normalize.PhoneConfig{DefaultCountry: country},
			Email: normalize.EmailConfig{StripAliases: true},
		})
		for _, p := range n.Phones {
			if !strings.HasPrefix(p, "+") {
				localPhones++
			}
		}
		contacts = append(contacts, normalizedContact{
			Name:   n.Name,
			Phones: n.Phones,
			Emails: n.Emails,
			Org:    n.Organization,
		})
	}

	warnLocalPhones(cmd, localPhones)
	if jsonOutput {
		return writeJSON(w, contacts)
	}
	return writeCSV(w, contacts)
}

type normalizedContact struct {
	Name   string   `json:"name"`
	Phones []string `json:"phones"`
	Emails []string `json:"emails"`
	Org    string   `json:"org"`
}

func writeJSON(w io.Writer, contacts []normalizedContact) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(contacts)
}

func writeCSV(w io.Writer, contacts []normalizedContact) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write([]string{"name", "phone", "email", "org"}); err != nil {
		return err
	}

	for _, c := range contacts {
		if err := writer.Write([]string{
			c.Name,
			strings.Join(c.Phones, "; "),
			strings.Join(c.Emails, "; "),
			c.Org,
		}); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}
