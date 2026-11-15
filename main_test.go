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

func TestSessionFlags(t *testing.T) {
	fs := flag.NewFlagSet("tgpt", flag.ContinueOnError)
	sessionName := fs.String("session", "", "session name")
	resumeSession := fs.Bool("r", false, "resume session")
	fs.BoolVar(resumeSession, "resume", false, "resume session")
	listSessions := fs.Bool("list-sessions", false, "list sessions")
	deleteSession := fs.String("delete-session", "", "delete session")

	if err := fs.Parse([]string{"--session", "my-chat", "-r"}); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if *sessionName != "my-chat" {
		t.Errorf("sessionName = %q, want my-chat", *sessionName)
	}
	if !*resumeSession {
		t.Errorf("resumeSession = false, want true")
	}
	if *listSessions {
		t.Errorf("listSessions = true, want false")
	}
	if *deleteSession != "" {
		t.Errorf("deleteSession = %q, want empty", *deleteSession)
	}
}

func TestSessionModeResolution(t *testing.T) {
	tests := []struct {
		name              string
		sessionName       string
		isHelp            bool
		isFind            bool
		isInteractiveFind bool
		isInteractive     bool
		wantFind          bool
		wantInterFind     bool
		wantInteractive   bool
	}{
		{
			name:            "session alone enables normal interactive",
			sessionName:     "work",
			wantInteractive: true,
		},
		{
			name:            "session with -h does not force interactive",
			sessionName:     "work",
			isHelp:          true,
			wantInteractive: false,
		},
		{
			name:            "session with -f promotes to interactive find",
			sessionName:     "work",
			isFind:          true,
			wantFind:        false,
			wantInterFind:   true,
			wantInteractive: false,
		},
		{
			name:              "session with -if stays interactive find",
			sessionName:       "work",
			isInteractiveFind: true,
			wantInterFind:     true,
			wantInteractive:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isFind := tc.isFind
			isInteractiveFind := tc.isInteractiveFind
			isInteractive := tc.isInteractive

			if tc.sessionName != "" {
				if isFind && !isInteractiveFind {
					isFind = false
					isInteractiveFind = true
				}
				if !tc.isHelp && !isInteractive && !isInteractiveFind {
					isInteractive = true
				}
			}

			if isFind != tc.wantFind {
				t.Errorf("isFind = %v, want %v", isFind, tc.wantFind)
			}
			if isInteractiveFind != tc.wantInterFind {
				t.Errorf("isInteractiveFind = %v, want %v", isInteractiveFind, tc.wantInterFind)
			}
			if isInteractive != tc.wantInteractive {
				t.Errorf("isInteractive = %v, want %v", isInteractive, tc.wantInteractive)
			}
		})
	}
}


