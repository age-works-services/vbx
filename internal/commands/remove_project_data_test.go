package commands

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"vbx/internal/backend/lima"
	"vbx/internal/command"
	"vbx/internal/config"
	"vbx/internal/module"
	"vbx/internal/state"
)

func TestRemoveProjectData(t *testing.T) {
	for _, status := range []string{"Running", "Stopped", "absent", "cancel"} {
		t.Run(status, func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			t.Setenv("XDG_STATE_HOME", t.TempDir())
			paths := state.Paths{}
			id := "vbx-project-123456abcdef"
			other := "vbx-other-abcdef123456"
			c := config.Defaults()
			c.Folder = "/workspaces/project"
			c.Modules = []string{"sessions"}
			c.Persistence = []config.PersistencePath{{Guest: "~/.tool/settings", Data: "settings"}}
			defs := []module.Definition{{Name: "sessions"}}
			r, err := paths.AllocateData(id, c, defs, t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			sibling, err := paths.AllocateData(other, c, defs, t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			configMarker := filepath.Join(r.Bindings[1].Host, "settings")
			if err = os.WriteFile(configMarker, []byte("valuable"), 0600); err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(r.Bindings[0].Host, "history")
			os.WriteFile(marker, []byte("session"), 0600)
			inventory := &inventoryStub{}
			if status == "Running" || status == "Stopped" {
				inventory.instances = []lima.Instance{{Name: id, Status: status}}
			}
			answer := id
			if status == "cancel" {
				answer = "wrong"
			}
			cmd := newRemoveProjectDataCommand(Dependencies{Paths: paths, Inventory: inventory, Interaction: promptStub(answer), Out: io.Discard})
			err = cmd.Run(context.Background(), command.NewInvocation(command.GeneralOptions{}, nil, []string{id}))
			if status == "absent" {
				if err != nil {
					t.Fatal(err)
				}
				if _, err = os.Stat(marker); !os.IsNotExist(err) {
					t.Fatal("data retained", err)
				}
			} else {
				if err == nil {
					t.Fatal("deletion accepted")
				}
				if _, err = os.Stat(marker); err != nil {
					t.Fatal("data lost", err)
				}
			}
			_, configErr := os.Stat(configMarker)
			if status == "absent" {
				if !os.IsNotExist(configErr) {
					t.Fatal("configuration data retained", configErr)
				}
			} else if configErr != nil {
				t.Fatal("configuration data lost", configErr)
			}
			if _, err = os.Stat(sibling.Bindings[0].Host); err != nil {
				t.Fatal("other project lost", err)
			}
		})
	}
}
