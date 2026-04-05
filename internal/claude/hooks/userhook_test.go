package hooks

import (
	"encoding/json"
	"testing"
)

// --- ParseIfField tests ---

func TestParseIfField_BashWithGlob(t *testing.T) {
	tool, pattern, err := ParseIfField("Bash(rm *)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool != "Bash" {
		t.Errorf("tool = %q, want %q", tool, "Bash")
	}
	if pattern != "rm *" {
		t.Errorf("pattern = %q, want %q", pattern, "rm *")
	}
}

func TestParseIfField_EditWithExtension(t *testing.T) {
	tool, pattern, err := ParseIfField("Edit(*.ts)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool != "Edit" {
		t.Errorf("tool = %q, want %q", tool, "Edit")
	}
	if pattern != "*.ts" {
		t.Errorf("pattern = %q, want %q", pattern, "*.ts")
	}
}

func TestParseIfField_ExactCommand(t *testing.T) {
	tool, pattern, err := ParseIfField("Bash(npm run build)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool != "Bash" {
		t.Errorf("tool = %q, want %q", tool, "Bash")
	}
	if pattern != "npm run build" {
		t.Errorf("pattern = %q, want %q", pattern, "npm run build")
	}
}

func TestParseIfField_CatchAll(t *testing.T) {
	tool, pattern, err := ParseIfField("WebSearch(*)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool != "WebSearch" {
		t.Errorf("tool = %q, want %q", tool, "WebSearch")
	}
	if pattern != "*" {
		t.Errorf("pattern = %q, want %q", pattern, "*")
	}
}

func TestParseIfField_MCPToolEmptyPattern(t *testing.T) {
	tool, pattern, err := ParseIfField("mcp__puppeteer__puppeteer_navigate()")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool != "mcp__puppeteer__puppeteer_navigate" {
		t.Errorf("tool = %q, want %q", tool, "mcp__puppeteer__puppeteer_navigate")
	}
	if pattern != "" {
		t.Errorf("pattern = %q, want %q", pattern, "")
	}
}

func TestParseIfField_Empty(t *testing.T) {
	tool, pattern, err := ParseIfField("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool != "" {
		t.Errorf("tool = %q, want %q", tool, "")
	}
	if pattern != "" {
		t.Errorf("pattern = %q, want %q", pattern, "")
	}
}

func TestParseIfField_NoParens(t *testing.T) {
	// "Bash" without parens means match all uses of Bash
	tool, pattern, err := ParseIfField("Bash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool != "Bash" {
		t.Errorf("tool = %q, want %q", tool, "Bash")
	}
	if pattern != "" {
		t.Errorf("pattern = %q, want %q", pattern, "")
	}
}

func TestParseIfField_WildcardInMiddle(t *testing.T) {
	tool, pattern, err := ParseIfField("Bash(git * main)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool != "Bash" {
		t.Errorf("tool = %q, want %q", tool, "Bash")
	}
	if pattern != "git * main" {
		t.Errorf("pattern = %q, want %q", pattern, "git * main")
	}
}

func TestParseIfField_WildcardAtStart(t *testing.T) {
	tool, pattern, err := ParseIfField("Bash(* --version)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool != "Bash" {
		t.Errorf("tool = %q, want %q", tool, "Bash")
	}
	if pattern != "* --version" {
		t.Errorf("pattern = %q, want %q", pattern, "* --version")
	}
}

func TestParseIfField_DoubleStarPath(t *testing.T) {
	tool, pattern, err := ParseIfField("Edit(/src/**/*.ts)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool != "Edit" {
		t.Errorf("tool = %q, want %q", tool, "Edit")
	}
	if pattern != "/src/**/*.ts" {
		t.Errorf("pattern = %q, want %q", pattern, "/src/**/*.ts")
	}
}

func TestParseIfField_Malformed_UnclosedParen(t *testing.T) {
	_, _, err := ParseIfField("Bash(unclosed")
	if err == nil {
		t.Fatal("expected error for unclosed paren")
	}
}

