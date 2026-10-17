package dedupe

import (
	internal "github.com/init0-lux/vcf-toolkit/internal/dedupe"
	"github.com/init0-lux/vcf-toolkit/model"
)

// Public SDK wrapper around the internal implementation.

type Config = internal.Config

func DefaultConfig() Config { return internal.DefaultConfig() }

// Representatives returns one contact per cluster in input order: singletons
// as they are and each duplicate cluster as its merged contact.
func Representatives(result model.DedupeResult) []model.Contact {
	return internal.Representatives(result)
}

func Deduplicate(contacts []model.Contact, cfg Config) model.DedupeResult {
	// model.Contact is an alias over internal/model.Contact, so this is safe.
	return internal.Deduplicate(contacts, cfg)
}
