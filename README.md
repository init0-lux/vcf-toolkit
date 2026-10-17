# vcf-toolkit

`vcf-toolkit` is a Go-native toolkit for cleaning up contact data, deduplicating records, and converting CSV contacts into VCF/vCard output.

It ships as:

- a CLI for day-to-day use
- importable Go packages for SDK-style integration

The library is designed around deterministic, local processing by default. External LLM-based name normalization is opt-in.

## What it does

- Normalize names, phone numbers, and email addresses
- Detect and merge duplicate contacts
- Convert CSV contacts to vCard 4.0 (`.vcf`), including Google Contacts and Outlook exports
- Parse VCF back into contacts (Go API)
- Provide a terminal UI for interactive workflows

## Requirements

- Go 1.24.2 or newer

## Install

### npx (no install)

```bash
npx @init0-lux/vcf-toolkit --help
```

The npm package downloads the matching release binary for your platform on first run.

### Go

```bash
go install github.com/init0-lux/vcf-toolkit@latest
```

This installs the `vcf-toolkit` binary if your `GOBIN` or `GOPATH/bin` is on `PATH`.

### Prebuilt binaries

Download an archive for Linux, macOS or Windows from the [releases page](https://github.com/init0-lux/vcf-toolkit/releases).

### From source

```bash
git clone https://github.com/init0-lux/vcf-toolkit.git
cd vcf-toolkit
go build ./...
```

## CLI usage

Show help:

```bash
vcf-toolkit --help
vcf-toolkit --version
vcf-toolkit convert --help
vcf-toolkit dedupe --help
vcf-toolkit normalize --help
vcf-toolkit tui
```

### Convert CSV to VCF

```bash
vcf-toolkit convert contacts.csv > contacts.vcf
cat contacts.csv | vcf-toolkit convert > contacts.vcf
vcf-toolkit convert contacts.csv --out contacts.vcf
vcf-toolkit convert contacts.csv --dedupe --out contacts.vcf
vcf-toolkit convert registrations.csv --country +91 --out contacts.vcf
```

Pass `--country` with the calling code your numbers belong to when they are written without one (for example `98765 43210` or `(212) 555-0101`). See [Phone numbers](#phone-numbers).

### Deduplicate CSV contacts

```bash
vcf-toolkit dedupe contacts.csv > deduped.csv
vcf-toolkit dedupe contacts.csv --json > dedupe-report.json
vcf-toolkit dedupe contacts.csv --dry-run
```

### Normalize CSV contacts

```bash
vcf-toolkit normalize contacts.csv > normalized.csv
vcf-toolkit normalize contacts.csv --json > normalized.json
vcf-toolkit normalize contacts.csv --country +1 --strip-aliases
```

`--strip-aliases` turns `user+tag@example.com` into `user@example.com`. It is off by default because the alias is part of the address the contact gave you.

### Interactive TUI

```bash
vcf-toolkit tui
VCF_TOOLKIT_COUNTRY=+91 vcf-toolkit tui
```

Set `VCF_TOOLKIT_COUNTRY` to give the TUI a default calling code.

## Go SDK usage

Import the public packages from the module root:

- [`convert`](./convert)
- [`dedupe`](./dedupe)
- [`normalize`](./normalize)
- [`vcf`](./vcf)
- [`model`](./model)

### Convert CSV to VCF

```go
package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/init0-lux/vcf-toolkit/convert"
)

func main() {
	csvData := "name,phone,email,org\nAlice,+1 212 555 0101,alice@example.com,Acme\n"

	var out bytes.Buffer
	summary, err := convert.ConvertCSVToVCF(
		strings.NewReader(csvData),
		&out,
		convert.Config{
			Deduplicate: true,
		},
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(out.String())
	fmt.Printf("contacts in: %d, vcards out: %d\n", summary.InputContacts, summary.OutputVCards)
}
```

### Deduplicate contacts

```go
package main

import (
	"fmt"

	"github.com/init0-lux/vcf-toolkit/dedupe"
	"github.com/init0-lux/vcf-toolkit/model"
)

func main() {
	contacts := []model.Contact{
		{Name: "Alice", Emails: []string{"alice@example.com"}},
		{Name: "Alice Smith", Emails: []string{"Alice@Example.com"}},
	}

	result := dedupe.Deduplicate(contacts, dedupe.DefaultConfig())
	fmt.Printf("clusters: %d, duplicates: %d\n", len(result.Clusters), result.Report.DuplicatesFound)
}
```

### Normalize primitives

```go
package main

import (
	"fmt"

	"github.com/init0-lux/vcf-toolkit/normalize"
)

func main() {
	name := normalize.NormalizeName("John Doe Inc.", normalize.NameConfig{
		StripSuffix: true,
		OrgPatterns: []string{"Inc."},
	})

	phone := normalize.NormalizePhone("(212) 555-0101", normalize.PhoneConfig{
		DefaultCountry:   "+1",
		StrictValidation: true,
	})

	email := normalize.NormalizeEmail("User+tag@Example.com", normalize.EmailConfig{
		StripAliases: true,
	})

	fmt.Println(name.FullName)
	fmt.Println(phone.Normalized)
	fmt.Println(email.Normalized)
}
```

### Parse VCF

```go
package main

import (
	"fmt"
	"os"

	"github.com/init0-lux/vcf-toolkit/vcf"
)

func main() {
	res, err := vcf.ParseVCF(os.Stdin)
	if err != nil {
		panic(err)
	}
	fmt.Println(len(res.Contacts))
}
```

## CSV input expectations

Every command expects a header row. Headers are matched by their words, so exports from Google Contacts (both the current and legacy formats), Outlook and most sign-up forms work without configuration.

Recognized columns include:

- name: `Name`, `Full Name`, `Display Name`, `Your Name`, or separate `First Name` / `Middle Name` / `Last Name` (also `Given Name`, `Additional Name`, `Family Name`, `Surname`)
- phone: `Phone`, `Mobile`, `Mobile Phone`, `Business Phone`, `Phone 1 - Value`, `Contact Number`, `WhatsApp Number`
- email: `Email`, `E-mail Address`, `E-mail 2 Address`, `E-mail 1 - Value`, `Primary Email`
- organization: `Org`, `Company`, `Organization Name`, `Organization 1 - Name`, `College`

Label, type, title, fax, pager, nickname and phonetic columns are skipped, as are numbers that belong to someone else (assistant, manager, emergency contact, parent). Every phone and email column is kept, and Google's `value ::: value` cells are split.

If no column is recognized the command exits with an error instead of producing empty output. Use `--verbose` to see how each column was mapped and which were ignored.

Map custom headers with `--map` (repeatable). Fields are `name`, `given_name`, `middle_name`, `family_name`, `phone`, `email`, `org` and `ignore`:

```bash
vcf-toolkit convert export.csv --map "Primary Contact=phone" --map "Team Name=ignore"
```

The same mapping is available in the Go API:

```go
cfg := convert.Config{
	HeaderMapping: map[string]string{
		"Full Name":     convert.FieldName,
		"Primary Phone": convert.FieldPhone,
		"Work Email":    convert.FieldEmail,
	},
}
```

## Output formats

### VCF

- Writes vCard 4.0 records
- Supports names, phones, emails, and organization fields
- Preserves multiple phone and email values per contact
- Builds the structured `N` field from separate name columns when present, and understands `Family, Given` and `Given Family, Jr.`
- Writes phone numbers with a known country as `tel:` URIs in E.164 form and all others as typed
- Writes email addresses exactly as given

### CSV

- Normalized CSV output uses the columns `name`, `phone`, `email`, `org`
- Multiple values are joined with `; `

### JSON

- Available from the CLI for `normalize` and `dedupe`
- Useful for piping into other tools or for inspection

## Determinism and behavior

Default behavior is deterministic for the same input and configuration:

- parsing is sequential
- deduplication output order is stable
- VCF output preserves the processed contact order

External LLM-based name normalization is opt-in and not part of the default deterministic path. If you enable it, output quality and exact wording depend on your configured client.

## Deduplication model

The dedupe engine uses a weighted score across:

- exact email match, including alias normalization
- exact phone match after digit cleanup
- fuzzy name similarity using Jaro-Winkler (counted only at 0.80 or above, so a shared office phone or team inbox does not merge different people)
- organization match

Default settings are tuned for practical CSV cleanup, not strict identity resolution. Preview merges with `--dry-run` and tune them with `--threshold`, `--min-name-sim` and the weight flags. For high-volume or high-risk data, validate results on a sample before using the output downstream.

## Phone numbers

A phone number becomes international (E.164) when its country is known:

- it starts with `+` or `00`, or
- you pass `--country` (Go: `Config.DefaultCountry`), e.g. `--country +91`

With `--country +91`, `98765 43210`, `098765 43210` and `919876543210` all become `+919876543210`. A leading `0` is treated as a national trunk prefix. For `+1` and `+91` numbers must have exactly 10 national digits; anything else is kept as typed.

Without a known country, a number is never given one. It is written as typed (`TEL;VALUE=text:...`) and the CLI prints a warning suggesting `--country`.

Phone handling is heuristic and is not a full international numbering library. If you need strict telecom-grade validation, wrap this package with a dedicated phone-number library in your application.

## LLM-based name normalization

`normalize.HTTPNameLLM` lets you plug in a provider-agnostic HTTP endpoint that returns JSON name metadata.

Use this only when you explicitly want non-local normalization:

```go
cfg := normalize.NameConfig{
	UseLLM: true,
	LLMClient: &normalize.HTTPNameLLM{
		URL: "https://your-endpoint.example/normalize-name",
	},
}
```

If `UseLLM` is false, normalization stays local and deterministic.

## Development

Run tests:

```bash
make test
```

Build the CLI into `bin/vcf-toolkit` (version taken from `git describe`):

```bash
make build
```

## License

MIT. See [LICENSE](./LICENSE).

## Repository layout

- `cmd/`: CLI subcommands
- `convert/`, `dedupe/`, `normalize/`, `vcf/`, `model/`: public Go wrappers
- `internal/`: core implementation
- `tui/`: terminal UI
- `testdata/`: sample CSV and VCF fixtures

## Notes for consumers

- Import the public wrapper packages, not the `internal/...` packages.
- The root module is a CLI entrypoint, so the package at the repository root is `main`.
- If you need stable machine-to-machine output, keep LLM normalization disabled.

