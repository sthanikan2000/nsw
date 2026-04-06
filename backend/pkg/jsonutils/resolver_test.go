package jsonutils

import (
	"reflect"
	"testing"
)

func TestResolveTemplate(t *testing.T) {
	// Mock lookup data
	store := map[string]any{
		"consignment.id": "C-123",
		"exporter.name":  "Organic Farms",
		"items.0.code":   "HS-001",
		"items.1.code":   "HS-002",
		"meta.priority":  10,
		"meta.active":    true,
	}

	lookup := func(key string) any {
		return store[key]
	}

	tests := []struct {
		name     string
		template any
		want     any
	}{
		{
			name:     "Simple string replacement",
			template: "consignment.id",
			want:     "C-123",
		},
		{
			name: "Nested map resolution",
			template: map[string]any{
				"header": map[string]any{
					"exporter": "exporter.name",
				},
				"id": "consignment.id",
			},
			want: map[string]any{
				"header": map[string]any{
					"exporter": "Organic Farms",
				},
				"id": "C-123",
			},
		},
		{
			name:     "Array of strings",
			template: []any{"consignment.id", "exporter.name"},
			want:     []any{"C-123", "Organic Farms"},
		},
		{
			name: "Complex nested structure (The OGA example)",
			template: map[string]any{
				"header": map[string]any{
					"priority": "meta.priority",
				},
				"body": map[string]any{
					"items": []any{
						map[string]any{"hs_code": "items.0.code"},
						map[string]any{"hs_code": "items.1.code"},
					},
				},
				"status": "meta.active",
			},
			want: map[string]any{
				"header": map[string]any{
					"priority": 10,
				},
				"body": map[string]any{
					"items": []any{
						map[string]any{"hs_code": "HS-001"},
						map[string]any{"hs_code": "HS-002"},
					},
				},
				"status": true,
			},
		},
		{
			name: "Preserve non-matching strings and types",
			template: map[string]any{
				"literal": "I am a literal string",
				"number":  42,
				"missing": "not.in.store",
			},
			want: map[string]any{
				"literal": "I am a literal string",
				"number":  42,
				"missing": "not.in.store",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveTemplate(tt.template, lookup)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}
