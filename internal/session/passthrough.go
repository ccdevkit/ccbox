package session

// FilePassthrough represents a host file or directory to mount into the container.
type FilePassthrough struct {
	HostPath      string
	ContainerPath string
	ReadOnly      bool
}

// DockerBindMountProvider collects host paths to bind-mount into the container.
type DockerBindMountProvider struct {
	Passthroughs []FilePassthrough
}

// NewDockerBindMountProvider returns an empty DockerBindMountProvider.
func NewDockerBindMountProvider() *DockerBindMountProvider {
	return &DockerBindMountProvider{}
}

// AddPassthrough appends a file passthrough entry.
func (p *DockerBindMountProvider) AddPassthrough(hostPath, containerPath string, readOnly bool) error {
	p.Passthroughs = append(p.Passthroughs, FilePassthrough{
		HostPath:      hostPath,
		ContainerPath: containerPath,
		ReadOnly:      readOnly,
	})
	return nil
}
