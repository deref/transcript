package core

import (
	"context"
	"strings"
	"testing"
)

func TestDepUnescape(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{in: ``, want: ``},
		{in: `abc`, want: `abc`},
		{in: `\\`, want: `\`},
		{in: `\$`, want: `$`},
		{in: `\n`, want: "\n"},
		{in: `\x`, want: `\x`},
		{in: `\`, want: `\`},
	}
	for _, tc := range cases {
		if got := depUnescape(tc.in); got != tc.want {
			t.Fatalf("depUnescape(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRecordDepfile(t *testing.T) {
	t.Setenv("FOO", "set")
	depfile := strings.Join([]string{
		"# comment",
		"",
		"$FOO",
		`a-path`,
		`\$literal-dollar-path`,
		`\\literal-backslash`,
		"",
	}, "\n")

	if err := recordDepfile(t.TempDir(), strings.NewReader(depfile)); err != nil {
		t.Fatalf("recordDepfile: %v", err)
	}
}

func TestRecordDepfile_InvalidEnvLine(t *testing.T) {
	t.Parallel()

	if err := recordDepfile(t.TempDir(), strings.NewReader("$\n")); err == nil {
		t.Fatalf("expected error")
	}
	if err := recordDepfile(t.TempDir(), strings.NewReader("$   \n")); err == nil {
		t.Fatalf("expected error")
	}
}

func TestValidateDepStmt(t *testing.T) {
	t.Parallel()

	okStmt, err := parseStmt(`dep foo "$BAR" < deps.txt`)
	if err != nil {
		t.Fatalf("parseStmt: %v", err)
	}
	if err := validateDepStmt(okStmt); err != nil {
		t.Fatalf("validateDepStmt: %v", err)
	}

	reject := []string{
		`FOO=bar dep foo`,
		`dep foo > out.txt`,
		`dep foo 2> out.txt`,
		"dep foo <<EOF\nbar\nEOF",
		`dep foo <<< bar`,
		`dep foo && other`,
		`dep foo | other`,
		`dep $(echo hi)`,
		"dep `echo hi`",
		`dep <(echo hi)`,
		`dep foo &`,
	}
	for _, s := range reject {
		stmt, err := parseStmt(s)
		if err != nil {
			t.Fatalf("parseStmt(%q): %v", s, err)
		}
		if err := validateDepStmt(stmt); err == nil {
			t.Fatalf("validateDepStmt(%q): expected error", s)
		}
	}
}

func TestRunDepDirective_RejectsMultipleStatements(t *testing.T) {
	var rec Recorder
	if err := rec.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := rec.RunDepDirective(context.Background(), "foo; bar"); err == nil {
		t.Fatalf("expected error")
	}
}
