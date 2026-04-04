package session

import "testing"

func TestDockerBindMountProvider_AddPassthrough(t *testing.T) {
	p := NewDockerBindMountProvider()
	err := p.AddPassthrough("/host/path", "/container/path", true)
	if err != nil {
		t.Fatalf("AddPassthrough returned error: %v", err)
	}

	if len(p.Passthroughs) != 1 {
		t.Fatalf("expected 1 passthrough, got %d", len(p.Passthroughs))
	}

	pt := p.Passthroughs[0]
	if pt.HostPath != "/host/path" {
		t.Errorf("HostPath = %q, want %q", pt.HostPath, "/host/path")
	}
	if pt.ContainerPath != "/container/path" {
		t.Errorf("ContainerPath = %q, want %q", pt.ContainerPath, "/container/path")
	}
	if pt.ReadOnly != true {
		t.Errorf("ReadOnly = %v, want true", pt.ReadOnly)
	}
}

func TestDockerBindMountProvider_MultipleAdds(t *testing.T) {
	p := NewDockerBindMountProvider()

	_ = p.AddPassthrough("/a", "/b", false)
	_ = p.AddPassthrough("/c", "/d", true)
	_ = p.AddPassthrough("/e", "/f", false)

	if len(p.Passthroughs) != 3 {
		t.Fatalf("expected 3 passthroughs, got %d", len(p.Passthroughs))
	}

	if p.Passthroughs[1].HostPath != "/c" {
		t.Errorf("second entry HostPath = %q, want %q", p.Passthroughs[1].HostPath, "/c")
	}
	if p.Passthroughs[2].ReadOnly != false {
		t.Errorf("third entry ReadOnly = %v, want false", p.Passthroughs[2].ReadOnly)
	}
}

func TestDockerBindMountProvider_ImplementsFilePassthroughProvider(t *testing.T) {
	var _ FilePassthroughProvider = (*DockerBindMountProvider)(nil)
}
