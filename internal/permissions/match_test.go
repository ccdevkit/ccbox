package permissions

import (
	"testing"
)

func mustParsePattern(t *testing.T, raw string) *ArgPattern {
	t.Helper()
	pat, err := ParsePattern(raw)
	if err != nil {
		t.Fatalf("ParsePattern(%q) error: %v", raw, err)
	}
	return pat
}

func TestMatchPattern_LiteralSingleArg(t *testing.T) {
	pat := mustParsePattern(t, "pull")
	if !MatchPattern(pat, []string{"pull"}) {
		t.Error("expected pattern 'pull' to match args [pull]")
	}
}

func TestMatchPattern_LiteralMismatch(t *testing.T) {
	pat := mustParsePattern(t, "pull")
	if MatchPattern(pat, []string{"push"}) {
		t.Error("expected pattern 'pull' not to match args [push]")
	}
}

func TestMatchPattern_MultiLiteralMatch(t *testing.T) {
	pat := mustParsePattern(t, "push origin")
	if !MatchPattern(pat, []string{"push", "origin"}) {
		t.Error("expected pattern 'push origin' to match args [push, origin]")
	}
}

func TestMatchPattern_MultiLiteralMismatch(t *testing.T) {
	pat := mustParsePattern(t, "push origin")
	if MatchPattern(pat, []string{"push", "other"}) {
		t.Error("expected pattern 'push origin' not to match args [push, other]")
	}
}

func TestMatchPattern_PrefixMatching(t *testing.T) {
	pat := mustParsePattern(t, "pull")
	if !MatchPattern(pat, []string{"pull", "origin"}) {
		t.Error("expected pattern 'pull' to match args [pull, origin] via prefix matching")
	}
}

func TestMatchPattern_PrefixMatchingExtraArgs(t *testing.T) {
	pat := mustParsePattern(t, "status")
	if !MatchPattern(pat, []string{"status", "--short"}) {
		t.Error("expected pattern 'status' to match args [status, --short] via prefix matching")
	}
}

func TestMatchPattern_ExactMatchBlocksExtraArgs(t *testing.T) {
	pat := mustParsePattern(t, "status $")
	if MatchPattern(pat, []string{"status", "--short"}) {
		t.Error("expected pattern 'status $' NOT to match args [status, --short] with exact match")
	}
}

func TestMatchPattern_ExactMatchNoExtraArgs(t *testing.T) {
	pat := mustParsePattern(t, "status $")
	if !MatchPattern(pat, []string{"status"}) {
		t.Error("expected pattern 'status $' to match args [status] with exact match")
	}
}

func TestMatchPattern_WildcardMatchesSingleArg(t *testing.T) {
	pat := mustParsePattern(t, "*")
	if !MatchPattern(pat, []string{"anything"}) {
		t.Error("expected pattern '*' to match args [anything]")
	}
}

func TestMatchPattern_WildcardNoArgs(t *testing.T) {
	pat := mustParsePattern(t, "*")
	if MatchPattern(pat, []string{}) {
		t.Error("expected pattern '*' not to match empty args")
	}
}

func TestMatchPattern_LiteralWithGlobPrefix(t *testing.T) {
	pat := mustParsePattern(t, "--*")
	if !MatchPattern(pat, []string{"--verbose"}) {
		t.Error("expected pattern '--*' to match args [--verbose]")
	}
}

func TestMatchPattern_LiteralWithGlobPrefixMismatch(t *testing.T) {
	pat := mustParsePattern(t, "--*")
	if MatchPattern(pat, []string{"verbose"}) {
		t.Error("expected pattern '--*' not to match args [verbose]")
	}
}

func TestMatchPattern_LiteralWithGlobInfix(t *testing.T) {
	pat := mustParsePattern(t, "pre*fix")
	if !MatchPattern(pat, []string{"prefix"}) {
		t.Error("expected pattern 'pre*fix' to match args [prefix]")
	}
}

func TestMatchPattern_LiteralWithGlobInfixLonger(t *testing.T) {
	pat := mustParsePattern(t, "pre*fix")
	if !MatchPattern(pat, []string{"pre-blah-fix"}) {
		t.Error("expected pattern 'pre*fix' to match args [pre-blah-fix]")
	}
}

func TestMatchPattern_LiteralWithGlobSuffix(t *testing.T) {
	pat := mustParsePattern(t, "*suffix")
	if !MatchPattern(pat, []string{"my-suffix"}) {
		t.Error("expected pattern '*suffix' to match args [my-suffix]")
	}
}

