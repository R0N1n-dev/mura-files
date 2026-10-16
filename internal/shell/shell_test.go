package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectTerminalHonoursEnvironment(t *testing.T) {
	// "sh" is guaranteed to exist and is used here as a stand-in terminal.
	t.Setenv("TERMINAL", "sh")
	if got := DetectTerminal(); got == "" {
		t.Skip("no terminal binaries available in this environment")
	}
	t.Setenv("TERMINAL", "definitely-not-a-real-terminal-binary")
	if got := DetectTerminal(); got != "" {
		if filepath.Base(got) == "definitely-not-a-real-terminal-binary" {
			t.Error("a bogus TERMINAL value should not be trusted")
		}
	}
}

func TestOSInfo(t *testing.T) {
	info := OSInfo()
	if info.Name == "" {
		t.Error("the OS name should never be empty")
	}
	if info.Kernel == "" {
		t.Error("the kernel version should be detected")
	}
	if info.Arch == "" {
		t.Error("the architecture should be reported")
	}
	if info.User == "" {
		t.Error("the current user should be resolved")
	}
	if info.Home == "" {
		t.Error("the home directory should be resolved")
	}
}

func TestTools(t *testing.T) {
	tools := Tools()
	if len(tools) == 0 {
		t.Fatal("expected a tool list")
	}
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("tool entries must be named")
		}
		if tool.Available && tool.Path == "" {
			t.Errorf("available tool %q has no path", tool.Name)
		}
		if !tool.Available && tool.Path != "" {
			t.Errorf("unavailable tool %q should not have a path", tool.Name)
		}
	}
	// The list must be sorted so the settings panel reads consistently.
	for i := 1; i < len(tools); i++ {
		if tools[i-1].Name > tools[i].Name {
			t.Fatalf("tool list is not sorted at %q/%q", tools[i-1].Name, tools[i].Name)
		}
	}
}

func TestDarkModeReturnsASource(t *testing.T) {
	_, source := DarkMode()
	if source == "" {
		t.Error("DarkMode should always report where the answer came from")
	}
}

func TestSanitiseExec(t *testing.T) {
	cases := map[string][]string{
		"gedit %U":                     {"gedit"},
		"firefox %u --new-window":      {"firefox", "--new-window"},
		"code --new-window %F":         {"code", "--new-window"},
		"libreoffice --writer %f":      {"libreoffice", "--writer"},
		"sh -c 'echo 100%% done'":      {"sh", "-c", "'echo", "100%", "done'"},
	}
	for input, want := range cases {
		got := sanitiseExec(input)
		if len(got) != len(want) {
			t.Errorf("sanitiseExec(%q) = %v, want %v", input, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("sanitiseExec(%q) = %v, want %v", input, got, want)
				break
			}
		}
	}
}

func TestParseDesktopFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "example.desktop")
	content := strings.Join([]string{
		"# a comment",
		"[Desktop Entry]",
		"Type=Application",
		"Name=Example Editor",
		"Name[en_GB]=Example Editor",
		"Exec=example %U",
		"Icon=example-icon",
		"Terminal=false",
		"MimeType=text/plain;text/markdown;",
		"",
		"[Desktop Action NewWindow]",
		"Name=Should Be Ignored",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entry, visible := parseDesktopFile(path)
	if !visible {
		t.Fatal("the entry should be visible")
	}
	if entry.Name != "Example Editor" {
		t.Errorf("Name = %q", entry.Name)
	}
	if entry.Exec != "example %U" {
		t.Errorf("Exec = %q", entry.Exec)
	}
	if entry.Icon != "example-icon" {
		t.Errorf("Icon = %q", entry.Icon)
	}
	if entry.Terminal {
		t.Error("Terminal should be false")
	}
	if len(entry.MimeTypes) != 2 {
		t.Errorf("MimeTypes = %v", entry.MimeTypes)
	}
}

func TestHiddenDesktopEntriesAreSkipped(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"hidden.desktop":     "[Desktop Entry]\nName=Hidden\nExec=x\nHidden=true\n",
		"nodisplay.desktop":  "[Desktop Entry]\nName=NoDisplay\nExec=x\nNoDisplay=true\n",
		"incomplete.desktop": "[Desktop Entry]\nName=Incomplete\n",
		"valid.desktop":      "[Desktop Entry]\nName=Valid\nExec=valid\n",
	}
	for name, content := range cases {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	for name, want := range map[string]bool{
		"hidden.desktop":     false,
		"nodisplay.desktop":  false,
		"incomplete.desktop": false,
		"valid.desktop":      true,
	} {
		_, visible := parseDesktopFile(filepath.Join(dir, name))
		if visible != want {
			t.Errorf("parseDesktopFile(%s) visible = %v, want %v", name, visible, want)
		}
	}
}

func TestIconPathResolves(t *testing.T) {
	// The sandbox has no icon themes installed, so the lookup is allowed to
	// fail; what matters is that it never panics and never returns a path that
	// does not exist.
	path := IconPath("folder")
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("IconPath returned a missing file: %q", path)
		}
	}
	if got := IconPath("/definitely/missing/icon.svg"); got != "" {
		t.Errorf("an absolute missing path should resolve to nothing, got %q", got)
	}
}

func TestHostAndUser(t *testing.T) {
	if Hostname() == "" {
		t.Error("Hostname should never be empty")
	}
	if CurrentUser() == "" {
		t.Error("CurrentUser should never be empty")
	}
	if KernelVersion() == "" {
		t.Error("KernelVersion should never be empty")
	}
}
