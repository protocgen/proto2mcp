package extract

import (
	"testing"

	validatepb "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
)

func TestExtractInt32Constraints(t *testing.T) {
	tests := []struct {
		name       string
		rules      *validatepb.Int32Rules
		wantResult map[string]any
		wantDescs  []string
	}{
		{
			name:       "nil rules",
			rules:      nil,
			wantResult: map[string]any{},
		},
		{
			name: "gte and lte",
			rules: func() *validatepb.Int32Rules {
				r := &validatepb.Int32Rules{}
				r.SetGte(1)
				r.SetLte(100)
				return r
			}(),
			wantResult: map[string]any{"minimum": int32(1), "maximum": int32(100)},
		},
		{
			name: "gt and lt (exclusive)",
			rules: func() *validatepb.Int32Rules {
				r := &validatepb.Int32Rules{}
				r.SetGt(0)
				r.SetLt(50)
				return r
			}(),
			wantResult: map[string]any{"exclusiveMinimum": int32(0), "exclusiveMaximum": int32(50)},
		},
		{
			name: "const",
			rules: func() *validatepb.Int32Rules {
				r := &validatepb.Int32Rules{}
				r.SetConst(42)
				return r
			}(),
			wantResult: map[string]any{"const": int32(42)},
		},
		{
			name: "in list",
			rules: &validatepb.Int32Rules{
				In: []int32{1, 2, 3},
			},
			wantResult: map[string]any{"enum": []int32{1, 2, 3}},
		},
		{
			name: "notIn list",
			rules: &validatepb.Int32Rules{
				NotIn: []int32{0, -1},
			},
			wantResult: map[string]any{},
			wantDescs:  []string{"Must not be one of: [0 -1]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := make(map[string]any)
			var descs []string
			extractInt32Constraints(tt.rules, result, &descs)

			for k, v := range tt.wantResult {
				got, ok := result[k]
				if !ok {
					t.Errorf("missing key %q", k)
					continue
				}
				if fmt_val(got) != fmt_val(v) {
					t.Errorf("key %q: got %v, want %v", k, got, v)
				}
			}

			if len(descs) != len(tt.wantDescs) {
				t.Errorf("descriptions: got %v, want %v", descs, tt.wantDescs)
			} else {
				for i := range descs {
					if descs[i] != tt.wantDescs[i] {
						t.Errorf("description[%d]: got %q, want %q", i, descs[i], tt.wantDescs[i])
					}
				}
			}
		})
	}
}

func TestExtractInt64Constraints(t *testing.T) {
	tests := []struct {
		name      string
		rules     *validatepb.Int64Rules
		wantDescs []string
	}{
		{
			name:  "nil rules",
			rules: nil,
		},
		{
			name: "gte and lte produce prose",
			rules: func() *validatepb.Int64Rules {
				r := &validatepb.Int64Rules{}
				r.SetGte(100)
				r.SetLte(1000)
				return r
			}(),
			wantDescs: []string{"Value must be >= 100", "Value must be <= 1000"},
		},
		{
			name: "gt produces prose",
			rules: func() *validatepb.Int64Rules {
				r := &validatepb.Int64Rules{}
				r.SetGt(0)
				return r
			}(),
			wantDescs: []string{"Value must be > 0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := make(map[string]any)
			var descs []string
			extractInt64Constraints(tt.rules, result, &descs)

			// int64 should never produce JSON Schema keywords
			if len(result) > 0 {
				t.Errorf("int64 constraints should produce prose only, got result keys: %v", result)
			}

			if len(descs) != len(tt.wantDescs) {
				t.Errorf("descriptions: got %v, want %v", descs, tt.wantDescs)
			} else {
				for i := range descs {
					if descs[i] != tt.wantDescs[i] {
						t.Errorf("description[%d]: got %q, want %q", i, descs[i], tt.wantDescs[i])
					}
				}
			}
		})
	}
}

func TestExtractFloatConstraints(t *testing.T) {
	tests := []struct {
		name       string
		rules      *validatepb.FloatRules
		wantResult map[string]any
	}{
		{
			name:       "nil rules",
			rules:      nil,
			wantResult: map[string]any{},
		},
		{
			name: "gte and lte",
			rules: func() *validatepb.FloatRules {
				r := &validatepb.FloatRules{}
				r.SetGte(0.0)
				r.SetLte(1.0)
				return r
			}(),
			wantResult: map[string]any{"minimum": float32(0.0), "maximum": float32(1.0)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := make(map[string]any)
			var descs []string
			extractFloatConstraints(tt.rules, result, &descs)

			for k, v := range tt.wantResult {
				got, ok := result[k]
				if !ok {
					t.Errorf("missing key %q", k)
					continue
				}
				if fmt_val(got) != fmt_val(v) {
					t.Errorf("key %q: got %v, want %v", k, got, v)
				}
			}
		})
	}
}

func TestExtractUint32Constraints(t *testing.T) {
	r := &validatepb.UInt32Rules{}
	r.SetGte(1)
	r.SetLte(255)

	result := make(map[string]any)
	var descs []string
	extractUint32Constraints(r, result, &descs)

	if result["minimum"] != uint32(1) {
		t.Errorf("expected minimum=1, got %v", result["minimum"])
	}
	if result["maximum"] != uint32(255) {
		t.Errorf("expected maximum=255, got %v", result["maximum"])
	}
}

func TestExtractUint64Constraints(t *testing.T) {
	r := &validatepb.UInt64Rules{}
	r.SetGte(100)

	result := make(map[string]any)
	var descs []string
	extractUint64Constraints(r, result, &descs)

	// uint64 should only produce prose
	if len(result) > 0 {
		t.Errorf("uint64 constraints should produce prose only, got %v", result)
	}
	if len(descs) != 1 || descs[0] != "Value must be >= 100" {
		t.Errorf("expected prose description, got %v", descs)
	}
}

// fmt_val converts a value to a comparable string for test assertions.
func fmt_val(v any) string {
	return fmt_sprint(v)
}

func fmt_sprint(v any) string {
	switch x := v.(type) {
	case int32:
		return fmt_int(int64(x))
	case uint32:
		return fmt_int(int64(x))
	case float32:
		return fmt_float(float64(x))
	case float64:
		return fmt_float(x)
	default:
		return ""
	}
}

func fmt_int(v int64) string {
	return string(rune(v + '0'))
}

func fmt_float(v float64) string {
	if v == 0 {
		return "0"
	}
	return "f"
}
