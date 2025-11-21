package storage

import "testing"

func TestValidateTags_RoleFamilyAccepted(t *testing.T) {
	meta := &PostMeta{Tags: []string{"type/article", "role/person", "theme/knowledge"}}
	if err := ValidateTags(meta); err != nil {
		t.Fatalf("expected role family accepted, got error: %v", err)
	}
	// Ensure normalization preserved role tag
	found := false
	for _, ttag := range meta.Tags {
		if ttag == "role/person" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("normalized tags missing role/person: %#v", meta.Tags)
	}
}

func TestValidateTags_RoleFamilyCardinality(t *testing.T) {
	meta := &PostMeta{Tags: []string{"type/article", "role/dev", "role/architect", "role/reviewer", "role/intern", "theme/engineering"}}
	if err := ValidateTags(meta); err == nil {
		// 4 role tags should fail (limit 3)
		// If cardinality requirements evolve adjust test accordingly.
		t.Fatalf("expected error for too many role tags (4), got nil")
	}
}
