package tui

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/init0-lux/vcf-toolkit/internal/normalize"
)

type envConfig struct {
	LLMURL        string
	LLMAuthHeader string
	LLMAPIKey     string
	LLMBodyTmpl   string
}

// defaultCountry returns the calling code from VCF_TOOLKIT_COUNTRY (e.g. +91)
// used for phone numbers written without one.
func defaultCountry() (string, error) {
	v := strings.TrimSpace(os.Getenv("VCF_TOOLKIT_COUNTRY"))
	if v == "" {
		return "", nil
	}
	cc, ok := normalize.NormalizeCountryCode(v)
	if !ok {
		return "", fmt.Errorf("invalid VCF_TOOLKIT_COUNTRY %q: use a calling code such as +91 or +1", v)
	}
	return cc, nil
}

func loadDotEnv(path string) (envConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return envConfig{}, err
	}
	defer f.Close()

	var c envConfig
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		v = strings.Trim(v, "\"")
		switch k {
		case "VCF_TOOLKIT_LLM_URL":
			c.LLMURL = v
		case "VCF_TOOLKIT_LLM_AUTH_HEADER":
			c.LLMAuthHeader = v
		case "VCF_TOOLKIT_LLM_API_KEY":
			c.LLMAPIKey = v
		case "VCF_TOOLKIT_LLM_BODY_TEMPLATE":
			c.LLMBodyTmpl = v
		}
	}
	if err := sc.Err(); err != nil {
		return envConfig{}, err
	}
	return c, nil
}

func writeDotEnv(path string, c envConfig) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("env path required")
	}
	// 0600 because this may include API keys.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	lines := []string{
		"# vcf-toolkit configuration",
		"# Provider-agnostic HTTP LLM settings for name normalization.",
		"VCF_TOOLKIT_LLM_URL=" + shellEscapeEnv(c.LLMURL),
		"VCF_TOOLKIT_LLM_AUTH_HEADER=" + shellEscapeEnv(c.LLMAuthHeader),
		"VCF_TOOLKIT_LLM_API_KEY=" + shellEscapeEnv(c.LLMAPIKey),
		"VCF_TOOLKIT_LLM_BODY_TEMPLATE=" + shellEscapeEnv(c.LLMBodyTmpl),
		"",
	}
	_, err = f.WriteString(strings.Join(lines, "\n"))
	return err
}

func shellEscapeEnv(v string) string {
	// Minimal .env-friendly quoting.
	v = strings.ReplaceAll(v, "\n", " ")
	v = strings.ReplaceAll(v, "\r", " ")
	if v == "" {
		return ""
	}
	if strings.ContainsAny(v, " #\t") {
		return `"` + strings.ReplaceAll(v, `"`, `\"`) + `"`
	}
	return v
}
