package contract

import "testing"

func TestValidateManifest(t *testing.T) {
	valid := Manifest{APIVersion: "harnesstalkie/v2", Kind: "Session", Server: "server-1", Identity: ManifestIdentity{Name: "agent"}, Membership: ManifestMembership{Join: "if-allowed"}}
	if err := ValidateManifest(valid); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []Manifest{{Kind: "Session", Server: "s", Identity: ManifestIdentity{Name: "a"}}, {APIVersion: "harnesstalkie/v2", Kind: "Wrong", Server: "s", Identity: ManifestIdentity{Name: "a"}}, {APIVersion: "harnesstalkie/v2", Kind: "Session", Identity: ManifestIdentity{Name: "a"}}, {APIVersion: "harnesstalkie/v2", Kind: "Session", Server: "s"}, {APIVersion: "harnesstalkie/v2", Kind: "Session", Server: "s", Identity: ManifestIdentity{Name: "a"}, Membership: ManifestMembership{Join: "maybe"}}} {
		if err := ValidateManifest(bad); err == nil {
			t.Fatalf("invalid manifest accepted: %+v", bad)
		}
	}
}

func TestBatchOrderRejectsCyclesAndOrdersDependencies(t *testing.T) {
	ordered, err := BatchOrder([]BatchOperation{{ID: "send", Operation: "SendDM", DependsOn: []string{"connect"}}, {ID: "connect", Operation: "ConnectTo"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(ordered) != 2 || ordered[0].ID != "connect" || ordered[1].ID != "send" {
		t.Fatalf("order = %#v", ordered)
	}
	if _, err := BatchOrder([]BatchOperation{{ID: "a", Operation: "A", DependsOn: []string{"b"}}, {ID: "b", Operation: "B", DependsOn: []string{"a"}}}); err == nil {
		t.Fatal("cycle accepted")
	}
	if _, err := BatchOrder([]BatchOperation{{ID: "a", Operation: "A", DependsOn: []string{"missing"}}}); err == nil {
		t.Fatal("missing dependency accepted")
	}
}