func TestParseIfField_Malformed_NoToolName(t *testing.T) {
	_, _, err := ParseIfField("(no tool)")
	if err == nil {
		t.Fatal("expected error for missing tool name")
	}
}

func TestParseIfField_NestedParens(t *testing.T) {
	// "Bash(echo (hello))" — the pattern should be "echo (hello)"
	tool, pattern, err := ParseIfField("Bash(echo (hello))")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tool != "Bash" {
		t.Errorf("tool = %q, want %q", tool, "Bash")
	}
	if pattern != "echo (hello)" {
		t.Errorf("pattern = %q, want %q", pattern, "echo (hello)")
	}
}

// --- GlobMatch tests ---

func TestGlobMatch_BasicWildcard(t *testing.T) {
	tests := []struct {
		pattern string
		subject string
		want    bool
	}{
		// Basic wildcard at end
		{"rm *", "rm -rf /tmp", true},
		{"rm *", "ls -la", false},
		{"npm run *", "npm run build", true},
		{"npm run *", "npm run test", true},
		{"npm run *", "npm install", false},

		// Catch-all
		{"*", "anything at all", true},
		{"*", "", true},

		// Empty pattern matches empty string
		{"", "", true},
		{"", "non-empty", false},

		// Wildcard in middle
		{"git * main", "git checkout main", true},
		{"git * main", "git merge main", true},
		{"git * main", "git status", false},
		{"git * main", "git rebase feature main", true},

		// Wildcard at start
		{"* --version", "node --version", true},
		{"* --version", "npm --version", true},
		{"* --version", "node --help", false},

		// Word boundary: "ls *" (space before *) requires "ls" followed by space or end
		{"ls *", "ls -la", true},
		{"ls *", "ls", false}, // "ls *" requires a space after "ls"
		{"ls *", "lsof", false},

		// No space before * — no word boundary
		{"ls*", "ls -la", true},
		{"ls*", "lsof", true},
		{"ls*", "ls", true},

		// File extensions
		{"*.ts", "foo.ts", true},
		{"*.ts", "src/bar.ts", true},
		{"*.ts", "foo.tsx", false},
		{"*.ts", "foo.js", false},

		// Double-star for nested paths
		{"/src/**/*.ts", "/src/components/Button.ts", true},
		{"/src/**/*.ts", "/src/utils/deep/nested/helper.ts", true},
		{"/src/**/*.ts", "/src/index.ts", true},
		{"/src/**/*.ts", "/lib/index.ts", false},

		// Literal dot (regex metachar must be escaped)
		{"file.txt", "file.txt", true},
		{"file.txt", "fileatxt", false},

		// Literal special chars
		{"hello+world", "hello+world", true},
		{"hello+world", "helloworld", false},

		// Exact match (no wildcards)
		{"npm run build", "npm run build", true},
		{"npm run build", "npm run test", false},

		// Multiple wildcards
		{"* run *", "npm run build", true},
		{"* run *", "yarn run test", true},
		{"* run *", "npm install", false},
	}
	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.subject, func(t *testing.T) {
			got := GlobMatch(tt.pattern, tt.subject)
			if got != tt.want {
				t.Errorf("GlobMatch(%q, %q) = %v, want %v", tt.pattern, tt.subject, got, tt.want)
			}
		})
	}
}

// --- MatchIf tests ---

func makeToolInput(toolName string, fields map[string]interface{}) json.RawMessage {
	m := map[string]interface{}{
		"hook_event_name": "PreToolUse",
		"tool_name":       toolName,
	}
	if fields != nil {
		m["tool_input"] = fields
	}
	data, _ := json.Marshal(m)
	return data
}

func TestMatchIf_BashRmMatches(t *testing.T) {
	input := makeToolInput("Bash", map[string]interface{}{"command": "rm -rf /tmp"})
	if !MatchIf("Bash(rm *)", PreToolUse, input) {
		t.Error("expected match for Bash rm command")
	}
}

