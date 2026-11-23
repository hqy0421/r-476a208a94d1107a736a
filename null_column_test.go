package trdsql

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// supplierCSV uses `\N` for "unknown" and an empty string for
// "manually confirmed empty". The name column is not covered by the
// placeholder convention, so a literal `\N` there is real data.
const supplierCSV = "id,name,age,memo\n" +
	"1,alice,30,\n" +
	"2,\\N,\\N,\\N\n" +
	"3,bob,25,tokyo\n"

var supplierNULLColumns = map[string]string{"age": `\N`, "memo": `\N`}

// execSupplierQuery imports the supplier CSV with per-column NULL
// decoding and runs the query, returning the CSV output.
func execSupplierQuery(t *testing.T, fileName string, query string, writeOpts ...WriteOpt) string {
	t.Helper()
	outStream := new(bytes.Buffer)
	importer := NewImporter(
		InFormat(CSV),
		InHeader(true),
		InNULLColumns(supplierNULLColumns),
	)
	exporter := NewExporter(NewWriter(append([]WriteOpt{
		OutFormat(CSV),
		OutStream(outStream),
	}, writeOpts...)...))
	trd := NewTRDSQL(importer, exporter)
	// #nosec G201 -- test query built from a fixed set of strings.
	if err := trd.Exec(fmt.Sprintf(query, fileName)); err != nil {
		t.Fatalf("Exec(%q) error: %v", query, err)
	}
	return outStream.String()
}

func TestTRDSQL_NULLColumnConditions(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "supplier.csv")
	if err := os.WriteFile(fileName, []byte(supplierCSV), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "placeholderIsNULL",
			query: "SELECT id FROM %s WHERE memo IS NULL",
			want:  "2\n",
		},
		{
			name:  "emptyStringIsNotNULL",
			query: "SELECT id FROM %s WHERE memo = ''",
			want:  "1\n",
		},
		{
			name:  "placeholderNotDecodedOutsideColumns",
			query: `SELECT id FROM %s WHERE name = '\N'`,
			want:  "2\n",
		},
		{
			name:  "nullCount",
			query: "SELECT COUNT(*) FROM %s WHERE age IS NULL",
			want:  "1\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := execSupplierQuery(t, fileName, tt.query); got != tt.want {
				t.Errorf("output = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTRDSQL_NULLColumnRoundTrip(t *testing.T) {
	dir := t.TempDir()
	fileName := filepath.Join(dir, "supplier.csv")
	if err := os.WriteFile(fileName, []byte(supplierCSV), 0o600); err != nil {
		t.Fatal(err)
	}

	// Re-export: NULL is written back as the per-column placeholder,
	// empty strings stay empty fields. The header is kept so that the
	// next reader can apply the per-column convention by name.
	exported := execSupplierQuery(t, fileName,
		"SELECT * FROM %s ORDER BY id",
		OutHeader(true),
		OutNULLColumns(supplierNULLColumns),
	)
	if exported != supplierCSV {
		t.Errorf("re-exported CSV = %q, want %q", exported, supplierCSV)
	}

	// Another reader imports the exported file with the same per-column
	// convention: NULL and empty string are still distinguishable.
	reexportName := filepath.Join(dir, "reexport.csv")
	if err := os.WriteFile(reexportName, []byte(exported), 0o600); err != nil {
		t.Fatal(err)
	}
	got := execSupplierQuery(t, reexportName,
		"SELECT COUNT(*) FROM %s WHERE memo IS NULL OR age IS NULL",
	)
	if got != "1\n" {
		t.Errorf("NULL count after re-import = %q, want %q", got, "1\n")
	}
	got = execSupplierQuery(t, reexportName,
		"SELECT COUNT(*) FROM %s WHERE memo = ''",
	)
	if got != "1\n" {
		t.Errorf("empty string count after re-import = %q, want %q", got, "1\n")
	}
}
