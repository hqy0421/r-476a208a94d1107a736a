package trdsql

import (
	"bytes"
	"reflect"
	"testing"
)

func TestNewCSVWriter(t *testing.T) {
	outStream := new(bytes.Buffer)
	type args struct {
		writeOpts *WriteOpts
	}
	tests := []struct {
		name string
		args args
		want rune
	}{
		{
			name: "testDefault",
			args: args{
				writeOpts: &WriteOpts{
					OutDelimiter: ",",
					OutStream:    outStream,
				},
			},
			want: ',',
		},
		{
			name: "invalidDelimiter",
			args: args{
				writeOpts: &WriteOpts{
					OutDelimiter: "--",
					OutStream:    outStream,
				},
			},
			want: ',',
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewCSVWriter(tt.args.writeOpts)
			if !reflect.DeepEqual(got.outDelimiter, tt.want) {
				t.Errorf("NewCSVWriter() = %v, want %v", got.outDelimiter, tt.want)
			}
		})
	}
}

func TestCSVWriter_PreWrite(t *testing.T) {
	type args struct {
		columns []string
		types   []string
	}
	tests := []struct {
		name      string
		writeOpts WriteOpts
		args      args
		want      []string
		wantErr   bool
	}{
		{
			name: "empty",
			writeOpts: WriteOpts{
				OutDelimiter: ",",
			},
			args: args{
				columns: []string{},
				types:   []string{},
			},
			want:    []string{},
			wantErr: false,
		},
		{
			name: "emptyHeader",
			writeOpts: WriteOpts{
				OutDelimiter: ",",
				OutHeader:    true,
			},
			args: args{
				columns: []string{"h1", "h2"},
				types:   []string{"text", "text"},
			},
			want:    make([]string, 2),
			wantErr: false,
		},
		{
			name: "noHeader",
			writeOpts: WriteOpts{
				OutDelimiter: ",",
				OutHeader:    false,
			},
			args: args{
				columns: []string{"v1", "v2"},
				types:   []string{"text", "text"},
			},
			want:    make([]string, 2),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewCSVWriter(&tt.writeOpts)
			if err := w.PreWrite(tt.args.columns, tt.args.types); (err != nil) != tt.wantErr {
				t.Errorf("CSVWriter.PreWrite() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCSVWriter_WriteRow_NULLColumns(t *testing.T) {
	tests := []struct {
		name      string
		writeOpts WriteOpts
		columns   []string
		values    []any
		want      string
	}{
		{
			name: "perColumnPlaceholder",
			writeOpts: WriteOpts{
				OutDelimiter:   ",",
				OutNULLColumns: map[string]string{"age": `\N`, "memo": `\N`},
			},
			columns: []string{"id", "name", "age", "memo"},
			values:  []any{"2", `\N`, nil, nil},
			// name is not in the map: NULL becomes an empty field,
			// while a literal `\N` string is written as-is.
			want: "2,\\N,\\N,\\N\n",
		},
		{
			name: "emptyStringStaysEmpty",
			writeOpts: WriteOpts{
				OutDelimiter:   ",",
				OutNULLColumns: map[string]string{"memo": `\N`},
			},
			columns: []string{"id", "memo"},
			values:  []any{"1", ""},
			// empty string is written as an empty field,
			// distinguishable from the NULL placeholder.
			want: "1,\n",
		},
		{
			name: "perColumnPrecedesGlobal",
			writeOpts: WriteOpts{
				OutDelimiter:   ",",
				OutNeedNULL:    true,
				OutNULL:        "(NULL)",
				OutNULLColumns: map[string]string{"age": `\N`},
			},
			columns: []string{"age", "memo"},
			values:  []any{nil, nil},
			want:    "\\N,(NULL)\n",
		},
		{
			name: "globalOnlyWhenNoPerColumn",
			writeOpts: WriteOpts{
				OutDelimiter: ",",
				OutNeedNULL:  true,
				OutNULL:      "(NULL)",
			},
			columns: []string{"age", "memo"},
			values:  []any{nil, nil},
			want:    "(NULL),(NULL)\n",
		},
		{
			name: "nullBecomesEmptyWithoutReplacement",
			writeOpts: WriteOpts{
				OutDelimiter: ",",
			},
			columns: []string{"age", "memo"},
			values:  []any{nil, ""},
			want:    ",\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outStream := new(bytes.Buffer)
			tt.writeOpts.OutStream = outStream
			w := NewCSVWriter(&tt.writeOpts)
			if err := w.PreWrite(tt.columns, nil); err != nil {
				t.Fatal(err)
			}
			if err := w.WriteRow(tt.values, tt.columns); err != nil {
				t.Fatal(err)
			}
			if err := w.PostWrite(); err != nil {
				t.Fatal(err)
			}
			if got := outStream.String(); got != tt.want {
				t.Errorf("output = %q, want %q", got, tt.want)
			}
		})
	}
}
