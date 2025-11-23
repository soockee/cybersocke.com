package config

import (
	"testing"
)

func TestParsePostContract(t *testing.T) {
	// Minimal valid contract YAML
	yamlData := []byte(`version: 2
fields:
  - name: name
    type: string
    required: true
  - name: lead
    type: string
    required: true
  - name: created
    type: date
    required: true
  - name: updated
    type: datetime
    required: true
  - name: published
    type: boolean
    required: true
  - name: tags
    type: array[string]
    required: true
tagFamilies:
  type:
    min: 1
    max: 2
migrations:
  - from: 1
    to: 2
    steps:
      - op: removeField
        field: deprecated_field
`)

	contract, err := ParsePostContract(yamlData)
	if err != nil {
		t.Fatalf("ParsePostContract failed: %v", err)
	}

	if contract.Version != 2 {
		t.Errorf("Expected version 2, got %d", contract.Version)
	}

	if len(contract.Fields) != 6 {
		t.Errorf("Expected 6 fields, got %d", len(contract.Fields))
	}

	if contract.Hash() == "" {
		t.Error("Expected non-empty hash")
	}

	if len(contract.Hash()) != 16 {
		t.Errorf("Expected hash length 16, got %d", len(contract.Hash()))
	}

	// Verify required fields are present
	fieldNames := make(map[string]bool)
	for _, f := range contract.Fields {
		fieldNames[f.Name] = true
	}

	requiredFields := []string{"name", "lead", "created", "updated", "published", "tags"}
	for _, req := range requiredFields {
		if !fieldNames[req] {
			t.Errorf("Missing required field: %s", req)
		}
	}
}

func TestParsePostContractMissingRequiredField(t *testing.T) {
	// Missing 'tags' field
	yamlData := []byte(`version: 1
fields:
  - name: name
    type: string
    required: true
  - name: lead
    type: string
    required: true
  - name: created
    type: date
    required: true
  - name: updated
    type: datetime
    required: true
  - name: published
    type: boolean
    required: true
`)

	_, err := ParsePostContract(yamlData)
	if err == nil {
		t.Fatal("Expected error for missing required field, got nil")
	}

	if err.Error() != "post contract missing required field rules: tags" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestApplyMigrationsRespectsChain(t *testing.T) {
	contract := &PostContract{
		Version: 3,
		Migrations: []Migration{
			{From: 1, To: 2, Steps: []MigrationStep{{Op: "setField", Field: "ran12", Value: "applied"}}},
			{From: 2, To: 3, Steps: []MigrationStep{{Op: "setField", Field: "ran23", Value: "applied"}}},
		},
	}
	front := map[string]any{}
	if _, err := contract.ApplyMigrations(front); err != nil {
		t.Fatalf("ApplyMigrations returned error: %v", err)
	}
	if front["ran12"] != "applied" {
		t.Fatalf("expected ran12 to be applied, got %v", front["ran12"])
	}
	if front["ran23"] != "applied" {
		t.Fatalf("expected ran23 to be applied, got %v", front["ran23"])
	}
	if v := front["schema_version"]; v != "3" {
		t.Fatalf("expected schema_version 3, got %v", v)
	}
}

func TestApplyMigrationsFromClampsBaseline(t *testing.T) {
	contract := &PostContract{
		Version: 3,
		Migrations: []Migration{
			{From: 1, To: 2, Steps: []MigrationStep{{Op: "setField", Field: "ran12", Value: "applied"}}},
			{From: 2, To: 3, Steps: []MigrationStep{{Op: "setField", Field: "ran23", Value: "applied"}}},
		},
	}
	front := map[string]any{}
	if _, err := contract.ApplyMigrationsFrom(front, 2); err != nil {
		t.Fatalf("ApplyMigrationsFrom returned error: %v", err)
	}
	if _, ok := front["ran12"]; ok {
		t.Fatalf("expected ran12 to be skipped when baseline is 2")
	}
	if front["ran23"] != "applied" {
		t.Fatalf("expected ran23 to be applied, got %v", front["ran23"])
	}
	if v := front["schema_version"]; v != "3" {
		t.Fatalf("expected schema_version 3, got %v", v)
	}
}
