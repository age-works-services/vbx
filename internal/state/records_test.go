package state

import "testing"

func TestGuestRecordsStableMountTag(t *testing.T) {
	// Lima 2.3's tag is the first eight SHA-256 bytes of host + ":" + guest.
	r := Record{Bindings: []Binding{{Module: "dev-tools", Host: "/host/data/dev-tools", Guest: "/mnt/vbx-data/dev-tools", Source: "mount2"}}}
	bindings, _ := r.GuestRecords()
	if want := "dev-tools\t/mnt/vbx-data/dev-tools\tmount2\tlima-34ff09956ef2385f\n"; bindings != want {
		t.Fatalf("bindings = %q, want %q", bindings, want)
	}
	r.Bindings[0].Guest = "/mnt/vbx-data/other"
	bindings, _ = r.GuestRecords()
	if want := "dev-tools\t/mnt/vbx-data/other\tmount2\tlima-c0e08dd593f9922b\n"; bindings != want {
		t.Fatalf("bindings = %q, want %q", bindings, want)
	}
}
