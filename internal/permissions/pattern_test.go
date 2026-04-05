package permissions

import (
	"testing"
)

func TestParsePattern_ExactMatchTrailingDollar(t *testing.T) {
	p, err := ParsePattern("push --force$")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.ExactMatch {
		t.Error("expected ExactMatch=true")
	}
	if len(p.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(p.Elements))
	}
	if p.Elements[0].Value != "push" {
		t.Errorf("expected first element 'push', got %q", p.Elements[0].Value)
	}
	if p.Elements[1].Value != "--force" {
		t.Errorf("expected second element '--force', got %q", p.Elements[1].Value)
	}
}

func TestParsePattern_ExactMatchDollarWithSpace(t *testing.T) {
	p, err := ParsePattern("push --force $")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.ExactMatch {
		t.Error("expected ExactMatch=true")
	}
	if len(p.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(p.Elements))
	}
	if p.Elements[0].Value != "push" {
		t.Errorf("expected first element 'push', got %q", p.Elements[0].Value)
	}
	if p.Elements[1].Value != "--force" {
		t.Errorf("expected second element '--force', got %q", p.Elements[1].Value)
	}
}

func TestParsePattern_ExactMatchDollarWithExtraWhitespace(t *testing.T) {
	p, err := ParsePattern("push --force           $")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.ExactMatch {
		t.Error("expected ExactMatch=true")
	}
	if len(p.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(p.Elements))
	}
	if p.Elements[0].Value != "push" {
		t.Errorf("expected first element 'push', got %q", p.Elements[0].Value)
	}
	if p.Elements[1].Value != "--force" {
		t.Errorf("expected second element '--force', got %q", p.Elements[1].Value)
	}
}

func TestParsePattern_WhitespaceCollapsing(t *testing.T) {
	p, err := ParsePattern("push        --force")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ExactMatch {
		t.Error("expected ExactMatch=false")
	}
	if len(p.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(p.Elements))
	}
	if p.Elements[0].Value != "push" {
		t.Errorf("expected first element 'push', got %q", p.Elements[0].Value)
	}
	if p.Elements[1].Value != "--force" {
		t.Errorf("expected second element '--force', got %q", p.Elements[1].Value)
	}
}

func TestParsePattern_DollarMidPattern_Error(t *testing.T) {
	_, err := ParsePattern("push $ --force")
	if err == nil {
		t.Fatal("expected error for $ mid-pattern, got nil")
	}
}

