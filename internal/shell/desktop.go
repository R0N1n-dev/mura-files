package shell

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/R0N1n-dev/mura-files/internal/model"
	"github.com/R0N1n-dev/mura-files/internal/places"
)

// desktopEntry is a parsed application launcher.
type desktopEntry struct {
	Name      string
	Exec      string
	Icon      string
	Path      string
	Terminal  bool
	NoDisplay bool
	Hidden    bool
	MimeTypes []string
}

// applicationDirs returns the XDG directories that hold .desktop files.
func applicationDirs() []string {
	dirs := []string{}
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		dirs = append(dirs, filepath.Join(dataHome, "applications"))
	} else {
		dirs = append(dirs, filepath.Join(places.Home(), ".local", "share", "applications"))
	}
	for _, base := range filepath.SplitList(os.Getenv("XDG_DATA_DIRS")) {
		if base == "" {
			continue
		}
		dirs = append(dirs, filepath.Join(base, "applications"))
	}
	if len(dirs) == 1 {
		dirs = append(dirs,
			"/usr/local/share/applications",
			"/usr/share/applications",
			"/var/lib/flatpak/exports/share/applications",
			filepath.Join(places.Home(), ".local", "share", "flatpak", "exports", "share", "applications"),
		)
	}
	return dirs
}

func parseDesktopFile(path string) (desktopEntry, bool) {
	f, err := os.Open(path)
	if err != nil {
		return desktopEntry{}, false
	}
	defer f.Close()

	entry := desktopEntry{Path: path}
	inEntry := false
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 512*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "[") {
			inEntry = line == "[Desktop Entry]"
			continue
		}
		if !inEntry || line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch {
		case key == "Name":
			if entry.Name == "" {
				entry.Name = value
			}
		case strings.HasPrefix(key, "Name[en"):
			entry.Name = value
		case key == "Exec":
			entry.Exec = value
		case key == "Icon":
			entry.Icon = value
		case key == "Terminal":
			entry.Terminal = strings.EqualFold(value, "true")
		case key == "NoDisplay":
			entry.NoDisplay = strings.EqualFold(value, "true")
		case key == "Hidden":
			entry.Hidden = strings.EqualFold(value, "true")
		case key == "MimeType":
			for _, m := range strings.Split(value, ";") {
				if m = strings.TrimSpace(m); m != "" {
					entry.MimeTypes = append(entry.MimeTypes, m)
				}
			}
		}
	}
	if entry.Name == "" || entry.Exec == "" {
		return entry, false
	}
	return entry, !entry.Hidden && !entry.NoDisplay
}

// Applications lists launchable desktop applications, deduplicated by file
// name so that user overrides win over system entries.
func Applications() []model.AppInfo {
	seen := map[string]bool{}
	out := []model.AppInfo{}
	for _, dir := range applicationDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".desktop") || seen[e.Name()] {
				continue
			}
			seen[e.Name()] = true
			entry, visible := parseDesktopFile(filepath.Join(dir, e.Name()))
			if !visible {
				continue
			}
			out = append(out, model.AppInfo{
				Name: entry.Name,
				Path: entry.Path,
				Icon: entry.Icon,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// ApplicationsFor returns the applications registered for a MIME type.
func ApplicationsFor(mime string) []model.AppInfo {
	if mime == "" {
		return nil
	}
	seen := map[string]bool{}
	out := []model.AppInfo{}
	for _, dir := range applicationDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".desktop") || seen[e.Name()] {
				continue
			}
			seen[e.Name()] = true
			entry, visible := parseDesktopFile(filepath.Join(dir, e.Name()))
			if !visible {
				continue
			}
			for _, m := range entry.MimeTypes {
				if m == mime {
					out = append(out, model.AppInfo{Name: entry.Name, Path: entry.Path, Icon: entry.Icon})
					break
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// LaunchDesktopEntry starts an application from a .desktop file.
func LaunchDesktopEntry(path string) model.OpenResult {
	entry, ok := parseDesktopFile(path)
	if !ok {
		return model.OpenResult{OK: false, Message: "Not a launchable desktop entry"}
	}
	argv := sanitiseExec(entry.Exec)
	if len(argv) == 0 {
		return model.OpenResult{OK: false, Message: "Desktop entry has no command"}
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	if err := cmd.Start(); err != nil {
		return model.OpenResult{OK: false, Command: strings.Join(argv, " "), Message: err.Error()}
	}
	go func() { _ = cmd.Wait() }()
	return model.OpenResult{OK: true, Command: strings.Join(argv, " ")}
}

// sanitiseExec strips the field codes defined by the desktop entry spec.
func sanitiseExec(exec string) []string {
	replacer := strings.NewReplacer(
		"%U", "", "%u", "", "%F", "", "%f", "",
		"%i", "", "%c", "", "%k", "", "%%", "%",
	)
	fields := strings.Fields(replacer.Replace(exec))
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

// DesktopMeta returns the display name and icon name for a .desktop file.
func DesktopMeta(path string) (name, icon string) {
	entry, ok := parseDesktopFile(path)
	if !ok {
		return "", ""
	}
	return entry.Name, entry.Icon
}
