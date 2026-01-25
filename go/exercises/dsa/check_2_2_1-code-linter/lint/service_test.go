package lint_test

import (
	"check_2_2_1-code-linter/lint"
	"embed"
	"testing"
)

//go:embed sample_test.go.test
var sampleFileFS embed.FS

func TestLint(t *testing.T) {
	f, err := sampleFileFS.Open("sample_test.go.test")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	err = lint.Lint(f)
	if err != nil {
		t.Errorf("did not expect error: %+v", err)
	}
}
