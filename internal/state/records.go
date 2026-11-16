package state

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

func scopeLabel(module string) string {
	if module == "" {
		return "@config"
	}
	return module
}
func (r Record) GuestRecords() (bindings, mappings string) {
	var b, m strings.Builder
	for _, v := range r.Bindings {
		// Lima 2.3+ derives stable 9p tags from both paths. Keep the legacy
		// positional tag as well, so captured records work with older Lima.
		hash := sha256.Sum256([]byte(v.Host + ":" + v.Guest))
		fmt.Fprintf(&b, "%s\t%s\t%s\tlima-%x\n", scopeLabel(v.Module), v.Guest, v.Source, hash[:8])
	}
	for _, v := range r.Mappings {
		fmt.Fprintf(&m, "%s\t%s\t%s\n", scopeLabel(v.Module), v.Guest, v.Data)
	}
	return b.String(), m.String()
}
