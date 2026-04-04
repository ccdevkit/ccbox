package matcher

import "testing"

func TestCommandMatcher_Matches(t *testing.T) {
	tests := []struct {
		name    string
		cmds    []string
		input   string
		want    bool
	}{
		{
			name:  "exact match single word",
			cmds:  []string{"git"},
			input: "git",
			want:  true,
		},
		{
			name:  "first word matches with args",
			cmds:  []string{"git"},
			input: "git status",
			want:  true,
		},
		{
			name:  "prefix does not match different command",
			cmds:  []string{"git"},
			input: "gitk",
			want:  false,
		},
		{
			name:  "no match",
			cmds:  []string{"git"},
			input: "docker ps",
			want:  false,
		},
		{
			name:  "multiple commands first matches",
			cmds:  []string{"git", "docker"},
			input: "git push origin main",
			want:  true,
		},
		{
			name:  "multiple commands second matches",
			cmds:  []string{"git", "docker"},
			input: "docker ps",
			want:  true,
		},
		{
			name:  "multiple commands none match",
			cmds:  []string{"git", "docker"},
			input: "npm install",
			want:  false,
		},
		{
			name:  "empty input",
			cmds:  []string{"git"},
			input: "",
			want:  false,
		},
		{
			name:  "empty command list",
			cmds:  []string{},
			input: "git status",
			want:  false,
		},
		{
			name:  "leading whitespace in input",
			cmds:  []string{"git"},
			input: "  git status",
			want:  true,
		},
		{
			name:  "tab-separated args",
			cmds:  []string{"git"},
			input: "git\tstatus",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewCommandMatcher(tt.cmds)
			got := m.Matches(tt.input)
			if got != tt.want {
				t.Errorf("Matches(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
