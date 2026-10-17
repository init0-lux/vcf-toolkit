package dedupe

import (
	"testing"

	"github.com/init0-lux/vcf-toolkit/internal/model"
	"github.com/init0-lux/vcf-toolkit/internal/normalize"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Threshold != 0.50 {
		t.Errorf("expected threshold 0.50, got %f", cfg.Threshold)
	}
	if cfg.EmailExactWeight != 0.40 {
		t.Errorf("expected email weight 0.40, got %f", cfg.EmailExactWeight)
	}
	if cfg.MinNameSimilarity != 0.80 {
		t.Errorf("expected min name similarity 0.80, got %f", cfg.MinNameSimilarity)
	}
}

func TestDeduplicate_Empty(t *testing.T) {
	result := Deduplicate([]model.Contact{}, DefaultConfig())
	if result.Report.TotalInput != 0 {
		t.Errorf("expected 0 contacts, got %d", result.Report.TotalInput)
	}
	if len(result.Clusters) != 0 {
		t.Errorf("expected empty clusters, got %d", len(result.Clusters))
	}
}

func TestDeduplicate_NoDuplicates(t *testing.T) {
	contacts := []model.Contact{
		{Name: "Alice", Phones: []string{"+1-212-555-0101"}, Emails: []string{"alice@example.com"}},
		{Name: "Bob", Phones: []string{"+1-212-555-0102"}, Emails: []string{"bob@example.com"}},
		{Name: "Charlie", Phones: []string{"+1-212-555-0103"}, Emails: []string{"charlie@example.com"}},
	}

	result := Deduplicate(contacts, DefaultConfig())

	if result.Report.TotalInput != 3 {
		t.Errorf("expected 3 total, got %d", result.Report.TotalInput)
	}
	if result.Report.DuplicatesFound != 0 {
		t.Errorf("expected 0 duplicates, got %d", result.Report.DuplicatesFound)
	}
	if len(result.Clusters) != 3 {
		t.Errorf("expected 3 clusters, got %d", len(result.Clusters))
	}
}

func TestDeduplicate_ByEmail(t *testing.T) {
	contacts := []model.Contact{
		{Name: "Alice Smith", Emails: []string{"alice@example.com"}},
		{Name: "Alice", Emails: []string{"Alice@Example.com"}},
		{Name: "Bob", Emails: []string{"bob@example.com"}},
	}

	cfg := DefaultConfig()
	cfg.EmailExactWeight = 1.0
	cfg.PhoneExactWeight = 0.0
	cfg.NameFuzzyWeight = 0.0
	cfg.OrganizationWeight = 0.0
	cfg.Threshold = 0.5

	result := Deduplicate(contacts, cfg)

	if result.Report.DuplicatesFound != 1 {
		t.Errorf("expected 1 duplicate (Alice/Alice), got %d", result.Report.DuplicatesFound)
	}
	if len(result.Clusters) != 2 {
		t.Errorf("expected 2 clusters, got %d", len(result.Clusters))
	}
}

func TestDeduplicate_ByPhone(t *testing.T) {
	contacts := []model.Contact{
		{Name: "Alice", Phones: []string{"+1 (212) 555-0101"}},
		{Name: "Alice Smith", Phones: []string{"+1-212-555-0101"}},
		{Name: "Bob", Phones: []string{"+1-212-555-0102"}},
	}

	cfg := DefaultConfig()
	cfg.PhoneExactWeight = 1.0
	cfg.EmailExactWeight = 0.0
	cfg.NameFuzzyWeight = 0.0
	cfg.OrganizationWeight = 0.0
	cfg.Threshold = 0.5

	result := Deduplicate(contacts, cfg)

	if result.Report.DuplicatesFound != 1 {
		t.Errorf("expected 1 duplicate (by phone), got %d", result.Report.DuplicatesFound)
	}
}

