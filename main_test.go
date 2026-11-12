package main

import (
	"flag"
	"reflect"
	"testing"
)

func TestNormalizeToolsFlagArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantToolsOn bool
		wantTools   []string
		wantInter   bool
		wantPre     string
		wantPos     []string
	}{
		{
			name:        "tool list followed by interactive flag",
			args:        []string{"-t", "web_search_exa", "-i"},
			wantToolsOn: true,
			wantTools:   []string{"web_search_exa"},
			wantInter:   true,
		},
		{
			name:        "long tools flag followed by interactive flag and prompt",
			args:        []string{"--tools", "web_search_exa,read_file", "-i", "search docs"},
			wantToolsOn: true,
			wantTools:   []string{"web_search_exa", "read_file"},
			wantInter:   true,
			wantPos:     []string{"search docs"},
		},
		{
			name:        "tools flag without tool list keeps prompt",
			args:        []string{"-t", "What files are here?"},
			wantToolsOn: true,
			wantPos:     []string{"What files are here?"},
		},
		{
			name:        "tools flag followed directly by interactive flag",
			args:        []string{"-t", "-i"},
			wantToolsOn: true,
			wantInter:   true,
		},
		{
			name:        "value flag consuming -t as value",
			args:        []string{"-preprompt", "-t", "-i", "hello"},
			wantPre:     "-t",
			wantInter:   true,
			wantPos:     []string{"hello"},
		},
		{
			name:        "double dash stops flag parsing",
			args:        []string{"-t", "--", "web_search_exa"},
			wantToolsOn: true,
			wantPos:     []string{"web_search_exa"},
		},
		{
			name:        "multiple -t flags with tool names and -i",
			args:        []string{"-t", "web_search_exa", "-t", "read_file", "-i"},
			wantToolsOn: true,
			wantTools:   []string{"web_search_exa", "read_file"},
			wantInter:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := flag.NewFlagSet("tgpt", flag.ContinueOnError)
			var toolsFlag toolsFlagValue
			fs.Var(&toolsFlag, "t", "tools")
			fs.Var(&toolsFlag, "tools", "tools")
			isInteractive := fs.Bool("i", false, "interactive")
			preprompt := fs.String("preprompt", "", "preprompt")

			norm := normalizeToolsFlagArgs(fs, tc.args)
			if err := fs.Parse(norm); err != nil {
				t.Fatalf("Parse(%v) failed: %v", norm, err)
			}

			if toolsFlag.enabled != tc.wantToolsOn {
				t.Errorf("toolsFlag.enabled = %v, want %v", toolsFlag.enabled, tc.wantToolsOn)
			}
			if !reflect.DeepEqual(toolsFlag.toolNames, tc.wantTools) {
				t.Errorf("toolsFlag.toolNames = %v, want %v", toolsFlag.toolNames, tc.wantTools)
			}
			if *isInteractive != tc.wantInter {
				t.Errorf("isInteractive = %v, want %v", *isInteractive, tc.wantInter)
			}
			if *preprompt != tc.wantPre {
				t.Errorf("preprompt = %q, want %q", *preprompt, tc.wantPre)
			}
			gotPos := fs.Args()
			if len(gotPos) == 0 && len(tc.wantPos) == 0 {
				return
			}
			if !reflect.DeepEqual(gotPos, tc.wantPos) {
				t.Errorf("Args() = %v, want %v", gotPos, tc.wantPos)
			}
		})
	}
}

