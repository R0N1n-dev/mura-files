package shell

import (
	"bufio"
	"fmt"
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

// SessionBusAvailable reports whether a D-Bus session bus can be reached.
//
// Several desktop integrations (single instance hand-off, notifications,
// trash portals) need one. On minimal systems and inside containers there is
// often no bus at all, and the application must keep working in that case
// instead of refusing to start.
func SessionBusAvailable() bool {
	if runtime.GOOS != "linux" {
		return true
	}
	// An advertised bus address is only trustworthy when the socket it points
	// at actually exists; a stale variable would otherwise turn into a fatal
	// startup error inside the single instance manager.
	if address := os.Getenv("DBUS_SESSION_BUS_ADDRESS"); address != "" {
		if busAddressReachable(address) {
			return true
		}
	}
	// systemd user sessions expose the bus as a socket under /run/user.
	if info, err := os.Stat(fmt.Sprintf("/run/user/%d/bus", os.Getuid())); err == nil && !info.IsDir() {
		return true
	}
	// Falling back to spawning a private bus only works when the helper exists.
	if _, err := exec.LookPath("dbus-launch"); err == nil {
		return true
	}
	if _, err := exec.LookPath("dbus-daemon"); err == nil {
		return true
	}
	return false
}

// busAddressReachable inspects a D-Bus address string and reports whether the
// endpoint it names exists on this machine.
func busAddressReachable(address string) bool {
	for _, part := range strings.Split(address, ";") {
		for _, field := range strings.Split(part, ",") {
			key, value, ok := strings.Cut(field, "=")
			if !ok {
				continue
			}
			switch key {
			case "path":
				if _, err := os.Stat(value); err != nil {
					return false
				}
				return true
			case "abstract":
				// Abstract sockets have no filesystem presence to check.
				return true
			}
		}
	}
	// Anything we cannot interpret (a TCP bus, for instance) is assumed usable.
	return strings.HasPrefix(address, "tcp:") || strings.HasPrefix(address, "unix:abstract=")
}
