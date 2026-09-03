package xstrings_test

import (
	"strings"
	"testing"

	"github.com/AeonDigital/Go-Core-xfs/pkg/xstrings"
)

func TestFormatSingleValuesToOutput(t *testing.T) {
	// Definimos a estrutura dos casos de teste (Table-Driven Test)
	tests := []struct {
		name         string
		open         string
		close        string
		sep          string
		values       []string
		inputBuilder *strings.Builder // Permite testar passando nil ou um builder já existente
		want         string
	}{
		{
			name:         "Standard behavior with single element",
			open:         "[",
			close:        "]",
			sep:          ",",
			values:       []string{"Go"},
			inputBuilder: nil,
			want:         "[Go]",
		},
		{
			name:         "Standard behavior with multiple elements",
			open:         "<",
			close:        ">",
			sep:          "-",
			values:       []string{"A", "B", "C"},
			inputBuilder: nil,
			want:         "<A>-<B>-<C>",
		},
		{
			name:         "Empty values slice returns empty string",
			open:         "(",
			close:        ")",
			sep:          "|",
			values:       []string{},
			inputBuilder: nil,
			want:         "",
		},
		{
			name:         "Nil values slice returns empty string safely",
			open:         "(",
			close:        ")",
			sep:          "|",
			values:       nil,
			inputBuilder: nil,
			want:         "",
		},
		{
			name:         "Values containing empty strings are still formatted",
			open:         "\"",
			close:        "\"",
			sep:          ",",
			values:       []string{"first", "", "last"},
			inputBuilder: nil,
			want:         "\"first\",\"\",\"last\"",
		},
		{
			name:   "Reusing an existing strings.Builder (fluent API validation)",
			open:   "{",
			close:  "}",
			sep:    ";",
			values: []string{"x", "y"},
			inputBuilder: func() *strings.Builder {
				sb := &strings.Builder{}
				sb.WriteString("PRE_EXISTING_DATA:")
				return sb
			}(),
			want: "PRE_EXISTING_DATA:{x};{y}",
		},
		{
			name:         "Formatting with empty wrappers and delimiters",
			open:         "",
			close:        "",
			sep:          "",
			values:       []string{"hello", "world"},
			inputBuilder: nil,
			want:         "helloworld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Executa a função pública exposta pelo pacote xstrings
			gotBuilder := xstrings.FormatSingleValuesToOutput(tt.open, tt.close, tt.sep, tt.values, tt.inputBuilder)

			// Valida se o ponteiro retornado não é nil (garante inicialização segura)
			if gotBuilder == nil {
				t.Fatalf("FormatSingleValuesToOutput() returned a nil *strings.Builder pointer")
			}

			// Extrai a string final gerada pelo buffer
			gotString := gotBuilder.String()

			// Valida o resultado final esperado
			if gotString != tt.want {
				t.Errorf("FormatSingleValuesToOutput() = %q, want %q", gotString, tt.want)
			}
		})
	}
}

