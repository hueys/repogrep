package cli

import (
	"reflect"
	"testing"
)

func TestSplitSearchArgs(t *testing.T) {
	cases := []struct {
		name         string
		args         []string
		wantQuery    []string
		wantFlagArgs []string
	}{
		{
			name:         "query only",
			args:         []string{"retrieval", "augmented", "generation"},
			wantQuery:    []string{"retrieval", "augmented", "generation"},
			wantFlagArgs: []string{},
		},
		{
			name:         "single word query",
			args:         []string{"rag"},
			wantQuery:    []string{"rag"},
			wantFlagArgs: []string{},
		},
		{
			name:         "query followed by flags",
			args:         []string{"vector", "database", "--lang", "Go", "--limit", "10"},
			wantQuery:    []string{"vector", "database"},
			wantFlagArgs: []string{"--lang", "Go", "--limit", "10"},
		},
		{
			name:         "query followed by short flag",
			args:         []string{"rag", "-v"},
			wantQuery:    []string{"rag"},
			wantFlagArgs: []string{"-v"},
		},
		{
			name:         "flags only, no query",
			args:         []string{"--lang", "Go"},
			wantQuery:    nil,
			wantFlagArgs: []string{"--lang", "Go"},
		},
		{
			name:         "empty args",
			args:         []string{},
			wantQuery:    nil,
			wantFlagArgs: []string{},
		},
		{
			name:         "hyphenated query word looks like a flag and is not part of the query",
			args:         []string{"-vector-db", "--lang", "Go"},
			wantQuery:    nil,
			wantFlagArgs: []string{"-vector-db", "--lang", "Go"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotQuery, gotFlagArgs := splitSearchArgs(c.args)
			if !reflect.DeepEqual(gotQuery, c.wantQuery) {
				t.Errorf("query = %#v, want %#v", gotQuery, c.wantQuery)
			}
			if !reflect.DeepEqual(gotFlagArgs, c.wantFlagArgs) {
				t.Errorf("flagArgs = %#v, want %#v", gotFlagArgs, c.wantFlagArgs)
			}
		})
	}
}
