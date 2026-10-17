package convert

import (
	"io"

	internal "github.com/init0-lux/vcf-toolkit/internal/convert"
	"github.com/init0-lux/vcf-toolkit/model"
)

// Public SDK wrapper around the internal implementation.

type Config = internal.Config
type ParseResult = internal.ParseResult
type ConversionSummary = internal.ConversionSummary
type NormalizeVCFConfig = internal.NormalizeVCFConfig
type VCFOptions = internal.VCFOptions

const (
	FieldName       = internal.FieldName
	FieldGivenName  = internal.FieldGivenName
	FieldMiddleName = internal.FieldMiddleName
	FieldFamilyName = internal.FieldFamilyName
	FieldPhone      = internal.FieldPhone
	FieldEmail      = internal.FieldEmail
	FieldOrg        = internal.FieldOrg
	FieldIgnore     = internal.FieldIgnore
)

var ErrNoRecognizedColumns = internal.ErrNoRecognizedColumns

var ContactToVCF = internal.ContactToVCF
var ContactToVCFWithOptions = internal.ContactToVCFWithOptions

func NormalizeContact(c Contact, cfg NormalizeVCFConfig) Contact {
	return internal.NormalizeContact(c, cfg)
}

func ParseCSV(r io.Reader, cfg Config) (*ParseResult, error) { return internal.ParseCSV(r, cfg) }

func ConvertCSVToVCF(r io.Reader, w io.Writer, cfg Config) (*ConversionSummary, error) {
	return internal.ConvertCSVToVCF(r, w, cfg)
}

func NormalizeCSVToVCF(r io.Reader, w io.Writer, cfg NormalizeVCFConfig) (*ConversionSummary, error) {
	return internal.NormalizeCSVToVCF(r, w, cfg)
}

// Re-export for convenience.
type Contact = model.Contact