func TestMatchPattern_TwoWildcardsMatchTwoArgs(t *testing.T) {
	pat := mustParsePattern(t, "* *")
	if !MatchPattern(pat, []string{"a", "b"}) {
		t.Error("expected pattern '* *' to match args [a, b]")
	}
}

func TestMatchPattern_DoubleWildcardMatchesMultipleArgs(t *testing.T) {
	pat := mustParsePattern(t, "push **")
	if !MatchPattern(pat, []string{"push", "origin", "main"}) {
		t.Error("expected pattern 'push **' to match args [push, origin, main]")
	}
}

func TestMatchPattern_DoubleWildcardMatchesAnything(t *testing.T) {
	pat := mustParsePattern(t, "**")
	if !MatchPattern(pat, []string{"anything", "at", "all"}) {
		t.Error("expected pattern '**' to match args [anything, at, all]")
	}
}

func TestMatchPattern_DoubleWildcardMatchesZeroTrailingArgs(t *testing.T) {
	pat := mustParsePattern(t, "push **")
	if !MatchPattern(pat, []string{"push"}) {
		t.Error("expected pattern 'push **' to match args [push] (** matches zero trailing args)")
	}
}

func TestMatchPattern_DoubleWildcardMatchesEmptyArgs(t *testing.T) {
	pat := mustParsePattern(t, "**")
	if !MatchPattern(pat, []string{}) {
		t.Error("expected pattern '**' to match empty args")
	}
}

func TestMatchPattern_DoubleWildcardExactMatchMultipleArgs(t *testing.T) {
	pat := mustParsePattern(t, "push **$")
	if !MatchPattern(pat, []string{"push", "origin"}) {
		t.Error("expected pattern 'push **$' to match args [push, origin]")
	}
}

func TestMatchPattern_DoubleWildcardExactMatchZeroTrailing(t *testing.T) {
	pat := mustParsePattern(t, "push **$")
	if !MatchPattern(pat, []string{"push"}) {
		t.Error("expected pattern 'push **$' to match args [push] (** matches zero)")
	}
}

func TestMatchPattern_SingleCharDotMatchesSingleCharArg(t *testing.T) {
	pat := mustParsePattern(t, ".")
	if !MatchPattern(pat, []string{"x"}) {
		t.Error("expected pattern '.' to match args [x]")
	}
}

func TestMatchPattern_SingleCharDotRejectsMultiCharArg(t *testing.T) {
	pat := mustParsePattern(t, ".")
	if MatchPattern(pat, []string{"xx"}) {
		t.Error("expected pattern '.' not to match args [xx]")
	}
}

func TestMatchPattern_LiteralWithDotSuffix(t *testing.T) {
	pat := mustParsePattern(t, "v.")
	if !MatchPattern(pat, []string{"v1"}) {
		t.Error("expected pattern 'v.' to match args [v1]")
	}
}

func TestMatchPattern_LiteralWithDotInfix(t *testing.T) {
	pat := mustParsePattern(t, "a.b")
	if !MatchPattern(pat, []string{"axb"}) {
		t.Error("expected pattern 'a.b' to match args [axb]")
	}
}

func TestMatchPattern_LiteralWithDotInfixTooShort(t *testing.T) {
	pat := mustParsePattern(t, "a.b")
	if MatchPattern(pat, []string{"ab"}) {
		t.Error("expected pattern 'a.b' not to match args [ab]")
	}
}

func TestMatchPattern_LiteralWithDotPrefix(t *testing.T) {
	pat := mustParsePattern(t, ".v")
	if !MatchPattern(pat, []string{"xv"}) {
		t.Error("expected pattern '.v' to match args [xv]")
	}
}

func TestMatchPattern_RegexMatchesArg(t *testing.T) {
	pat := mustParsePattern(t, `/^https?:\/\//`)
	if !MatchPattern(pat, []string{"https://github.com"}) {
		t.Error(`expected pattern '/^https?:\/\//' to match args ["https://github.com"]`)
	}
}

func TestMatchPattern_RegexRejectsNonMatch(t *testing.T) {
	pat := mustParsePattern(t, `/^https?:\/\//`)
	if MatchPattern(pat, []string{"ftp://server"}) {
		t.Error(`expected pattern '/^https?:\/\//' not to match args ["ftp://server"]`)
	}
}

func TestMatchPattern_RegexNoArgs(t *testing.T) {
	pat := mustParsePattern(t, `/^foo/`)
	if MatchPattern(pat, []string{}) {
		t.Error(`expected pattern '/^foo/' not to match empty args`)
	}
}

