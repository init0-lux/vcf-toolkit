package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/init0-lux/vcf-toolkit/internal/convert"
	"github.com/spf13/cobra"
)

var headerMap map[string]string

// addCSVFlags registers flags shared by every command that reads CSV input.
func addCSVFlags(c *cobra.Command) {
	c.Flags().StringToStringVar(&headerMap, "map", nil,
		`map a CSV column to a field, e.g. --map "Primary Phone=phone" (fields: name, given_name, middle_name, family_name, phone, email, org, ignore)`)
}

// parseCSVInput parses CSV contacts with the shared header detection and
// reports per-row problems on stderr in verbose mode.
func parseCSVInput(cmd *cobra.Command, r io.Reader) (*convert.ParseResult, error) {
	result, err := convert.ParseCSV(r, convert.Config{
		HeaderMapping: headerMap,
		Verbose:       verbose,
		Stderr:        cmd.ErrOrStderr(),
	})
	if err != nil {
		return nil, err
	}
	if verbose {
		for _, pe := range result.Errors {
			fmt.Fprintf(cmd.ErrOrStderr(), "row %d: %s\n", pe.Row, pe.Message)
		}
	}
	return result, nil
}

// openInput resolves the input source from CLI args or defaults to stdin.
func openInput(args []string) (io.ReadCloser, error) {
	if len(args) > 0 && args[0] != "-" {
		return os.Open(args[0])
	}
	return os.Stdin, nil
}

// openOutput resolves the output destination from --out flag or defaults to stdout.
func openOutput() (io.WriteCloser, error) {
	if outputFile != "" {
		return os.Create(outputFile)
	}
	return os.Stdout, nil
}
