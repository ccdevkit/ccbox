package permissions

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// T046: Validate that the quickstart.md example parses and evaluates correctly.

const quickstartYAML = `
passthrough:
  # Allow all git commands except force push
  git:
    rules:
      - pattern: "**"
        effect: allow
      - pattern: "push ~--force"
        effect: deny
        reason: "Force push is destructive — use regular push"

  # Only allow safe npm commands
  npm:
    rules:
      - pattern: "**"
        effect: deny
      - pattern: "install"
        effect: allow
      - pattern: "ci"
        effect: allow
      - pattern: "run build"
        effect: allow

  # Allow docker with no restrictions
  docker:
`

func TestQuickstartExample_ParsesAndEvaluates(t *testing.T) {
	var config PermissionsConfig
	if err := yaml.Unmarshal([]byte(quickstartYAML), &config); err != nil {
		t.Fatalf("failed to parse quickstart YAML: %v", err)
	}

	checker, err := NewChecker(&config, nil)
	if err != nil {
		t.Fatalf("failed to create checker: %v", err)
	}

	tests := []struct {
		command string
		allowed bool
		desc    string
	}{
		{"git pull origin", true, "git pull allowed by ** rule"},
		{"git push origin main", true, "git push (no --force) allowed by ** rule"},
		{"git push --force origin main", false, "git push --force denied by deny rule"},
		{"npm install", true, "npm install explicitly allowed"},
		{"npm ci", true, "npm ci explicitly allowed"},
		{"npm run build", true, "npm run build explicitly allowed"},
		{"npm publish", false, "npm publish not in allowed list, denied by ** deny"},
		{"docker ps", true, "docker unrestricted (null config)"},
		{"docker rm -f container", true, "docker unrestricted (null config)"},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := checker.Check(tt.command)
			if result.Allowed != tt.allowed {
				if tt.allowed {
					t.Errorf("expected allowed (%s), got denied: %s", tt.desc, result.Reason)
				} else {
					t.Errorf("expected denied (%s), got allowed", tt.desc)
				}
			}
		})
	}
}
