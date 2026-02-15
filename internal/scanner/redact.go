package scanner

import (
	"strings"

	"github.com/zricethezav/gitleaks/v8/detect"
)

// Redactor wraps gitleaks to detect and redact secrets in text.
type Redactor struct {
	detector *detect.Detector
}

// NewRedactor creates a Redactor with the default gitleaks rule set.
func NewRedactor() (*Redactor, error) {
	d, err := detect.NewDetectorDefaultConfig()
	if err != nil {
		return nil, err
	}
	return &Redactor{detector: d}, nil
}

// Redact scans text for secrets and replaces them with [REDACTED].
func (r *Redactor) Redact(text string) string {
	findings := r.detector.DetectString(text)
	if len(findings) == 0 {
		return text
	}

	result := text
	for _, f := range findings {
		if f.Secret != "" {
			result = strings.ReplaceAll(result, f.Secret, "[REDACTED]")
		}
	}
	return result
}
