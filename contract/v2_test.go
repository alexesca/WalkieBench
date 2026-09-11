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
	for _, bad := range []Manifest{
		{APIVersion: "harnesstalkie/v2", Kind: "Session", Server: "s", Identity: ManifestIdentity{Name: "a"}, Sync: ManifestSync{Since: "tomorrow"}},
		{APIVersion: "harnesstalkie/v2", Kind: "Session", Server: "s", Identity: ManifestIdentity{Name: "a"}, Response: ResponseOptions{Mode: "tiny"}},
		{APIVersion: "harnesstalkie/v2", Kind: "Session", Server: "s", Identity: ManifestIdentity{Name: "a"}, Discover: ManifestDiscovery{Limit: -1}},
	} {
		if err := ValidateManifest(bad); err == nil {
			t.Fatalf("invalid extended manifest accepted: %+v", bad)
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
	withReference, err := BatchOrder([]BatchOperation{{ID: "create", Operation: "CreateServer"}, {ID: "join", Operation: "JoinServer", DependsOn: []string{"create"}, Params: map[string]any{"server_id": "$ref:create.id"}}})
	if err != nil || len(withReference) != 2 || withReference[1].ID != "join" {
		t.Fatalf("valid result reference rejected: order=%+v err=%v", withReference, err)
	}
	for _, bad := range []BatchRequest{
		{Operations: []BatchOperation{{ID: "a", Operation: "A", Params: map[string]any{"id": "$ref:missing.id"}}}},
		{Operations: []BatchOperation{{ID: "a", Operation: "A"}, {ID: "b", Operation: "B", Params: map[string]any{"id": "$ref:a.id"}}}},
		{Operations: []BatchOperation{{ID: "a", Operation: "A", Params: map[string]any{"id": "$ref:a"}}}},
	} {
		if err := ValidateBatch(bad); err == nil {
			t.Fatalf("invalid result reference accepted: %+v", bad)
		}
	}
}

func TestValidateSchemaDocumentRequiresEveryAdvertisedOperation(t *testing.T) {
	doc := SchemaDocument{
		Name: "harnesstalkie/v2", Version: "2.0",
		Schema: map[string]any{"type": "object"},
		Operations: map[string]OperationSchema{
			"CreateServer": {AuthRequired: true, Params: map[string]any{"type": "object"}, Result: map[string]any{"type": "object"}},
		},
	}
	if err := ValidateSchemaDocument(doc, []string{"CreateServer"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSchemaDocument(doc, []string{"CreateServer", "ApplyManifest"}); err == nil {
		t.Fatal("missing advertised operation schema accepted")
	}
	doc.Schema = map[string]any{"type": "array"}
	if err := ValidateSchemaDocument(doc, []string{"CreateServer"}); err == nil {
		t.Fatal("invalid schema root accepted")
	}
}

func TestRequiredPermissionVocabularyIsUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, permission := range RequiredServerPermissions {
		if permission == "" || seen[permission] {
			t.Fatalf("invalid permission vocabulary: %q", permission)
		}
		seen[permission] = true
	}
	if len(seen) < 15 {
		t.Fatalf("permission vocabulary is too broad: %v", RequiredServerPermissions)
	}
}
