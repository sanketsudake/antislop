package summary_test

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/sanketsudake/antislop"
	"github.com/sanketsudake/antislop/internal/summary"
)

func TestWanted(t *testing.T) {
	if !summary.Wanted([]string{"-summary", "./..."}) || summary.Wanted([]string{"./..."}) || summary.Wanted([]string{"--", "-summary"}) {
		t.Fatal("Wanted misclassified its arguments")
	}
}

// The example module is the golden fixture: summary must count exactly the
// findings listed in example/expected.txt, once each, despite the test and
// non-test package variants both being analyzed.
func TestRunOnExample(t *testing.T) {
	golden, err := os.ReadFile("../../example/expected.txt")
	if err != nil {
		t.Fatal(err)
	}
	want := len(strings.Split(strings.TrimSpace(string(golden)), "\n"))

	var out, errOut bytes.Buffer
	code := summary.Run([]string{"-summary", "-dir=../../example", "./..."}, antislop.Analyzers(), &out, &errOut)
	if code != 3 {
		t.Fatalf("exit code %d, stderr: %s", code, errOut.String())
	}
	text := out.String()
	if !strings.Contains(text, "analyzer") || !strings.Contains(text, "package") {
		t.Fatalf("unexpected output:\n%s", text)
	}
	totalLine := ""
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "total") {
			totalLine = line
		}
	}
	if fields := strings.Fields(totalLine); len(fields) < 2 || fields[1] != strconv.Itoa(want) {
		t.Fatalf("total line %q does not report %d findings\n%s", totalLine, want, text)
	}
}

func TestDisableFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := summary.Run([]string{"-summary", "-dir=../../example", "-noanyparams=false", "./..."}, antislop.Analyzers(), &out, &errOut)
	if code != 3 {
		t.Fatalf("exit code %d, stderr: %s", code, errOut.String())
	}
	if strings.Contains(out.String(), "noanyparams") {
		t.Fatalf("disabled analyzer still reported:\n%s", out.String())
	}
}