func TestMatchPattern_RegexPartialMatch(t *testing.T) {
	pat := mustParsePattern(t, `/bar/`)
	if !MatchPattern(pat, []string{"foobar"}) {
		t.Error(`expected pattern '/bar/' to match args ["foobar"] (partial match)`)
	}
}

func TestMatchPattern_LiteralPlusRegexMatch(t *testing.T) {
	pat := mustParsePattern(t, `clone /^https/`)
	if !MatchPattern(pat, []string{"clone", "https://foo"}) {
		t.Error(`expected pattern 'clone /^https/' to match args ["clone", "https://foo"]`)
	}
}

func TestMatchPattern_LiteralPlusRegexMismatch(t *testing.T) {
	pat := mustParsePattern(t, `clone /^https/`)
	if MatchPattern(pat, []string{"clone", "ftp://foo"}) {
		t.Error(`expected pattern 'clone /^https/' not to match args ["clone", "ftp://foo"]`)
	}
}

func TestMatchPattern_RegexMultiMatchesSingleArg(t *testing.T) {
	pat := mustParsePattern(t, `/--force|--hard/**`)
	if !MatchPattern(pat, []string{"--force"}) {
		t.Error(`expected pattern '/--force|--hard/**' to match args ["--force"]`)
	}
}

func TestMatchPattern_RegexMultiMatchesArgAnywhere(t *testing.T) {
	pat := mustParsePattern(t, `/--force|--hard/**`)
	if !MatchPattern(pat, []string{"origin", "--force", "main"}) {
		t.Error(`expected pattern '/--force|--hard/**' to match args ["origin", "--force", "main"]`)
	}
}

func TestMatchPattern_RegexMultiNoMatch(t *testing.T) {
	pat := mustParsePattern(t, `/--force|--hard/**`)
	if MatchPattern(pat, []string{"origin", "main"}) {
		t.Error(`expected pattern '/--force|--hard/**' not to match args ["origin", "main"]`)
	}
}

func TestMatchPattern_RegexMultiNoArgs(t *testing.T) {
	pat := mustParsePattern(t, `/--force/**`)
	if MatchPattern(pat, []string{}) {
		t.Error(`expected pattern '/--force/**' not to match empty args`)
	}
}

func TestMatchPattern_NonPositionalFoundMiddle(t *testing.T) {
	pat := mustParsePattern(t, "push ~--force")
	if !MatchPattern(pat, []string{"push", "--force", "origin"}) {
		t.Error("expected pattern 'push ~--force' to match [push, --force, origin]")
	}
}

func TestMatchPattern_NonPositionalFoundLater(t *testing.T) {
	pat := mustParsePattern(t, "push ~--force")
	if !MatchPattern(pat, []string{"push", "origin", "--force", "main"}) {
		t.Error("expected pattern 'push ~--force' to match [push, origin, --force, main]")
	}
}

func TestMatchPattern_NonPositionalNotFound(t *testing.T) {
	pat := mustParsePattern(t, "push ~--force")
	if MatchPattern(pat, []string{"push", "origin", "main"}) {
		t.Error("expected pattern 'push ~--force' not to match [push, origin, main]")
	}
}

func TestMatchPattern_OptionalNotPresent(t *testing.T) {
	pat := mustParsePattern(t, "pull origin?")
	if !MatchPattern(pat, []string{"pull"}) {
		t.Error("expected pattern 'pull origin?' to match [pull] (origin is optional, not present)")
	}
}

func TestMatchPattern_OptionalPresent(t *testing.T) {
	pat := mustParsePattern(t, "pull origin?")
	if !MatchPattern(pat, []string{"pull", "origin"}) {
		t.Error("expected pattern 'pull origin?' to match [pull, origin] (origin is optional, present)")
	}
}

func TestMatchPattern_OptionalExactMatchUnmatchedArg(t *testing.T) {
	pat := mustParsePattern(t, "pull origin?$")
	if MatchPattern(pat, []string{"pull", "upstream"}) {
		t.Error("expected pattern 'pull origin?$' not to match [pull, upstream] (exact match, unmatched arg)")
	}
}

func TestMatchPattern_OptionalPrefixMatchExtraArg(t *testing.T) {
	pat := mustParsePattern(t, "pull origin?")
	if !MatchPattern(pat, []string{"pull", "upstream"}) {
		t.Error("expected pattern 'pull origin?' to match [pull, upstream] (prefix matching, extra arg OK)")
	}
}