func TestMatchIf_BashRmNoMatch(t *testing.T) {
	input := makeToolInput("Bash", map[string]interface{}{"command": "npm test"})
	if MatchIf("Bash(rm *)", PreToolUse, input) {
		t.Error("expected no match for npm test with Bash(rm *)")
	}
}

func TestMatchIf_ToolNameMismatch(t *testing.T) {
	input := makeToolInput("Edit", map[string]interface{}{"file_path": "foo.ts"})
	if MatchIf("Bash(rm *)", PreToolUse, input) {
		t.Error("expected no match for Edit tool with Bash if-field")
	}
}

func TestMatchIf_EditTsMatches(t *testing.T) {
	input := makeToolInput("Edit", map[string]interface{}{"file_path": "src/foo.ts"})
	if !MatchIf("Edit(*.ts)", PreToolUse, input) {
		t.Error("expected match for Edit .ts file")
	}
}

func TestMatchIf_EditTsNoMatch(t *testing.T) {
	input := makeToolInput("Edit", map[string]interface{}{"file_path": "src/foo.go"})
	if MatchIf("Edit(*.ts)", PreToolUse, input) {
		t.Error("expected no match for .go file with Edit(*.ts)")
	}
}

func TestMatchIf_NonToolEventNeverFires(t *testing.T) {
	input, _ := json.Marshal(map[string]interface{}{
		"hook_event_name": "SessionStart",
	})
	if MatchIf("Bash(rm *)", SessionStart, input) {
		t.Error("hooks with if on non-tool events should never fire")
	}
}

func TestMatchIf_EmptyIfAlwaysMatches(t *testing.T) {
	input := makeToolInput("Bash", map[string]interface{}{"command": "anything"})
	if !MatchIf("", PreToolUse, input) {
		t.Error("empty if field should always match")
	}
}

func TestMatchIf_NoParensMatchesAllToolUses(t *testing.T) {
	input := makeToolInput("Bash", map[string]interface{}{"command": "anything"})
	if !MatchIf("Bash", PreToolUse, input) {
		t.Error("if field without parens should match all Bash calls")
	}
}

func TestMatchIf_NoParensToolMismatch(t *testing.T) {
	input := makeToolInput("Edit", map[string]interface{}{"file_path": "foo.ts"})
	if MatchIf("Bash", PreToolUse, input) {
		t.Error("Bash if-field should not match Edit tool")
	}
}

func TestMatchIf_ReadFilePath(t *testing.T) {
	input := makeToolInput("Read", map[string]interface{}{"file_path": "/etc/passwd"})
	if !MatchIf("Read(/etc/*)", PreToolUse, input) {
		t.Error("expected match for Read /etc/passwd")
	}
}

func TestMatchIf_WriteFilePath(t *testing.T) {
	input := makeToolInput("Write", map[string]interface{}{"file_path": "output.json"})
	if !MatchIf("Write(*.json)", PreToolUse, input) {
		t.Error("expected match for Write .json file")
	}
}

func TestMatchIf_PostToolUse(t *testing.T) {
	input := makeToolInput("Bash", map[string]interface{}{"command": "git push"})
	// PostToolUse is also a tool event
	if !MatchIf("Bash(git *)", PostToolUse, input) {
		t.Error("expected match for PostToolUse git command")
	}
}

func TestMatchIf_PermissionRequest(t *testing.T) {
	input := makeToolInput("Bash", map[string]interface{}{"command": "rm -rf /"})
	if !MatchIf("Bash(rm *)", PermissionRequest, input) {
		t.Error("expected match for PermissionRequest rm command")
	}
}

func TestMatchIf_EmptyIfOnNonToolEvent(t *testing.T) {
	input, _ := json.Marshal(map[string]interface{}{
		"hook_event_name": "Stop",
	})
	// Empty if field always matches, even on non-tool events
	if !MatchIf("", Stop, input) {
		t.Error("empty if field should match on non-tool events too")
	}
}

// --- extractIfSubject tests ---

