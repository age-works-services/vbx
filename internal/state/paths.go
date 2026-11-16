package state

import "path/filepath"

// Paths resolves XDG locations lazily, when the operation needs them.
type Paths struct{}

func (Paths) ConfigDir() (string, error) { return Base("XDG_CONFIG_HOME", ".config") }
func (Paths) StateDir() (string, error)  { return Base("XDG_STATE_HOME", ".local/state") }
func (p Paths) SSHDir() (string, error) {
	base, err := p.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "ssh"), nil
}

type InstancePaths struct{ Dir, SSHLink string }

func (p Paths) Instance(name string) (InstancePaths, error) {
	base, err := p.StateDir()
	if err != nil {
		return InstancePaths{}, err
	}
	ssh, err := p.SSHDir()
	if err != nil {
		return InstancePaths{}, err
	}
	return InstancePaths{filepath.Join(base, "instances", name), filepath.Join(ssh, name+".conf")}, nil
}