func TestParsePattern_LeadingTrailingWhitespace(t *testing.T) {
	p, err := ParsePattern("  push  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ExactMatch {
		t.Error("expected ExactMatch=false")
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Value != "push" {
		t.Errorf("expected element 'push', got %q", p.Elements[0].Value)
	}
}

func TestParsePattern_SingleWildcard(t *testing.T) {
	p, err := ParsePattern("*")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementWildcard {
		t.Errorf("expected type ElementWildcard, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "*" {
		t.Errorf("expected value '*', got %q", p.Elements[0].Value)
	}
}

func TestParsePattern_DoubleWildcard(t *testing.T) {
	p, err := ParsePattern("**")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementDoubleWildcard {
		t.Errorf("expected type ElementDoubleWildcard, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "**" {
		t.Errorf("expected value '**', got %q", p.Elements[0].Value)
	}
}

func TestParsePattern_WildcardEmbeddedInToken(t *testing.T) {
	p, err := ParsePattern("--*")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementLiteral {
		t.Errorf("expected type ElementLiteral, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "--*" {
		t.Errorf("expected value '--*', got %q", p.Elements[0].Value)
	}
}

func TestParsePattern_WildcardInMiddleOfToken(t *testing.T) {
	p, err := ParsePattern("pre*fix")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementLiteral {
		t.Errorf("expected type ElementLiteral, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "pre*fix" {
		t.Errorf("expected value 'pre*fix', got %q", p.Elements[0].Value)
	}
}

func TestParsePattern_WildcardSuffix(t *testing.T) {
	p, err := ParsePattern("*suffix")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementLiteral {
		t.Errorf("expected type ElementLiteral, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "*suffix" {
		t.Errorf("expected value '*suffix', got %q", p.Elements[0].Value)
	}
}

func TestParsePattern_LiteralPlusDoubleWildcard(t *testing.T) {
	p, err := ParsePattern("push **")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementLiteral {
		t.Errorf("element 0: expected type ElementLiteral, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "push" {
		t.Errorf("element 0: expected value 'push', got %q", p.Elements[0].Value)
	}
	if p.Elements[1].Type != ElementDoubleWildcard {
		t.Errorf("element 1: expected type ElementDoubleWildcard, got %q", p.Elements[1].Type)
	}
	if p.Elements[1].Value != "**" {
		t.Errorf("element 1: expected value '**', got %q", p.Elements[1].Value)
	}
}

func TestParsePattern_SingleLiteralToken(t *testing.T) {
	p, err := ParsePattern("pull")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementLiteral {
		t.Errorf("expected type ElementLiteral, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "pull" {
		t.Errorf("expected value 'pull', got %q", p.Elements[0].Value)
	}
}

func TestParsePattern_MultipleLiteralTokens(t *testing.T) {
	p, err := ParsePattern("push origin main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(p.Elements))
	}
	expected := []string{"push", "origin", "main"}
	for i, want := range expected {
		if p.Elements[i].Type != ElementLiteral {
			t.Errorf("element %d: expected type ElementLiteral, got %q", i, p.Elements[i].Type)
		}
		if p.Elements[i].Value != want {
			t.Errorf("element %d: expected value %q, got %q", i, want, p.Elements[i].Value)
		}
	}
}

func TestParsePattern_DotSingleChar(t *testing.T) {
	p, err := ParsePattern(".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementSingleChar {
		t.Errorf("expected type ElementSingleChar, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "." {
		t.Errorf("expected value '.', got %q", p.Elements[0].Value)
	}
}

func TestParsePattern_DotEmbeddedSuffix(t *testing.T) {
	p, err := ParsePattern("v.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementLiteral {
		t.Errorf("expected type ElementLiteral, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "v." {
		t.Errorf("expected value 'v.', got %q", p.Elements[0].Value)
	}
}

func TestParsePattern_DotEmbeddedMiddle(t *testing.T) {
	p, err := ParsePattern("a.b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementLiteral {
		t.Errorf("expected type ElementLiteral, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "a.b" {
		t.Errorf("expected value 'a.b', got %q", p.Elements[0].Value)
	}
}

func TestParsePattern_DotEmbeddedPrefix(t *testing.T) {
	p, err := ParsePattern(".v")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementLiteral {
		t.Errorf("expected type ElementLiteral, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != ".v" {
		t.Errorf("expected value '.v', got %q", p.Elements[0].Value)
	}
}

func TestParsePattern_DoubleDot(t *testing.T) {
	p, err := ParsePattern("..")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(p.Elements))
	}
	for i, elem := range p.Elements {
		if elem.Type != ElementSingleChar {
			t.Errorf("element %d: expected type ElementSingleChar, got %q", i, elem.Type)
		}
		if elem.Value != "." {
			t.Errorf("element %d: expected value '.', got %q", i, elem.Value)
		}
	}
}

func TestParsePattern_RegexToken(t *testing.T) {
	p, err := ParsePattern("/^https/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementRegex {
		t.Errorf("expected type ElementRegex, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "^https" {
		t.Errorf("expected value %q, got %q", "^https", p.Elements[0].Value)
	}
}

func TestParsePattern_RegexMultiToken(t *testing.T) {
	p, err := ParsePattern("/^https/**")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementRegexMulti {
		t.Errorf("expected type ElementRegexMulti, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "^https" {
		t.Errorf("expected value %q, got %q", "^https", p.Elements[0].Value)
	}
}

func TestParsePattern_RegexInvalidError(t *testing.T) {
	_, err := ParsePattern("/invalid[/")
	if err == nil {
		t.Fatal("expected error for invalid regex, got nil")
	}
}

func TestParsePattern_RegexEscapedSlashes(t *testing.T) {
	p, err := ParsePattern(`/pattern\/with\/slashes/`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementRegex {
		t.Errorf("expected type ElementRegex, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "pattern/with/slashes" {
		t.Errorf("expected value %q, got %q", "pattern/with/slashes", p.Elements[0].Value)
	}
}

func TestParsePattern_NonPositionalLiteral(t *testing.T) {
	p, err := ParsePattern("~--force")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementLiteral {
		t.Errorf("expected type ElementLiteral, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "--force" {
		t.Errorf("expected value '--force', got %q", p.Elements[0].Value)
	}
	if !p.Elements[0].NonPositional {
		t.Error("expected NonPositional=true")
	}
}

func TestParsePattern_NonPositionalMixed(t *testing.T) {
	p, err := ParsePattern("push ~--force")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementLiteral {
		t.Errorf("element 0: expected type ElementLiteral, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "push" {
		t.Errorf("element 0: expected value 'push', got %q", p.Elements[0].Value)
	}
	if p.Elements[0].NonPositional {
		t.Error("element 0: expected NonPositional=false")
	}
	if p.Elements[1].Type != ElementLiteral {
		t.Errorf("element 1: expected type ElementLiteral, got %q", p.Elements[1].Type)
	}
	if p.Elements[1].Value != "--force" {
		t.Errorf("element 1: expected value '--force', got %q", p.Elements[1].Value)
	}
	if !p.Elements[1].NonPositional {
		t.Error("element 1: expected NonPositional=true")
	}
}

func TestParsePattern_NonPositionalWildcard(t *testing.T) {
	p, err := ParsePattern("~*")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementWildcard {
		t.Errorf("expected type ElementWildcard, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "*" {
		t.Errorf("expected value '*', got %q", p.Elements[0].Value)
	}
	if !p.Elements[0].NonPositional {
		t.Error("expected NonPositional=true")
	}
}

func TestParsePattern_OptionalLiteral(t *testing.T) {
	p, err := ParsePattern("origin?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementLiteral {
		t.Errorf("expected type ElementLiteral, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "origin" {
		t.Errorf("expected value 'origin', got %q", p.Elements[0].Value)
	}
	if !p.Elements[0].Optional {
		t.Error("expected Optional=true")
	}
}

func TestParsePattern_OptionalSecondToken(t *testing.T) {
	p, err := ParsePattern("pull origin?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementLiteral {
		t.Errorf("element 0: expected type ElementLiteral, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "pull" {
		t.Errorf("element 0: expected value 'pull', got %q", p.Elements[0].Value)
	}
	if p.Elements[0].Optional {
		t.Error("element 0: expected Optional=false")
	}
	if p.Elements[1].Type != ElementLiteral {
		t.Errorf("element 1: expected type ElementLiteral, got %q", p.Elements[1].Type)
	}
	if p.Elements[1].Value != "origin" {
		t.Errorf("element 1: expected value 'origin', got %q", p.Elements[1].Value)
	}
	if !p.Elements[1].Optional {
		t.Error("element 1: expected Optional=true")
	}
}

func TestParsePattern_NonPositionalOptional(t *testing.T) {
	p, err := ParsePattern("~--force?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementLiteral {
		t.Errorf("expected type ElementLiteral, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "--force" {
		t.Errorf("expected value '--force', got %q", p.Elements[0].Value)
	}
	if !p.Elements[0].NonPositional {
		t.Error("expected NonPositional=true")
	}
	if !p.Elements[0].Optional {
		t.Error("expected Optional=true")
	}
}

func TestParsePattern_BareOptionalError(t *testing.T) {
	_, err := ParsePattern("?")
	if err == nil {
		t.Fatal("expected error for bare '?', got nil")
	}
}

func TestParsePattern_DoubleQuotedString(t *testing.T) {
	p, err := ParsePattern(`"my file"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementQuoted {
		t.Errorf("expected type ElementQuoted, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "my file" {
		t.Errorf("expected value %q, got %q", "my file", p.Elements[0].Value)
	}
}

func TestParsePattern_SingleQuotedString(t *testing.T) {
	p, err := ParsePattern(`'my file'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementQuoted {
		t.Errorf("expected type ElementQuoted, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "my file" {
		t.Errorf("expected value %q, got %q", "my file", p.Elements[0].Value)
	}
}

func TestParsePattern_EscapedWildcard(t *testing.T) {
	p, err := ParsePattern(`\*`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementQuoted {
		t.Errorf("expected type ElementQuoted, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "*" {
		t.Errorf("expected value %q, got %q", "*", p.Elements[0].Value)
	}
}

func TestParsePattern_EscapedDot(t *testing.T) {
	p, err := ParsePattern(`\.`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementQuoted {
		t.Errorf("expected type ElementQuoted, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != "." {
		t.Errorf("expected value %q, got %q", ".", p.Elements[0].Value)
	}
}

func TestParsePattern_UnclosedDoubleQuote(t *testing.T) {
	_, err := ParsePattern(`"my file`)
	if err == nil {
		t.Fatal("expected error for unclosed double quote, got nil")
	}
}

func TestParsePattern_UnclosedSingleQuote(t *testing.T) {
	_, err := ParsePattern(`'my file`)
	if err == nil {
		t.Fatal("expected error for unclosed single quote, got nil")
	}
}

func TestParsePattern_QuotedWithOtherTokens(t *testing.T) {
	p, err := ParsePattern(`push "my file" --force`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementLiteral || p.Elements[0].Value != "push" {
		t.Errorf("element 0: expected literal 'push', got %q %q", p.Elements[0].Type, p.Elements[0].Value)
	}
	if p.Elements[1].Type != ElementQuoted || p.Elements[1].Value != "my file" {
		t.Errorf("element 1: expected quoted 'my file', got %q %q", p.Elements[1].Type, p.Elements[1].Value)
	}
	if p.Elements[2].Type != ElementLiteral || p.Elements[2].Value != "--force" {
		t.Errorf("element 2: expected literal '--force', got %q %q", p.Elements[2].Type, p.Elements[2].Value)
	}
}

func TestParsePattern_QuotedWithEscapedQuoteInside(t *testing.T) {
	p, err := ParsePattern(`"my \"file\""`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	if p.Elements[0].Type != ElementQuoted {
		t.Errorf("expected type ElementQuoted, got %q", p.Elements[0].Type)
	}
	if p.Elements[0].Value != `my "file"` {
		t.Errorf("expected value %q, got %q", `my "file"`, p.Elements[0].Value)
	}
}

func TestParsePattern_GroupOptional(t *testing.T) {
	p, err := ParsePattern("(origin main)?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	elem := p.Elements[0]
	if elem.Type != ElementGroup {
		t.Errorf("expected type ElementGroup, got %q", elem.Type)
	}
	if !elem.Optional {
		t.Error("expected Optional=true")
	}
	if elem.NonPositional {
		t.Error("expected NonPositional=false")
	}
	if len(elem.Group) != 2 {
		t.Fatalf("expected 2 sub-elements, got %d", len(elem.Group))
	}
	if elem.Group[0].Type != ElementLiteral || elem.Group[0].Value != "origin" {
		t.Errorf("sub-element 0: expected literal 'origin', got %q %q", elem.Group[0].Type, elem.Group[0].Value)
	}
	if elem.Group[1].Type != ElementLiteral || elem.Group[1].Value != "main" {
		t.Errorf("sub-element 1: expected literal 'main', got %q %q", elem.Group[1].Type, elem.Group[1].Value)
	}
}

func TestParsePattern_GroupNonPositional(t *testing.T) {
	p, err := ParsePattern("~(-n 0)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	elem := p.Elements[0]
	if elem.Type != ElementGroup {
		t.Errorf("expected type ElementGroup, got %q", elem.Type)
	}
	if !elem.NonPositional {
		t.Error("expected NonPositional=true")
	}
	if elem.Optional {
		t.Error("expected Optional=false")
	}
	if len(elem.Group) != 2 {
		t.Fatalf("expected 2 sub-elements, got %d", len(elem.Group))
	}
	if elem.Group[0].Type != ElementLiteral || elem.Group[0].Value != "-n" {
		t.Errorf("sub-element 0: expected literal '-n', got %q %q", elem.Group[0].Type, elem.Group[0].Value)
	}
	if elem.Group[1].Type != ElementLiteral || elem.Group[1].Value != "0" {
		t.Errorf("sub-element 1: expected literal '0', got %q %q", elem.Group[1].Type, elem.Group[1].Value)
	}
}

func TestParsePattern_GroupPlain(t *testing.T) {
	p, err := ParsePattern("(origin main)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(p.Elements))
	}
	elem := p.Elements[0]
	if elem.Type != ElementGroup {
		t.Errorf("expected type ElementGroup, got %q", elem.Type)
	}
	if elem.Optional {
		t.Error("expected Optional=false")
	}
	if elem.NonPositional {
		t.Error("expected NonPositional=false")
	}
	if len(elem.Group) != 2 {
		t.Fatalf("expected 2 sub-elements, got %d", len(elem.Group))
	}
	if elem.Group[0].Type != ElementLiteral || elem.Group[0].Value != "origin" {
		t.Errorf("sub-element 0: expected literal 'origin', got %q %q", elem.Group[0].Type, elem.Group[0].Value)
	}
	if elem.Group[1].Type != ElementLiteral || elem.Group[1].Value != "main" {
		t.Errorf("sub-element 1: expected literal 'main', got %q %q", elem.Group[1].Type, elem.Group[1].Value)
	}
}

func TestParsePattern_UnclosedParen(t *testing.T) {
	_, err := ParsePattern("(origin main")
	if err == nil {
		t.Fatal("expected error for unclosed paren, got nil")
	}
}

func TestParsePattern_RawPreserved(t *testing.T) {
	raw := "  push  --force$  "
	p, err := ParsePattern(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Raw != raw {
		t.Errorf("expected Raw=%q, got %q", raw, p.Raw)
	}
}
