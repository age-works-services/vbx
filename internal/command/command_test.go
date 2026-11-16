package command

import "testing"

func TestInvocationOwnsCollections(t *testing.T) {
	flags := map[string]bool{"terminal": true}
	args := []string{"printf", "literal"}
	in := NewInvocation(GeneralOptions{Project: "p"}, flags, args)
	flags["terminal"] = false
	flags["vscode"] = true
	args[1] = "changed"
	if !in.Flags.Bool("terminal") || in.Flags.Bool("vscode") || in.Flags.Len() != 1 || in.Args[1] != "literal" {
		t.Fatal(in)
	}
}