func TestDeduplicate_ByName(t *testing.T) {
	contacts := []model.Contact{
		{Name: "Alice Smith"},
		{Name: "Alice Smithson", Organization: "Corp A"},
		{Name: "Bob Jones"},
	}

	cfg := DefaultConfig()
	cfg.NameFuzzyWeight = 1.0
	cfg.EmailExactWeight = 0.0
	cfg.PhoneExactWeight = 0.0
	cfg.OrganizationWeight = 0.0
	cfg.MinNameSimilarity = 0.5
	cfg.Threshold = 0.5

	result := Deduplicate(contacts, cfg)

	if result.Report.DuplicatesFound < 1 {
		t.Errorf("expected at least 1 duplicate by fuzzy name match, got %d", result.Report.DuplicatesFound)
	}
}

func TestDeduplicate_ByOrganization(t *testing.T) {
	contacts := []model.Contact{
		{Name: "Alice", Organization: "Acme Corp"},
		{Name: "Bob", Organization: "Acme Corp"},
		{Name: "Charlie", Organization: "Other Inc"},
	}

	cfg := DefaultConfig()
	cfg.OrganizationWeight = 1.0
	cfg.EmailExactWeight = 0.0
	cfg.PhoneExactWeight = 0.0
	cfg.NameFuzzyWeight = 0.0
	cfg.Threshold = 0.5

	result := Deduplicate(contacts, cfg)

	if result.Report.DuplicatesFound != 1 {
		t.Errorf("expected 1 duplicate (by org), got %d", result.Report.DuplicatesFound)
	}
}

func TestDeduplicate_EmailAliasNormalization(t *testing.T) {
	contacts := []model.Contact{
		{Name: "Alice", Emails: []string{"alice+spam@example.com"}},
		{Name: "Alice Smith", Emails: []string{"alice@example.com"}},
	}

	cfg := DefaultConfig()
	cfg.EmailExactWeight = 1.0
	cfg.PhoneExactWeight = 0.0
	cfg.NameFuzzyWeight = 0.0
	cfg.OrganizationWeight = 0.0
	cfg.Threshold = 0.5

	result := Deduplicate(contacts, cfg)

	if result.Report.DuplicatesFound != 1 {
		t.Errorf("expected 1 duplicate (aliased email), got %d", result.Report.DuplicatesFound)
	}
}

func TestDeduplicate_PhoneFormatNormalization(t *testing.T) {
	contacts := []model.Contact{
		{Name: "Alice", Phones: []string{"+91 98765 43210"}},
		{Name: "Alice Smith", Phones: []string{"+919876543210"}},
	}

	cfg := DefaultConfig()
	cfg.PhoneExactWeight = 1.0
	cfg.EmailExactWeight = 0.0
	cfg.NameFuzzyWeight = 0.0
	cfg.OrganizationWeight = 0.0
	cfg.Threshold = 0.5

	result := Deduplicate(contacts, cfg)

	if result.Report.DuplicatesFound != 1 {
		t.Errorf("expected 1 duplicate (phone normalization), got %d", result.Report.DuplicatesFound)
	}
}

func TestDeduplicate_MultiSignalScoring(t *testing.T) {
	contacts := []model.Contact{
		{Name: "Alice", Emails: []string{"alice@example.com"}, Organization: "Acme"},
		{Name: "Alice Smith", Emails: []string{"alice@example.com"}, Organization: "Acme"},
	}

	cfg := DefaultConfig()
	cfg.Threshold = 0.3

	result := Deduplicate(contacts, cfg)

	if result.Report.DuplicatesFound != 1 {
		t.Errorf("expected 1 duplicate (email+org match), got %d", result.Report.DuplicatesFound)
	}
	if len(result.Merged) != 1 {
		t.Fatalf("expected 1 merged contact, got %d", len(result.Merged))
	}
	if result.Merged[0].Confidence <= 0.5 {
		t.Errorf("expected high confidence for multi-signal match, got %f", result.Merged[0].Confidence)
	}
}