func TestMatchPattern_GroupPositionalMatch(t *testing.T) {
	pat := mustParsePattern(t, "push (origin main)")
	if !MatchPattern(pat, []string{"push", "origin", "main"}) {
		t.Error("expected pattern 'push (origin main)' to match [push, origin, main]")
	}
}

func TestMatchPattern_GroupPositionalMismatch(t *testing.T) {
	pat := mustParsePattern(t, "push (origin main)")
	if MatchPattern(pat, []string{"push", "origin", "dev"}) {
		t.Error("expected pattern 'push (origin main)' not to match [push, origin, dev]")
	}
}

func TestMatchPattern_GroupPositionalTooFewArgs(t *testing.T) {
	pat := mustParsePattern(t, "push (origin main)")
	if MatchPattern(pat, []string{"push", "origin"}) {
		t.Error("expected pattern 'push (origin main)' not to match [push, origin] (group needs 2 args)")
	}
}

func TestMatchPattern_GroupOptionalPresent(t *testing.T) {
	pat := mustParsePattern(t, "(origin main)?")
	if !MatchPattern(pat, []string{"origin", "main"}) {
		t.Error("expected pattern '(origin main)?' to match [origin, main]")
	}
}

func TestMatchPattern_GroupOptionalAbsentPrefix(t *testing.T) {
	pat := mustParsePattern(t, "(origin main)?")
	if !MatchPattern(pat, []string{}) {
		t.Error("expected pattern '(origin main)?' to match [] (optional group absent, prefix matching)")
	}
}

func TestMatchPattern_GroupOptionalAbsentExact(t *testing.T) {
	pat := mustParsePattern(t, "(origin main)?$")
	if !MatchPattern(pat, []string{}) {
		t.Error("expected pattern '(origin main)?$' to match [] (optional group absent, exact match)")
	}
}

func TestMatchPattern_NonPositionalGroupFound(t *testing.T) {
	pat := mustParsePattern(t, "~(-n 0)")
	if !MatchPattern(pat, []string{"cmd", "foo", "-n", "0", "bar"}) {
		t.Error("expected pattern '~(-n 0)' to match [cmd, foo, -n, 0, bar]")
	}
}

func TestMatchPattern_NonPositionalGroupNotFound(t *testing.T) {
	pat := mustParsePattern(t, "~(-n 0)")
	if MatchPattern(pat, []string{"cmd", "foo"}) {
		t.Error("expected pattern '~(-n 0)' not to match [cmd, foo]")
	}
}

func TestMatchPattern_QuotedDoubleQuoteMatch(t *testing.T) {
	pat := mustParsePattern(t, `"my file"`)
	if !MatchPattern(pat, []string{"my file"}) {
		t.Error(`expected pattern '"my file"' to match args ["my file"]`)
	}
}

func TestMatchPattern_QuotedSingleQuoteMatch(t *testing.T) {
	pat := mustParsePattern(t, `'my file'`)
	if !MatchPattern(pat, []string{"my file"}) {
		t.Error(`expected pattern "'my file'" to match args ["my file"]`)
	}
}

func TestMatchPattern_QuotedMismatch(t *testing.T) {
	pat := mustParsePattern(t, `"my file"`)
	if MatchPattern(pat, []string{"my"}) {
		t.Error(`expected pattern '"my file"' not to match args ["my"]`)
	}
}

func TestMatchPattern_QuotedDotIsLiteral(t *testing.T) {
	pat := mustParsePattern(t, `"v."`)
	if !MatchPattern(pat, []string{"v."}) {
		t.Error(`expected pattern '"v."' to match args ["v."] (literal dot)`)
	}
}

func TestMatchPattern_QuotedDotDoesNotMatchWildcard(t *testing.T) {
	pat := mustParsePattern(t, `"v."`)
	if MatchPattern(pat, []string{"v1"}) {
		t.Error(`expected pattern '"v."' not to match args ["v1"] (dot is literal in quotes)`)
	}
}

func TestMatchPattern_EscapedStarMatchesLiteralAsterisk(t *testing.T) {
	pat := mustParsePattern(t, `\*`)
	if !MatchPattern(pat, []string{"*"}) {
		t.Error(`expected pattern '\*' to match args ["*"] (literal asterisk)`)
	}
}

func TestMatchPattern_EscapedStarDoesNotMatchArbitraryArg(t *testing.T) {
	pat := mustParsePattern(t, `\*`)
	if MatchPattern(pat, []string{"foo"}) {
		t.Error(`expected pattern '\*' not to match args ["foo"] (escaped star is not a wildcard)`)
	}
}
