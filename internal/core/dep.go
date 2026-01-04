package core

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	filepathpkg "path/filepath"
	"strings"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// validateDepStmt ensures `% dep ...` stays a cache-dependency declaration,
// not a general-purpose shell escape hatch.
//
// We parse and expand the payload using mvdan/sh so quoting, parameter
// expansion, and stdin redirection work naturally. But we conservatively
// reject constructs that could execute other commands (command substitution,
// process substitution, background jobs, and non-stdin redirections), since
// those would create hidden subprocess dependencies that `go test` cannot
// reliably track.
func validateDepStmt(node syntax.Node) error {
	stmt, ok := node.(*syntax.Stmt)
	if !ok {
		return fmt.Errorf("internal error: expected *syntax.Stmt, got %T", node)
	}
	if stmt.Background {
		return fmt.Errorf("unsupported: background jobs")
	}

	call, ok := stmt.Cmd.(*syntax.CallExpr)
	if !ok {
		return fmt.Errorf("unsupported: expected a simple command")
	}
	if len(call.Assigns) > 0 {
		return fmt.Errorf("unsupported: assignments")
	}
	if len(call.Args) == 0 {
		return fmt.Errorf("internal error: missing command name")
	}
	if !isLiteralWord(call.Args[0], "dep") {
		return fmt.Errorf("internal error: expected command name \"dep\"")
	}
	for _, redir := range stmt.Redirs {
		if redir.Op != syntax.RdrIn {
			return fmt.Errorf("unsupported: only stdin redirections (<) are allowed")
		}
		if redir.N != nil && redir.N.Value != "0" {
			return fmt.Errorf("unsupported: only fd 0 redirections (<) are allowed")
		}
	}

	var walkErr error
	syntax.Walk(stmt, func(n syntax.Node) bool {
		if n == nil || walkErr != nil {
			return false
		}
		// Deliberately no default case: syntax.Walk will visit many "normal" node
		// types (words, quotes, parameter expansions, etc). Here we only reject
		// known-dangerous constructs that can execute other commands, since those
		// would create hidden subprocess dependencies that `go test` won't track.
		switch n.(type) {
		case *syntax.CmdSubst:
			walkErr = fmt.Errorf("unsupported: command substitution")
		case *syntax.ProcSubst:
			walkErr = fmt.Errorf("unsupported: process substitution")
		}
		return walkErr == nil
	})
	return walkErr
}

func isLiteralWord(w *syntax.Word, want string) bool {
	if w == nil || len(w.Parts) != 1 {
		return false
	}
	lit, ok := w.Parts[0].(*syntax.Lit)
	return ok && lit.Value == want
}

func runDepIntrinsic(ctx context.Context, args []string) error {
	hc := interp.HandlerCtx(ctx)
	for _, arg := range args {
		if err := recordDepArg(hc.Dir, arg); err != nil {
			return err
		}
	}

	if hc.Stdin != nil {
		if err := recordDepfile(hc.Dir, hc.Stdin); err != nil {
			return err
		}
	}
	return nil
}

func recordDepArg(dir, raw string) error {
	if raw == "" {
		return nil
	}

	switch raw[0] {
	case '$':
		name := strings.TrimSpace(depUnescape(raw[1:]))
		if name == "" {
			return fmt.Errorf("invalid env var dependency: %q", raw)
		}
		_, _ = os.LookupEnv(name)
		return nil

	default:
		depStat(dir, depUnescape(raw))
		return nil
	}
}

func recordDepfile(dir string, r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" {
			continue
		}
		switch line[0] {
		case '#':
			continue
		case '$':
			name := strings.TrimSpace(depUnescape(line[1:]))
			if name == "" {
				return fmt.Errorf("invalid depfile env var line: %q", line)
			}
			_, _ = os.LookupEnv(name)
		default:
			depStat(dir, depUnescape(line))
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading depfile: %w", err)
	}
	return nil
}

func depStat(dir, path string) {
	if path == "" {
		return
	}
	if dir != "" && !filepathpkg.IsAbs(path) {
		path = filepathpkg.Join(dir, path)
	}

	// Best-effort: even errors still get recorded in the go test "testlog"
	// as a stat action, which is useful for caching.
	//
	// Prefer Stat over Open: `go test` intentionally refuses to cache runs that
	// "open" regular files with very recent mtimes ("too new"), to avoid flaky
	// cache hits on file systems with coarse mtime precision. Stat does not
	// trigger that cutoff.
	_, _ = os.Stat(path)
}

func depUnescape(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch != '\\' {
			b.WriteByte(ch)
			continue
		}
		if i == len(s)-1 {
			b.WriteByte('\\')
			continue
		}
		i++
		switch s[i] {
		case '\\':
			b.WriteByte('\\')
		case '$':
			b.WriteByte('$')
		case 'n':
			b.WriteByte('\n')
		default:
			b.WriteByte('\\')
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