func TestFormatPairValuesToOutput(t *testing.T) {
	tests := []struct {
		name         string
		openFirst    string
		closeFirst   string
		unionPair    string
		sepPair      string
		openSecond   string
		closeSecond  string
		values       []string
		inputBuilder *strings.Builder
		want         string
	}{
		{
			name:         "Standard complete pairs",
			openFirst:    "[",
			closeFirst:   "]",
			unionPair:    "=",
			sepPair:      ", ",
			openSecond:   "\"",
			closeSecond:  "\"",
			values:       []string{"key1", "val1", "key2", "val2"},
			inputBuilder: nil,
			want:         `[key1]="val1", [key2]="val2"`,
		},
		{
			name:         "Intercept orphan trailing element (odd number of elements)",
			openFirst:    "<",
			closeFirst:   ">",
			unionPair:    ":",
			sepPair:      "|",
			openSecond:   "(",
			closeSecond:  ")",
			values:       []string{"A", "B", "C"}, // 'C' is the orphan element
			inputBuilder: nil,
			want:         "<A>:(B)|<C>",
		},
		{
			name:         "Single orphan element inside slice",
			openFirst:    "@",
			closeFirst:   "@",
			unionPair:    "=>",
			sepPair:      "&&",
			openSecond:   "#",
			closeSecond:  "#",
			values:       []string{"SoleElement"},
			inputBuilder: nil,
			want:         "@SoleElement@",
		},
		{
			name:         "Empty values slice returns empty string safely",
			openFirst:    "[",
			closeFirst:   "]",
			unionPair:    "=",
			sepPair:      ",",
			values:       []string{},
			inputBuilder: nil,
			want:         "",
		},
		{
			name:         "Nil values slice returns empty string safely",
			openFirst:    "[",
			closeFirst:   "]",
			unionPair:    "=",
			sepPair:      ",",
			values:       nil,
			inputBuilder: nil,
			want:         "",
		},
		{
			name:        "Reusing existing strings.Builder buffer",
			openFirst:   "",
			closeFirst:  "",
			unionPair:   "=>",
			sepPair:     ";",
			openSecond:  "",
			closeSecond: "",
			values:      []string{"k1", "v1"},
			inputBuilder: func() *strings.Builder {
				sb := &strings.Builder{}
				sb.WriteString("INIT:")
				return sb
			}(),
			want: "INIT:k1=>v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBuilder := xstrings.FormatPairValuesToOutput(
				tt.openFirst,
				tt.closeFirst,
				tt.unionPair,
				tt.sepPair,
				tt.openSecond,
				tt.closeSecond,
				tt.values,
				tt.inputBuilder,
			)

			if gotBuilder == nil {
				t.Fatalf("FormatPairValuesToOutput() returned a nil *strings.Builder pointer")
			}

			gotString := gotBuilder.String()

			if gotString != tt.want {
				t.Errorf("FormatPairValuesToOutput() = %q, want %q", gotString, tt.want)
			}
		})
	}
}

