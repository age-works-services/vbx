package project

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIdentity(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"app", "ABCDEFGHIJKLMNOPQRSTUVWXYZ", "!!!", "K", "abcdefghijklmnopqrstuvw-zzz"} {
		p := filepath.Join(root, name)
		os.Mkdir(p, 0700)
		got, e := Resolve(p)
		if e != nil {
			t.Fatal(e)
		}
		digest := sha256.Sum256([]byte(p))
		if !strings.HasSuffix(got.Name, fmt.Sprintf("-%x", digest[:6])) {
			t.Fatal(got)
		}
		if name == "K" && !strings.HasPrefix(got.Name, "vbx-project-") {
			t.Fatal("Unicode lowercase changed identity")
		}
		if name == "abcdefghijklmnopqrstuvw-zzz" && !strings.HasPrefix(got.Name, "vbx-abcdefghijklmnopqrstuvw--") {
			t.Fatal("truncation changed identity")
		}
		link := filepath.Join(root, "link")
		os.Remove(link)
		os.Symlink(p, link)
		alias, e := Resolve(link)
		if e != nil || alias != got {
			t.Fatal(alias, e)
		}
	}
	for _, p := range []string{"/", filepath.Join(root, "missing")} {
		if _, e := Resolve(p); e == nil {
			t.Fatal("accepted", p)
		}
	}
}
