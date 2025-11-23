package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

var postContractPath = "schema/post_contract.yaml"

// FieldRule describes a single frontmatter field constraint.
type FieldRule struct {
	Name           string   `yaml:"name" json:"name"`
	Type           string   `yaml:"type" json:"type"`
	Required       bool     `yaml:"required" json:"required"`
	MaxLength      int      `yaml:"maxLength,omitempty" json:"maxLength,omitempty"`
	SingleLine     bool     `yaml:"singleLine,omitempty" json:"singleLine,omitempty"`
	FormatsAllowed []string `yaml:"formatsAllowed,omitempty" json:"formatsAllowed,omitempty"`
	Format         string   `yaml:"format,omitempty" json:"format,omitempty"`
	Description    string   `yaml:"description,omitempty" json:"description,omitempty"`
}

// TagFamilyRule constrains tag family cardinality.
type TagFamilyRule struct {
	Min         int    `yaml:"min" json:"min"`
	Max         int    `yaml:"max" json:"max"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// PostContract aggregates all field and tag family rules.
type PostContract struct {
	Version     int                      `yaml:"version" json:"version"`
	Fields      []FieldRule              `yaml:"fields" json:"fields"`
	TagFamilies map[string]TagFamilyRule `yaml:"tagFamilies" json:"tagFamilies"`
	Migrations  []Migration              `yaml:"migrations" json:"migrations"`
	hash        string                   // computed content hash for cache busting
}

// Migration defines a set of steps to transform frontmatter from one version to another.
type Migration struct {
	From  int             `yaml:"from" json:"from"`
	To    int             `yaml:"to" json:"to"`
	Steps []MigrationStep `yaml:"steps" json:"steps"`
}

// MigrationStep describes a single atomic change.
type MigrationStep struct {
	Op      string `yaml:"op" json:"op"` // removeField | renameField | setField
	Field   string `yaml:"field" json:"field"`
	ToField string `yaml:"toField,omitempty" json:"toField,omitempty"`
	Value   string `yaml:"value,omitempty" json:"value,omitempty"`
}

// Contract holds the loaded post contract (singleton for process lifetime).
var Contract *PostContract

// ParsePostContract parses raw YAML bytes into a PostContract and validates required fields.
// Returns the contract with computed hash (short SHA256 of input bytes).
func ParsePostContract(data []byte) (*PostContract, error) {
	c := &PostContract{}
	if err := yaml.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("unmarshal post contract: %w", err)
	}
	h := sha256.Sum256(data)
	c.hash = hex.EncodeToString(h[:8]) // short hash
	// Basic sanity: required core fields present.
	required := map[string]struct{}{"name": {}, "lead": {}, "created": {}, "updated": {}, "published": {}, "tags": {}}
	for _, f := range c.Fields {
		delete(required, f.Name)
	}
	if len(required) > 0 {
		return nil, fmt.Errorf("post contract missing required field rules: %s", keys(required))
	}
	return c, nil
}

// LoadPostContract parses the local YAML contract file into memory and sets the global Contract.
func LoadPostContract() (*PostContract, error) {
	postContractRaw, err := os.ReadFile(postContractPath)
	if err != nil {
		return nil, fmt.Errorf("read post contract: %w", err)
	}
	c, err := ParsePostContract(postContractRaw)
	if err != nil {
		return nil, err
	}
	Contract = c
	return c, nil
}

// SchemaStore defines minimal interface for reading/writing schema from remote storage.
type SchemaStore interface {
	GetSchema(ctx context.Context) ([]byte, error)
}

// LoadPostContractRemote attempts to load the contract from remote storage (schema/post_contract.yaml).
// Falls back to local filesystem (LoadPostContract) on any remote fetch error (auth, network, not found).
// Sets the global Contract variable on success from either source.
// Returns the loaded contract and a source indicator ("remote" or "local").
func LoadPostContractRemote(ctx context.Context, store SchemaStore) (*PostContract, string, error) {
	if store != nil {
		remoteData, err := store.GetSchema(ctx)
		if err == nil {
			c, parseErr := ParsePostContract(remoteData)
			if parseErr == nil {
				Contract = c
				return c, "remote", nil
			}
			// Remote fetch succeeded but parse failed - log and fall through to local
		}
		// Remote fetch failed (not found, auth, network) - fall through to local
	}
	// Fallback to local
	c, err := LoadPostContract()
	if err != nil {
		return nil, "", err
	}
	return c, "local", nil
}

func keys(m map[string]struct{}) string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return strings.Join(out, ",")
}

// EditorJSON produces a compact JSON string for injection into templates.
func (c *PostContract) EditorJSON() (string, error) {
	if c == nil {
		return "{}", nil
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return "{}", err
	}
	return string(raw), nil
}

// Hash returns a short hash identifying the loaded contract version.
func (c *PostContract) Hash() string {
	if c == nil {
		return ""
	}
	return c.hash
}

// ApplyMigrations modifies a generic frontmatter map in-place to reach targetVersion.
// It assumes currentVersion is either provided in the map as schema_version or defaults to 1.
func (c *PostContract) ApplyMigrations(front map[string]any) (int, error) {
	if c == nil {
		return 0, fmt.Errorf("contract not loaded")
	}
	cur := extractSchemaVersion(front)
	return c.runMigrations(front, cur, c.Version)
}

// ApplyMigrationsFrom enforces a minimum starting version before applying migrations.
// If the detected schema_version is lower than minVersion (or missing), it is clamped
// to minVersion so only newer migration steps are executed.
func (c *PostContract) ApplyMigrationsFrom(front map[string]any, minVersion int) (int, error) {
	if c == nil {
		return 0, fmt.Errorf("contract not loaded")
	}
	cur := extractSchemaVersion(front)
	if minVersion > 0 && cur < minVersion {
		cur = minVersion
		front["schema_version"] = fmt.Sprintf("%d", cur)
	}
	return c.runMigrations(front, cur, c.Version)
}

func (c *PostContract) runMigrations(front map[string]any, cur, target int) (int, error) {
	if cur >= target {
		return cur, nil
	}
	stepApplied := false
	for cur < target {
		mig, ok := findMigration(c.Migrations, cur)
		if !ok {
			return cur, fmt.Errorf("no migration path from %d to %d", cur, target)
		}
		for _, s := range mig.Steps {
			switch s.Op {
			case "removeField":
				delete(front, s.Field)
			case "renameField":
				if val, present := front[s.Field]; present {
					front[s.ToField] = val
					delete(front, s.Field)
				}
			case "setField":
				front[s.Field] = s.Value
			default:
				return cur, fmt.Errorf("unknown migration op: %s", s.Op)
			}
		}
		cur = mig.To
		stepApplied = true
		front["schema_version"] = fmt.Sprintf("%d", cur)
	}
	if !stepApplied {
		return cur, nil
	}
	return cur, nil
}

func extractSchemaVersion(front map[string]any) int {
	cur := 1
	if v, ok := front["schema_version"].(string); ok && v != "" {
		for _, ch := range v {
			if ch < '0' || ch > '9' {
				return 1
			}
		}
		if n := parseInt(v); n > 0 {
			cur = n
		}
	}
	return cur
}

func findMigration(migs []Migration, from int) (Migration, bool) {
	for _, m := range migs {
		if m.From == from {
			return m, true
		}
	}
	return Migration{}, false
}

func parseInt(s string) int {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return n
		}
		n = n*10 + int(ch-'0')
	}
	return n
}