func TestFormatPairsColonAndArrow(t *testing.T) {
	type inputArgs struct {
		pairs []string
		isNil bool
	}

	t.Run("FormatPairsColon: Table Verification", func(t *testing.T) {
		tests := []struct {
			name  string
			input inputArgs
			want  string
		}{
			{
				name: "Should return empty builder immediately if slice is empty",
				input: inputArgs{
					pairs: []string{},
					isNil: false,
				},
				want: "",
			},
			{
				name: "Should automatically initialize a new builder instance if nil is passed",
				input: inputArgs{
					pairs: []string{"Key", "Value"},
					isNil: true,
				},
				want: "'Key': 'Value'",
			},
			{
				name: "Should successfully format single even pair sequence",
				input: inputArgs{
					pairs: []string{"FilePath", "/app/config.json"},
					isNil: false,
				},
				want: "'FilePath': '/app/config.json'",
			},
			{
				name: "Should successfully join multiple pairs with colon and pipes layout",
				input: inputArgs{
					pairs: []string{"Prefix", "APP", "Separator", "_", "Env", "production"},
					isNil: false,
				},
				want: "'Prefix': 'APP' | 'Separator': '_' | 'Env': 'production'",
			},
			{
				name: "Should safely intercept trailing odd orphan element without causing a crash",
				input: inputArgs{
					pairs: []string{"Prefix", "APP", "OrphanKey"},
					isNil: false,
				},
				want: "'Prefix': 'APP' | 'OrphanKey'",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var sb *strings.Builder
				if !tt.input.isNil {
					sb = &strings.Builder{}
				}

				got := xstrings.FormatPairsColon(tt.input.pairs, sb)

				if got == nil {
					t.Fatal("Expected returned strings.Builder pointer to be non-nil")
				}
				if got.String() != tt.want {
					t.Errorf("FormatPairsColon() = %q, want %q", got.String(), tt.want)
				}
			})
		}
	})

	t.Run("FormatPairsArrow: Table Verification", func(t *testing.T) {
		tests := []struct {
			name  string
			input inputArgs
			want  string
		}{
			{
				name: "Should return empty builder immediately if slice is empty",
				input: inputArgs{
					pairs: []string{},
					isNil: false,
				},
				want: "",
			},
			{
				name: "Should automatically initialize a new builder instance if nil is passed",
				input: inputArgs{
					pairs: []string{"Key", "Value"},
					isNil: true,
				},
				want: "Key => Value",
			},
			{
				name: "Should successfully format single arrow structure leaving value unquoted",
				input: inputArgs{
					pairs: []string{"Target", "12345"},
					isNil: false,
				},
				want: "Target => 12345",
			},
			{
				name: "Should successfully join multiple pairs with arrows and pipes layout",
				input: inputArgs{
					pairs: []string{"Code", "E4003", "Status", "Failure", "Retries", "3"},
					isNil: false,
				},
				want: "Code => E4003 | Status => Failure | Retries => 3",
			},
			{
				name: "Should safely intercept trailing odd orphan element with clean encapsulation bounds",
				input: inputArgs{
					pairs: []string{"Code", "E4003", "MissingValueKey"},
					isNil: false,
				},
				want: "Code => E4003 | MissingValueKey",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var sb *strings.Builder
				if !tt.input.isNil {
					sb = &strings.Builder{}
				}

				got := xstrings.FormatPairsArrow(tt.input.pairs, sb)

				if got == nil {
					t.Fatal("Expected returned strings.Builder pointer to be non-nil")
				}
				if got.String() != tt.want {
					t.Errorf("FormatPairsArrow() = %q, want %q", got.String(), tt.want)
				}
			})
		}
	})
}

type MockUser struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type MockPrivate struct {
	secretData string
	tokenID    int
}

type MockMixed struct {
	PublicField  string `json:"public"`
	privateField string
}

type MockUnserializable struct {
	Data chan int `json:"data"`
}

