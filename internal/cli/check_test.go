package cli

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRunCheck_Jobs_DeterministicOutputOrder(t *testing.T) {
	tmp := t.TempDir()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { _ = os.Chdir(orig) })

	// These transcripts are constructed so that the first one cannot complete
	// until the second one runs (it waits for a file created by the second).
	// This makes it very likely that completion order differs from input order,
	// so output ordering is meaningful even under parallel execution.
	require.NoError(t, os.WriteFile("a.cmdt", []byte(
		"$ sh -c 'while [ ! -f ready ]; do sleep 0.01; done; echo one'\n"+
			"1 two\n",
	), 0600))
	require.NoError(t, os.WriteFile("b.cmdt", []byte(
		"$ sh -c 'echo B; : > ready'\n"+
			"1 C\n",
	), 0600))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var out bytes.Buffer
	failures, err := runCheck(ctx, checkOptions{Filenames: []string{"a.cmdt", "b.cmdt"}, Out: &out, Jobs: 2})
	require.NoError(t, err)
	require.Equal(t, 2, failures)

	s := out.String()
	aIdx := strings.Index(s, "failed check at a.cmdt:")
	bIdx := strings.Index(s, "failed check at b.cmdt:")
	require.NotEqual(t, -1, aIdx)
	require.NotEqual(t, -1, bIdx)
	require.Less(t, aIdx, bIdx)
}

func TestRunCheck_Verbose_PrintsTimings(t *testing.T) {
	tmp := t.TempDir()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { _ = os.Chdir(orig) })

	require.NoError(t, os.WriteFile("ok.cmdt", []byte(
		"$ echo hi\n"+
			"1 hi\n",
	), 0600))

	var out bytes.Buffer
	failures, err := runCheck(context.Background(), checkOptions{
		Filenames: []string{"ok.cmdt"},
		Out:       &out,
		Jobs:      1,
		Verbose:   true,
	})
	require.NoError(t, err)
	require.Equal(t, 0, failures)

	s := out.String()
	require.Contains(t, s, "=== RUN   ok.cmdt\n")
	require.Contains(t, s, "--- PASS: ok.cmdt (")
}
