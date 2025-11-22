package storage

import (
	"testing"

	"github.com/soockee/cybersocke.com/config"
	"github.com/soockee/cybersocke.com/storage/models"
	"github.com/soockee/cybersocke.com/storage/validation"
)

func TestValidateTags_RoleFamilyAccepted(t *testing.T) {
	loadContract(t)
	meta := &models.PostMeta{Tags: []string{"type/article", "role/person", "theme/knowledge"}}
	if err := validation.ValidateTags(meta, config.Contract); err != nil {
		t.Fatalf("expected role family accepted, got error: %v", err)
	}
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
	loadContract(t)
	meta := &models.PostMeta{Tags: []string{"type/article", "role/dev", "role/architect", "role/reviewer", "role/intern", "theme/engineering"}}
	if err := validation.ValidateTags(meta, config.Contract); err == nil {
		t.Fatalf("expected error for too many role tags (4), got nil")
	}
}