func TestParseStructToString(t *testing.T) {
	tests := []struct {
		name string
		item any
		want string
	}{
		{
			name: "Nil input value",
			item: nil,
			want: "<nil>",
		},
		{
			name: "Non-struct primitive value (fallback)",
			item: "just a string",
			want: "just a string",
		},
		{
			name: "Standard struct with public fields",
			item: MockUser{ID: 10, Name: "Alice"},
			want: `MockUser{"id":10,"name":"Alice"}`,
		},
		{
			name: "Pointer to a struct with public fields",
			item: &MockUser{ID: 20, Name: "Bob"},
			want: `MockUser{"id":20,"name":"Bob"}`,
		},
		{
			name: "Struct containing only private fields",
			item: MockPrivate{secretData: "hidden", tokenID: 123},
			want: "MockPrivate{}",
		},
		{
			name: "Struct with mixed fields (only public are serialized)",
			item: MockMixed{PublicField: "visible", privateField: "hidden"},
			want: `MockMixed{"public":"visible"}`,
		},
		{
			name: "Unserializable field triggers JSON marshal error fallback",
			item: MockUnserializable{Data: make(chan int)},
			want: "MockUnserializable{error:json: unsupported type: chan int}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := xstrings.ParseStructToString(tt.item)
			if got != tt.want {
				t.Errorf("ParseStructToString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConvertAllToStrings(t *testing.T) {
	type TestRole struct {
		Type string `json:"type"`
	}
	type TestUser struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	tests := []struct {
		name string
		args []any
		want []string
	}{
		{
			name: "Primitives and nil arguments",
			args: []any{"Go", 2026, true, nil},
			want: []string{"Go", "2026", "true", ""},
		},
		{
			name: "Individual struct and pointer to struct",
			args: []any{TestRole{Type: "Admin"}, &TestUser{ID: 1, Name: "Alice"}},
			want: []string{`TestRole{"type":"Admin"}`, `TestUser{"id":1,"name":"Alice"}`},
		},
		{
			name: "Standard numeric slice (not uint8)",
			args: []any{[]int{10, 20}},
			want: []string{"[]int{10,20}"},
		},
		{
			name: "Byte slice (uint8) bypasses iteration",
			args: []any{[]byte("Hello")},
			want: []string{"[72 101 108 108 111]"},
		},
		{
			name: "Slice of structs preserves collection type and encapsulates JSON",
			args: []any{[]TestRole{{Type: "Admin"}, {Type: "User"}}},
			want: []string{`[]xstrings_test.TestRole{{"type":"Admin"},{"type":"User"}}`},
		},
		{
			name: "Slice of pointers to structs encapsulates JSON correctly",
			args: []any{[]*TestRole{{Type: "Guest"}}},
			want: []string{`[]*xstrings_test.TestRole{{"type":"Guest"}}`},
		},
		{
			name: "Multidimensional slice (matrix) applies recursive formatting",
			args: []any{[][]int{{1, 2}, {3}}},
			want: []string{"[][]int{[]int{1,2},[]int{3}}"},
		},
		{
			name: "Complex mixed arguments pipeline",
			args: []any{
				"System",
				TestRole{Type: "Root"},
				[]int{99},
			},
			want: []string{
				"System",
				`TestRole{"type":"Root"}`,
				"[]int{99}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := xstrings.ConvertAllToStrings(tt.args...)

			if len(got) != len(tt.want) {
				t.Fatalf("ConvertAllToStrings() length = %d, want %d", len(got), len(tt.want))
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ConvertAllToStrings() at index %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestBuildErrorInfo(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "Zero arguments provided",
			args: []string{},
			want: "",
		},
		{
			name: "Nil argument slice provided",
			args: nil,
			want: "",
		},
		{
			name: "All arguments are empty strings",
			args: []string{"", "", "", "", ""},
			want: "",
		},
		{
			name: "Only Field provided (one parameter)",
			args: []string{"configPath"},
			want: "Field => configPath",
		},
		{
			name: "Field and Given provided (two parameters)",
			args: []string{"configPath", "raw_input_data"},
			want: "Field => configPath | Given => raw_input_data",
		},
		{
			name: "Field, Given and Expected provided (three parameters)",
			args: []string{"size", "XYZ", "B, K, KB, M, MB"},
			want: "Field => size | Given => XYZ | Expected => B, K, KB, M, MB",
		},
		{
			name: "Field, Given, Expected, and Extracted provided (four parameters)",
			args: []string{"version", "1.5.5mb", "float64", "1.5.5"},
			want: "Field => version | Given => 1.5.5mb | Expected => float64 | Extracted => 1.5.5",
		},
		{
			name: "All five standard positional parameters provided",
			args: []string{"payload", "bad_json", "JSON", "nested_node", "malformed syntax"},
			want: "Field => payload | Given => bad_json | Expected => JSON | Extracted => nested_node | Reason => malformed syntax",
		},
		{
			name: "Intercalated empty strings are safely pruned and skipped",
			args: []string{"field_ok", "", "expected_ok", "", "reason_ok"},
			want: "Field => field_ok | Expected => expected_ok | Reason => reason_ok",
		},
		{
			name: "Arguments exceeding the managed limits are safely ignored",
			args: []string{"val1", "val2", "val3", "val4", "val5", "unexpected_sixth_arg", "seventh_arg"},
			want: "Field => val1 | Given => val2 | Expected => val3 | Extracted => val4 | Reason => val5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := xstrings.BuildErrorInfo(tt.args...)
			if got != tt.want {
				t.Errorf("BuildErrorInfo() = %q, want %q", got, tt.want)
			}
		})
	}
}
