package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/init0-lux/vcf-toolkit/internal/dedupe"
	"github.com/init0-lux/vcf-toolkit/internal/model"
	"github.com/spf13/cobra"
)

var dedupeCmd = &cobra.Command{
	Use:   "dedupe [file]",
	Short: "Find and merge duplicate contacts",
	Long: `Find and merge duplicate contacts using multiple signals.

Uses four matching signals to identify duplicates:
  - Email (exact match with alias normalization)
  - Phone (digits-only exact match)
  - Name (fuzzy match via Jaro-Winkler similarity)
  - Organization (exact + substring match)

Results are output as a CSV of unique contacts, or as JSON
clusters when --json is used. Use --dry-run to preview
clusters without merging.

Examples:
  vcf-toolkit dedupe contacts.csv
  cat contacts.csv | vcf-toolkit dedupe --out clean.csv
  vcf-toolkit dedupe contacts.csv --json
  vcf-toolkit dedupe contacts.csv --dry-run
`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDedupe,
}

type dedupeFlags struct {
	emailWeight float64
	phoneWeight float64
	nameWeight  float64
	orgWeight   float64
	threshold   float64
	minNameSim  float64
}

var dFlags dedupeFlags

func init() {
	rootCmd.AddCommand(dedupeCmd)

	dedupeCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "output as JSON")
	dedupeCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "preview clusters without writing output")
	addCSVFlags(dedupeCmd)

	defaults := dedupe.DefaultConfig()
	dedupeCmd.Flags().Float64VarP(&dFlags.emailWeight, "email-weight", "", defaults.EmailExactWeight, "email match weight (0.0–1.0)")
	dedupeCmd.Flags().Float64VarP(&dFlags.phoneWeight, "phone-weight", "", defaults.PhoneExactWeight, "phone match weight (0.0–1.0)")
	dedupeCmd.Flags().Float64VarP(&dFlags.nameWeight, "name-weight", "", defaults.NameFuzzyWeight, "name fuzzy match weight (0.0–1.0)")
	dedupeCmd.Flags().Float64VarP(&dFlags.orgWeight, "org-weight", "", defaults.OrganizationWeight, "organization match weight (0.0–1.0)")
	dedupeCmd.Flags().Float64VarP(&dFlags.threshold, "threshold", "t", defaults.Threshold, "deduplication threshold (0.0–1.0)")
	dedupeCmd.Flags().Float64VarP(&dFlags.minNameSim, "min-name-sim", "", defaults.MinNameSimilarity, "minimum name similarity (0.0–1.0)")
}

func runDedupe(cmd *cobra.Command, args []string) error {
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

	parsed, err := parseCSVInput(cmd, r)
	if err != nil {
		return fmt.Errorf("parsing contacts: %w", err)
	}
	contacts := parsed.Contacts
	if len(contacts) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "no contacts found")
		return nil
	}

	if verbose {
		fmt.Fprintf(cmd.ErrOrStderr(), "Parsed %d contacts\n", len(contacts))
	}

	cfg := dedupe.Config{
		EmailExactWeight:   dFlags.emailWeight,
		PhoneExactWeight:   dFlags.phoneWeight,
		NameFuzzyWeight:    dFlags.nameWeight,
		OrganizationWeight: dFlags.orgWeight,
		Threshold:          dFlags.threshold,
		MinNameSimilarity:  dFlags.minNameSim,
	}

	result := dedupe.Deduplicate(contacts, cfg)

	if verbose {
		fmt.Fprintf(cmd.ErrOrStderr(), "Duplicates found: %d\n", result.Report.DuplicatesFound)
		fmt.Fprintf(cmd.ErrOrStderr(), "Clusters formed: %d\n", result.Report.ClustersFormed)
	}

	if dryRun {
		return outputDryRun(cmd, result)
	}
	if jsonOutput {
		return outputJSON(w, result)
	}
	return outputMergedCSV(w, result)
}

func outputDryRun(cmd *cobra.Command, result model.DedupeResult) error {
	fmt.Fprintf(cmd.ErrOrStderr(), "\nDeduplication Report:\n")
	fmt.Fprintf(cmd.ErrOrStderr(), "  Total input contacts:  %d\n", result.Report.TotalInput)
	fmt.Fprintf(cmd.ErrOrStderr(), "  Duplicates found:      %d\n", result.Report.DuplicatesFound)
	fmt.Fprintf(cmd.ErrOrStderr(), "  Clusters formed:       %d\n", result.Report.ClustersFormed)
	fmt.Fprintf(cmd.ErrOrStderr(), "\n")

	for i, cluster := range result.Clusters {
		if len(cluster) < 2 {
			continue
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Cluster %d (%d contacts):\n", i+1, len(cluster))
		for _, c := range cluster {
			fmt.Fprintf(cmd.ErrOrStderr(), "  - %s | %s | %s\n",
				c.Name,
				strings.Join(c.Emails, ", "),
				strings.Join(c.Phones, "; "),
			)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "\n")
	}
	return nil
}

type dedupeOutput struct {
	Report   model.DedupeReport    `json:"report"`
	Clusters [][]dedupeContact     `json:"clusters,omitempty"`
	Merged   []model.MergedContact `json:"merged,omitempty"`
}

type dedupeContact struct {
	Name   string   `json:"name"`
	Phones []string `json:"phones,omitempty"`
	Emails []string `json:"emails,omitempty"`
	Org    string   `json:"org,omitempty"`
	Source string   `json:"source,omitempty"`
}

func toDedupeClusters(clusters [][]model.Contact) [][]dedupeContact {
	var result [][]dedupeContact
	for _, cluster := range clusters {
		if len(cluster) < 2 {
			continue
		}
		dc := make([]dedupeContact, 0, len(cluster))
		for _, c := range cluster {
			source := fmt.Sprintf("%s:%d", c.Source.File, c.Source.Row)
			if source == ":0" {
				source = ""
			}
			dc = append(dc, dedupeContact{
				Name:   c.Name,
				Phones: c.Phones,
				Emails: c.Emails,
				Org:    c.Organization,
				Source: source,
			})
		}
		result = append(result, dc)
	}
	return result
}

func outputJSON(w io.Writer, result model.DedupeResult) error {
	out := dedupeOutput{
		Report:   result.Report,
		Clusters: toDedupeClusters(result.Clusters),
		Merged:   result.Merged,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func outputMergedCSV(w io.Writer, result model.DedupeResult) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"name", "phone", "email", "org"}); err != nil {
		return err
	}
	for _, c := range dedupe.Representatives(result) {
		if err := writer.Write([]string{
			c.Name,
			strings.Join(c.Phones, "; "),
			strings.Join(c.Emails, "; "),
			c.Organization,
		}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
