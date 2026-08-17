package callspec_test

import (
	"testing"

	"github.com/sanketsudake/antislop/internal/callspec"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in   string
		want callspec.Spec
	}{
		{"encoding/json.Unmarshal", callspec.Spec{Pkg: "encoding/json", Name: "Unmarshal"}},
		{"gopkg.in/yaml.v3.Unmarshal", callspec.Spec{Pkg: "gopkg.in/yaml.v3", Name: "Unmarshal"}},
		{"(*encoding/json.Decoder).Decode", callspec.Spec{Pkg: "encoding/json", Recv: "Decoder", Name: "Decode"}},
		{"(context.Context).Value", callspec.Spec{Pkg: "context", Recv: "Context", Name: "Value"}},
		{"(*container/list.Element).Value", callspec.Spec{Pkg: "container/list", Recv: "Element", Name: "Value"}},
	}
	for _, c := range cases {
		got, err := callspec.Parse(c.in)
		if err != nil {
			t.Errorf("Parse(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("Parse(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
	for _, bad := range []string{"", "Unmarshal", "encoding/json.", "(encoding/json.Decoder.Decode", "(*json.Decoder)Decode", "a.b.c(", "pkg.Func#1"} {
		if _, err := callspec.Parse(bad); err == nil {
			t.Errorf("Parse(%q) accepted a malformed entry", bad)
		}
	}
}

func TestSplitTarget(t *testing.T) {
	spec, idx, err := callspec.SplitTarget("(*encoding/json.Decoder).Decode#0")
	if err != nil || spec != "(*encoding/json.Decoder).Decode" || idx != 0 {
		t.Errorf("SplitTarget = %q, %d, %v", spec, idx, err)
	}
	for _, bad := range []string{"encoding/json.Unmarshal", "encoding/json.Unmarshal#", "encoding/json.Unmarshal#-1", "encoding/json.Unmarshal#x"} {
		if _, _, err := callspec.SplitTarget(bad); err == nil {
			t.Errorf("SplitTarget(%q) accepted a malformed entry", bad)
		}
	}
}
