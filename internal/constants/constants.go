package constants

// Container paths
const (
	// ContainerOptDir is the base directory for ccbox files inside the container.
	ContainerOptDir = "/opt/ccbox/"

	// ContainerBinDir is the directory for ccbox binaries inside the container (prepended to PATH).
	ContainerBinDir = "/opt/ccbox/bin/"

	// ContainerShimsDir is the directory for command hijacker shim scripts (prepended to PATH before ContainerBinDir).
	ContainerShimsDir = "/opt/ccbox/bin/shims/"

	// ContainerHomeDir is the home directory for the unprivileged claude user inside the container.
	ContainerHomeDir = "/home/claude/"

	// SystemPromptContainerPath is the stable container path for the injected system prompt file.
	// Referenced by --append-system-prompt-file CLI arg.
	SystemPromptContainerPath = "/opt/ccbox/ccbox-system-prompt.md"

	// ProxyConfigContainerPath is the container path for the ccptproxy configuration file.
	ProxyConfigContainerPath = "/opt/ccbox/ccbox-proxy.json"

	// SettingsContainerPath is the container path for the Claude Code settings override file.
	SettingsContainerPath = "/opt/ccbox/settings.json"
)

// Docker image configuration
const (
	// BaseImageRegistry is the base Docker image registry path for ccbox.
	BaseImageRegistry = "ghcr.io/ccdevkit/ccbox-base"

	// ImageNamePrefix is the local Docker image name prefix.
	ImageNamePrefix = "ccbox-local"
)

// Bridge directory for clipboard file drag-drop
const (
	// BridgeDirName is the host-side bridge directory name (created under user home).
	BridgeDirName = ".ccbox-bridge"

	// ContainerBridgeDir is the container-side path for the bridge directory mount.
	ContainerBridgeDir = "/home/claude/.ccbox-bridge"
)

// Container user configuration
const (
	// ContainerUserUID is the UID for the unprivileged claude user in the container.
	// Avoids conflict with Node.js UID 1000.
	ContainerUserUID = 1001
)

// Environment variable names
const (
	// EnvClaudeOAuthToken is the env var for passing the OAuth credential to the container.
	EnvClaudeOAuthToken = "CLAUDE_CODE_OAUTH_TOKEN"

	// EnvTerm is the env var for terminal type forwarding (FR-027).
	EnvTerm = "TERM"

	// EnvColorTerm is the env var for color support forwarding (FR-027).
	EnvColorTerm = "COLORTERM"

	// EnvCCBoxTCPPort is the env var for the dynamic TCP bridge port.
	EnvCCBoxTCPPort = "CCBOX_TCP_PORT"

	// EnvCCBoxClipPort is the env var for the clipboard daemon communication port.
	EnvCCBoxClipPort = "CCBOX_CLIP_PORT"

	// EnvDisplay is the env var for the Xvfb virtual display.
	EnvDisplay = "DISPLAY"
)

// Default values
const (
	// DefaultDisplay is the hardcoded DISPLAY value for Xvfb inside the container.
	DefaultDisplay = ":99"

	// DefaultClaudePath is the default path to the claude CLI on the host.
	DefaultClaudePath = "claude"
)

// Config file names
const (
	// ProxyConfigFileName is the filename for the ccptproxy configuration.
	ProxyConfigFileName = "ccbox-proxy.json"

	// SettingsFileName is the filename for the Claude Code settings override.
	SettingsFileName = "settings.json"

	// SettingsDirName is the directory name for ccbox settings discovery.
	SettingsDirName = ".ccbox"

	// ClaudeConfigDirName is the Claude Code config directory name.
	ClaudeConfigDirName = ".claude"
)

// Wire protocol constants
const (
	// ExecRequestType is the JSON type field value for exec requests.
	ExecRequestType = "exec"

	// LogRequestType is the JSON type field value for log requests.
	LogRequestType = "log"
)

// Clipboard bridge constants
const (
	// MaxClipboardPayload is the maximum clipboard image payload size (50 MB).
	MaxClipboardPayload = 50 * 1024 * 1024

	// ClipboardStatusSuccess is the success status byte from ccclipd.
	ClipboardStatusSuccess = 0x00

	// ClipboardStatusError is the error status byte from ccclipd.
	ClipboardStatusError = 0x01
)

// Docker networking
const (
	// DockerHostname is the hostname used by containers to reach the host.
	DockerHostname = "host.docker.internal"

	// TCPServerBindAddress is the address the host TCP server binds to.
	TCPServerBindAddress = "127.0.0.1:0"
)

// Passthrough output annotation
const (
	// PassthroughNote is prepended to all passthrough command output by ccptproxy.
	PassthroughNote = "[NOTE: This command was run on the host machine]"
)

// Session temp directory prefix
const (
	// TempDirPrefix is the prefix for session temporary directories.
	TempDirPrefix = "ccbox-"
)

// Log dial timeout (in seconds) for container→host log messages.
const (
	// LogDialTimeoutSec is the TCP dial timeout for log requests.
	LogDialTimeoutSec = 2
)
