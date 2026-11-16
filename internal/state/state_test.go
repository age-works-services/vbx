package state

import (
	"os"
	"path/filepath"
	"testing"
	"vbx/internal/config"
)

func TestSnapshot(t *testing.T) {
	c := config.Defaults()
	c.Folder = "/workspaces/app"
	b, e := Encode(c)
	if e != nil {
		t.Fatal(e)
	}
	got, e := Decode(b)
	if e != nil || got.Default != c.Default {
		t.Fatal(got, e)
	}
	for _, b := range [][]byte{[]byte(`{"Version":9}`), []byte("not-json"), []byte("invalid\x00data"), []byte(`{"Version":1,"Config":{}}`)} {
		if _, e := Decode(b); e == nil {
			t.Fatal("accepted broken snapshot")
		}
	}
}
func TestLock(t *testing.T) {
	base := t.TempDir()
	release, e := Lock(base, "name")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = Lock(base, "name"); e == nil {
		t.Fatal("duplicate lock")
	}
	release()
	release, e = Lock(base, "name")
	if e != nil {
		t.Fatal(e)
	}
	release()
	path := filepath.Join(base, "data")
	if e = Atomic(path, []byte("secret"), 0600); e != nil {
		t.Fatal(e)
	}
	s, _ := os.Stat(path)
	if s.Mode().Perm() != 0600 {
		t.Fatal(s.Mode())
	}
}
func TestXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "relative")
	if _, e := Base("XDG_CONFIG_HOME", ".config"); e == nil {
		t.Fatal("relative XDG accepted")
	}
}
