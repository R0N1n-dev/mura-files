// Package shell integrates with the Linux desktop: launching applications,
// resolving .desktop entries, opening terminals, reading icon themes and
// collecting system information for the about dialog.
package shell

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/R0N1n-dev/mura-files/internal/model"
	"github.com/R0N1n-dev/mura-files/internal/places"
)

// knownTerminals lists terminal emulators in order of preference.
var knownTerminals = []struct {
	Bin  string
	Args []string
}{
	{"ptyxis", []string{"--working-directory=%s"}},
	{"kgx", []string{"--working-directory=%s"}},
	{"gnome-terminal", []string{"--working-directory=%s"}},
	{"konsole", []string{"--workdir", "%s"}},
	{"kitty", []string{"--directory", "%s"}},
	{"alacritty", []string{"--working-directory", "%s"}},
	{"wezterm", []string{"start", "--cwd", "%s"}},
	{"foot", []string{"--working-directory=%s"}},
	{"tilix", []string{"--working-directory=%s"}},
	{"xfce4-terminal", []string{"--working-directory=%s"}},
	{"mate-terminal", []string{"--working-directory=%s"}},
	{"terminator", []string{"--working-directory=%s"}},
	{"urxvt", []string{"-cd", "%s"}},
	{"xterm", []string{"-cd", "%s"}},
}

// DetectTerminal returns the first available terminal emulator, preferring an
// explicit user setting from the environment.
func DetectTerminal() string {
	if env := os.Getenv("TERMINAL"); env != "" {
		if p, err := exec.LookPath(env); err == nil {
			return p
		}
	}
	for _, t := range knownTerminals {
		if p, err := exec.LookPath(t.Bin); err == nil {
			return p
		}
	}
	return ""
}

// terminalArgs returns the argument template for a terminal binary.
func terminalArgs(bin string) []string {
	name := filepath.Base(bin)
	for _, t := range knownTerminals {
		if t.Bin == name {
			return t.Args
		}
	}
	return []string{"--working-directory=%s"}
}

// OpenTerminal launches a terminal emulator in the given directory.
func OpenTerminal(dir string) model.OpenResult {
	if dir == "" {
		dir = places.Home()
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		dir = filepath.Dir(dir)
	}
	bin := DetectTerminal()
	if bin == "" {
		return model.OpenResult{OK: false, Message: "No terminal emulator found"}
	}
	args := make([]string, 0, 2)
	for _, a := range terminalArgs(bin) {
		args = append(args, strings.ReplaceAll(a, "%s", dir))
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	if err := cmd.Start(); err != nil {
		return model.OpenResult{OK: false, Command: bin, Message: err.Error()}
	}
	go func() { _ = cmd.Wait() }()
	return model.OpenResult{OK: true, Command: bin + " " + strings.Join(args, " ")}
}

// Open launches a path with the desktop default handler.
func Open(path string) model.OpenResult {
	if path == "" {
		return model.OpenResult{OK: false, Message: "No path supplied"}
	}
	// Prefer xdg-open, then gio open, then gtk-launch.
	for _, candidate := range [][]string{
		{"xdg-open", path},
		{"gio", "open", path},
		{"kioclient5", "exec", path},
	} {
		bin, err := exec.LookPath(candidate[0])
		if err != nil {
			continue
		}
		cmd := exec.Command(bin, candidate[1:]...)
		if err := cmd.Start(); err != nil {
			continue
		}
		go func() { _ = cmd.Wait() }()
		return model.OpenResult{OK: true, Command: strings.Join(candidate, " ")}
	}
	return model.OpenResult{OK: false, Message: "No desktop opener available"}
}

// OpenWith launches a path with an explicit application.
func OpenWith(app, path string) model.OpenResult {
	if app == "" {
		return Open(path)
	}
	cmd := exec.Command(app, path)
	if err := cmd.Start(); err != nil {
		return model.OpenResult{OK: false, Command: app, Message: err.Error()}
	}
	go func() { _ = cmd.Wait() }()
	return model.OpenResult{OK: true, Command: app + " " + path}
}

// Execute runs a detached shell command in the given working directory. It is
// used by the "custom command" feature and by the editor setting.
func Execute(command, dir string) model.OpenResult {
	if strings.TrimSpace(command) == "" {
		return model.OpenResult{OK: false, Message: "Empty command"}
	}
	cmd := exec.Command("/bin/sh", "-c", command)
	if dir != "" {
		cmd.Dir = dir
	}
	if err := cmd.Start(); err != nil {
		return model.OpenResult{OK: false, Command: command, Message: err.Error()}
	}
	go func() { _ = cmd.Wait() }()
	return model.OpenResult{OK: true, Command: command}
}

// Hostname returns the system hostname, tolerating sandboxes without one.
func Hostname() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "localhost"
}

// CurrentUser returns the login name of the current user.
func CurrentUser() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	if v := os.Getenv("USER"); v != "" {
		return v
	}
	return "user"
}

// KernelVersion reports the running kernel release.
func KernelVersion() string {
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return runtime.GOOS
	}
	return strings.TrimSpace(string(data))
}