func TestDeduplicate_StableClusterOrder(t *testing.T) {
	contacts := []model.Contact{
		{Name: "Zed", Emails: []string{"zed@example.com"}},
		{Name: "Alice", Emails: []string{"alice@example.com"}},
		{Name: "Alice Clone", Emails: []string{"Alice@Example.com"}},
	}

	cfg := DefaultConfig()
	cfg.EmailExactWeight = 1.0
	cfg.PhoneExactWeight = 0.0
	cfg.NameFuzzyWeight = 0.0
	cfg.OrganizationWeight = 0.0
	cfg.Threshold = 0.5

	result := Deduplicate(contacts, cfg)

	if len(result.Clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(result.Clusters))
	}
	if result.Clusters[0][0].Name != "Zed" {
		t.Fatalf("expected the first cluster to preserve input order, got %q", result.Clusters[0][0].Name)
	}
	if result.Clusters[1][0].Name != "Alice" {
		t.Fatalf("expected the duplicate cluster to follow the singleton cluster, got %q", result.Clusters[1][0].Name)
	}
}

func TestMergeIntoOne_DeduplicatesPhonesAndEmails(t *testing.T) {
	cluster := []model.Contact{
		{Name: "Alice", Phones: []string{"+1-212-555-0101"}, Emails: []string{"alice@example.com"}},
		{Name: "", Phones: []string{"+91-98765-43210"}, Emails: []string{"bob@example.com"}},
	}

	merged := mergeIntoOne(cluster)

	if merged.Name != "Alice" {
		t.Errorf("expected name 'Alice', got %q", merged.Name)
	}
	if len(merged.Phones) != 2 {
		t.Errorf("expected 2 phones merged, got %d", len(merged.Phones))
	}
	if len(merged.Emails) != 2 {
		t.Errorf("expected 2 emails merged, got %d", len(merged.Emails))
	}
}

func TestJaroWinkler_Identical(t *testing.T) {
	score := jaroWinklerSimilarity("alice", "alice")
	if score != 1.0 {
		t.Errorf("expected 1.0, got %f", score)
	}
}

func TestJaroWinkler_Similar(t *testing.T) {
	score := jaroWinklerSimilarity("alice", "alicia")
	if score < 0.5 || score > 0.99 {
		t.Errorf("expected score between 0.5 and 0.99 for similar names, got %f", score)
	}
}

func TestJaroWinkler_Different(t *testing.T) {
	score := jaroWinklerSimilarity("alice", "bob")
	if score > 0.5 {
		t.Errorf("expected low score for different names, got %f", score)
	}
}

func TestJaroWinkler_Empty(t *testing.T) {
	score := jaroWinklerSimilarity("", "alice")
	if score != 0.0 {
		t.Errorf("expected 0.0 for empty string, got %f", score)
	}
}