func TestExtractIfSubject_Bash(t *testing.T) {
	input := makeToolInput("Bash", map[string]interface{}{"command": "npm test"})
	subject := extractIfSubject("Bash", input)
	if subject != "npm test" {
		t.Errorf("subject = %q, want %q", subject, "npm test")
	}
}

func TestExtractIfSubject_Edit(t *testing.T) {
	input := makeToolInput("Edit", map[string]interface{}{"file_path": "src/main.go"})
	subject := extractIfSubject("Edit", input)
	if subject != "src/main.go" {
		t.Errorf("subject = %q, want %q", subject, "src/main.go")
	}
}

func TestExtractIfSubject_Write(t *testing.T) {
	input := makeToolInput("Write", map[string]interface{}{"file_path": "output.txt"})
	subject := extractIfSubject("Write", input)
	if subject != "output.txt" {
		t.Errorf("subject = %q, want %q", subject, "output.txt")
	}
}

func TestExtractIfSubject_Read(t *testing.T) {
	input := makeToolInput("Read", map[string]interface{}{"file_path": "/etc/passwd"})
	subject := extractIfSubject("Read", input)
	if subject != "/etc/passwd" {
		t.Errorf("subject = %q, want %q", subject, "/etc/passwd")
	}
}

func TestExtractIfSubject_UnknownTool(t *testing.T) {
	input := makeToolInput("CustomTool", map[string]interface{}{"key": "value"})
	subject := extractIfSubject("CustomTool", input)
	if subject != "" {
		t.Errorf("subject = %q, want empty for unknown tool", subject)
	}
}

// --- matchesUserHook tests (matcher regex matching) ---

func TestMatchesUserHook_ExactToolName(t *testing.T) {
	h := UserHook{Matcher: "Bash"}
	input := makeToolInput("Bash", nil)
	if !matchesUserHook(h, PreToolUse, input) {
		t.Error("expected Bash matcher to match Bash tool")
	}
}

func TestMatchesUserHook_AlternationMatcher(t *testing.T) {
	h := UserHook{Matcher: "Edit|Write"}
	inputEdit := makeToolInput("Edit", nil)
	inputWrite := makeToolInput("Write", nil)
	inputBash := makeToolInput("Bash", nil)

	if !matchesUserHook(h, PreToolUse, inputEdit) {
		t.Error("Edit|Write should match Edit")
	}
	if !matchesUserHook(h, PreToolUse, inputWrite) {
		t.Error("Edit|Write should match Write")
	}
	if matchesUserHook(h, PreToolUse, inputBash) {
		t.Error("Edit|Write should not match Bash")
	}
}

func TestMatchesUserHook_MCPWildcard(t *testing.T) {
	h := UserHook{Matcher: "mcp__.*"}
	input := makeToolInput("mcp__memory__read", nil)
	if !matchesUserHook(h, PreToolUse, input) {
		t.Error("mcp__.* should match mcp__memory__read")
	}
}

func TestMatchesUserHook_EmptyMatcherMatchesAll(t *testing.T) {
	h := UserHook{Matcher: ""}
	input := makeToolInput("AnyTool", nil)
	if !matchesUserHook(h, PreToolUse, input) {
		t.Error("empty matcher should match everything")
	}
}

func TestMatchesUserHook_NonToolEventMatchesAgainstEventName(t *testing.T) {
	h := UserHook{Matcher: "SessionStart"}
	input, _ := json.Marshal(map[string]interface{}{
		"hook_event_name": "SessionStart",
	})
	// For non-tool events the matcher matches against hook_event_name
	if !matchesUserHook(h, SessionStart, input) {
		t.Error("matcher should match hook_event_name for non-tool events")
	}
}

func TestMatchesUserHook_NonToolEventNoMatch(t *testing.T) {
	h := UserHook{Matcher: "resume"}
	input, _ := json.Marshal(map[string]interface{}{
		"hook_event_name": "SessionStart",
	})
	if matchesUserHook(h, SessionStart, input) {
		t.Error("resume matcher should not match SessionStart event name")
	}
}
