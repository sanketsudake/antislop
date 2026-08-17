package flagx_test

import (
	"slices"
	"testing"

	"github.com/sanketsudake/antislop/internal/flagx"
)

func TestListSetReplaces(t *testing.T) {
	list := []string{"one", "two"}
	v := flagx.NewList(&list)
	if got := v.String(); got != "one,two" {
		t.Errorf("String() = %q", got)
	}
	if err := v.Set("three, ,four"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(list, []string{"three", "four"}) {
		t.Errorf("Set replaced badly: %v", list)
	}
	if err := v.Set(""); err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Errorf("empty value should clear the list: %v", list)
	}
}