func TestNormalizeEmail_StripsPlus(t *testing.T) {
	result := normalize.NormalizeEmailStr("User+tag@Example.com")
	expected := "user@example.com"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestNormalizeEmail_Invalid(t *testing.T) {
	result := normalize.NormalizeEmailStr("notanemail")
	if result != "" {
		t.Errorf("expected empty for invalid email, got %q", result)
	}
}

func TestMatchEmails_Exact(t *testing.T) {
	score := matchEmails([]string{"alice@example.com"}, []string{"alice@example.com"})
	if score != 1.0 {
		t.Errorf("expected 1.0, got %f", score)
	}
}

func TestMatchEmails_NoMatch(t *testing.T) {
	score := matchEmails([]string{"alice@example.com"}, []string{"bob@example.com"})
	if score != 0.0 {
		t.Errorf("expected 0.0, got %f", score)
	}
}

func TestMatchPhones_Normalized(t *testing.T) {
	score := matchPhones([]string{"+1 (212) 555-0101"}, []string{"+1-212-555-0101"})
	if score != 1.0 {
		t.Errorf("expected 1.0 after normalization, got %f", score)
	}
}

func TestMatchPhones_NoMatch(t *testing.T) {
	score := matchPhones([]string{"+1-212-555-0101"}, []string{"+1-212-555-0102"})
	if score != 0.0 {
		t.Errorf("expected 0.0, got %f", score)
	}
}

func TestMatchNames_JaroWinkler(t *testing.T) {
	score := matchNames("Alice Smith", "Alice Smith")
	if score != 1.0 {
		t.Errorf("expected 1.0 for identical names, got %f", score)
	}
}

func TestMatchOrganizations_Exact(t *testing.T) {
	score := matchOrganizations("Acme Corp", "Acme Corp")
	if score != 1.0 {
		t.Errorf("expected 1.0, got %f", score)
	}
}

func TestMatchOrganizations_Substring(t *testing.T) {
	score := matchOrganizations("Acme Corporation", "Acme")
	if score <= 0.0 {
		t.Errorf("expected positive score for substring match, got %f", score)
	}
}

func TestMatchOrganizations_NoMatch(t *testing.T) {
	score := matchOrganizations("Acme Corp", "Other Inc")
	if score != 0.0 {
		t.Errorf("expected 0.0, got %f", score)
	}
}

func TestScorePair_MultiSignal(t *testing.T) {
	a := model.Contact{
		Name: "Alice Smith", Emails: []string{"alice@example.com"},
		Phones: []string{"+1-212-555-0101"}, Organization: "Acme",
	}
	b := model.Contact{
		Name: "Alice", Emails: []string{"alice@example.com"},
		Phones: []string{"2125550101"}, Organization: "Acme",
	}

	cfg := DefaultConfig()
	score := scorePair(a, b, cfg)

	if score <= 0.0 {
		t.Errorf("expected positive score for multi-signal match, got %f", score)
	}
}

func TestDeduplicate_DefaultConfigRealisticCases(t *testing.T) {
	tests := []struct {
		name  string
		a, b  model.Contact
		merge bool
	}{
		{
			"different people sharing an office phone",
			model.Contact{Name: "Front Desk", Phones: []string{"+1-212-555-0100"}, Emails: []string{"frontdesk@acme.com"}, Organization: "Acme"},
			model.Contact{Name: "John Doe", Phones: []string{"+1-212-555-0100"}, Emails: []string{"jdoe@acme.com"}, Organization: "Acme"},
			false,
		},
		{
			"shared team inbox",
			model.Contact{Name: "Info Desk", Emails: []string{"info@acme.com"}, Organization: "Acme"},
			model.Contact{Name: "Sales Team", Emails: []string{"info@acme.com"}, Organization: "Acme"},
			false,
		},
		{
			"name typo with same phone",
			model.Contact{Name: "Jon Doe", Phones: []string{"+12125550100"}},
			model.Contact{Name: "John Doe", Phones: []string{"+12125550100"}},
			true,
		},
		{
			"short name with same email",
			model.Contact{Name: "Bob", Emails: []string{"bob@x.com"}},
			model.Contact{Name: "Bob Johnson", Emails: []string{"bob@x.com"}},
			true,
		},
		{
			"initial with same email",
			model.Contact{Name: "J. Smith", Emails: []string{"jane@globex.com"}, Organization: "Globex"},
			model.Contact{Name: "Jane Smith", Emails: []string{"jane@globex.com"}, Organization: "Globex"},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Deduplicate([]model.Contact{tt.a, tt.b}, DefaultConfig())
			if merged := len(result.Clusters) == 1; merged != tt.merge {
				t.Errorf("merged = %v, want %v (score %.3f)", merged, tt.merge, scorePair(tt.a, tt.b, DefaultConfig()))
			}
		})
	}
}

func TestRepresentatives_KeepsSingletonsInOrder(t *testing.T) {
	contacts := []model.Contact{
		{Name: "Alice", Emails: []string{"alice@x.com"}},
		{Name: "Bob Johnson", Emails: []string{"bob@x.com"}},
		{Name: "Bob", Emails: []string{"bob@x.com"}, Phones: []string{"+12125550100"}},
		{Name: "Carol", Emails: []string{"carol@x.com"}},
	}
	got := Representatives(Deduplicate(contacts, DefaultConfig()))
	if len(got) != 3 {
		t.Fatalf("expected 3 contacts, got %d: %+v", len(got), got)
	}
	names := []string{got[0].Name, got[1].Name, got[2].Name}
	if names[0] != "Alice" || names[1] != "Bob Johnson" || names[2] != "Carol" {
		t.Errorf("unexpected order: %v", names)
	}
	if len(got[1].Phones) != 1 {
		t.Errorf("merged contact should carry the phone, got %v", got[1].Phones)
	}
}
