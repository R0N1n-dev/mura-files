package shell

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/R0N1n-dev/mura-files/internal/model"
	"github.com/R0N1n-dev/mura-files/internal/places"
)

// parseOSRelease reads /etc/os-release into a key/value map.
func parseOSRelease() map[string]string {
	out := map[string]string{}
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"`)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// OSInfo assembles the operating system description shown in the UI.
func OSInfo() model.OSInfo {
	release := parseOSRelease()
	name := release["PRETTY_NAME"]
	if name == "" {
		name = release["NAME"]
	}
	if name == "" {
		name = runtime.GOOS
	}
	return model.OSInfo{
		ID:       release["ID"],
		Name:     name,
		Version:  release["VERSION_ID"],
		Kernel:   KernelVersion(),
		Arch:     runtime.GOARCH,
		Hostname: Hostname(),
		User:     CurrentUser(),
		Home:     places.Home(),
		Shell:    filepath.Base(os.Getenv("SHELL")),
		Desktop:  firstNonEmpty(os.Getenv("XDG_CURRENT_DESKTOP"), os.Getenv("DESKTOP_SESSION")),
		Session:  os.Getenv("XDG_SESSION_TYPE"),
	}
}

// Tools reports which optional helper binaries are available. The frontend
// uses this to enable or hide features that depend on them.
func Tools() []model.Tool {
	names := []string{
		"xdg-open", "gio", "udisksctl", "unzip", "zip", "tar", "7z", "7za",
		"unrar", "zstd", "ffmpegthumbnailer", "ffmpeg", "pdftoppm",
		"trash-put", "git", "rsync", "wl-copy", "xclip",
	}
	out := make([]model.Tool, 0, len(names))
	for _, n := range names {
		t := model.Tool{Name: n}
		if p, err := exec.LookPath(n); err == nil {
			t.Path = p
			t.Available = true
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// DarkMode asks the desktop whether the preferred colour scheme is dark. It
// checks the freedesktop appearance portal settings and the GTK theme name.
func DarkMode() (bool, string) {
	if v := os.Getenv("MURA_FORCE_DARK"); v == "1" {
		return true, "environment"
	}
	// GNOME / GTK4 expose the preference through gsettings.
	if _, err := exec.LookPath("gsettings"); err == nil {
		out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "color-scheme").Output()
		if err == nil {
			value := strings.ToLower(strings.TrimSpace(string(out)))
			if strings.Contains(value, "dark") {
				return true, "gsettings"
			}
			if strings.Contains(value, "light") || strings.Contains(value, "default") {
				return false, "gsettings"
			}
		}
	}
	theme := strings.ToLower(os.Getenv("GTK_THEME"))
	if theme == "" {
		if _, err := exec.LookPath("gsettings"); err == nil {
			out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "gtk-theme").Output()
			if err == nil {
				theme = strings.ToLower(strings.TrimSpace(string(out)))
			}
		}
	}
	if strings.Contains(theme, "dark") {
		return true, "theme"
	}
	if theme != "" {
		return false, "theme"
	}
	return false, "default"
}

// AccentColour reads the desktop accent colour when the portal exposes one.
func AccentColour() string {
	if _, err := exec.LookPath("gsettings"); err == nil {
		out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "accent-color").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	return ""
}
