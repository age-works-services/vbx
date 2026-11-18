package lima

import (
	"testing"
	"testing/fstest"
	"vbx/internal/config"
	"vbx/internal/project"
	"vbx/internal/state"
)

func TestRenderWorkspaceMountCache(t *testing.T) {
	assets := fstest.MapFS{"lima/default.yaml": {Data: []byte("mountType: 9p\nmounts: [] # VBX_PROJECT_MOUNT\n")}}
	c := config.Defaults()
	c.Mounts = []config.Mount{
		{Source: "/host/extra", Target: "/workspaces/extra"},
		{Source: "/host/reference", Target: "/workspaces/reference", Readonly: true},
	}
	p := project.Project{Path: "/host/project", Primary: "/workspaces/project"}
	got, err := Render(assets, c, p, state.Binding{Host: "/host/data", Guest: "/persist/data", Writable: true})
	if err != nil {
		t.Fatal(err)
	}
	want := `mountType: 9p
mounts:
  - location: '/host/project'
    mountPoint: '/workspaces/project'
    writable: true
    9p:
      cache: loose
  - location: '/host/extra'
    mountPoint: '/workspaces/extra'
    writable: true
    9p:
      cache: loose
  - location: '/host/reference'
    mountPoint: '/workspaces/reference'
    writable: false
  - location: '/host/data'
    mountPoint: '/persist/data'
    writable: true
`
	if string(got) != want {
		t.Fatalf("rendered config:\n%s\nwant:\n%s", got, want)
	}
}
