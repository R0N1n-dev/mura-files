package shell

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// iconSizePreference orders the icon sizes we look for. Scalable SVG wins so
// that app icons stay crisp at any zoom level.
var iconSizePreference = []string{
	"scalable", "512x512", "256x256", "192x192", "128x128", "96x96",
	"64x64", "48x48", "32x32", "24x24", "22x22", "16x16",
}

// iconThemePreference is the fallback chain used when the desktop theme does
// not contain a requested icon.
var iconThemePreference = []string{"hicolor", "Adwaita", "gnome", "breeze", "Papirus", "elementary"}

type iconIndex struct {
	once  sync.Once
	roots []string
	// index maps icon name -> list of candidate files, best first.
	index map[string][]string
}

var globalIcons = &iconIndex{}

// iconRoots returns every directory that may contain themed icons.
func iconRoots() []string {
	dirs := []string{}
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		dirs = append(dirs, filepath.Join(dataHome, "icons"))
	} else {
		dirs = append(dirs, filepath.Join(homeDir(), ".local", "share", "icons"))
	}
	dirs = append(dirs, filepath.Join(homeDir(), ".icons"))
	for _, base := range filepath.SplitList(os.Getenv("XDG_DATA_DIRS")) {
		if base != "" {
			dirs = append(dirs, filepath.Join(base, "icons"))
		}
	}
	dirs = append(dirs, "/usr/local/share/icons", "/usr/share/icons")
	return dirs
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return "/root"
}

// currentIconTheme reads the GTK icon theme name from the environment or the
// gsettings schema, defaulting to hicolor.
func currentIconTheme() string {
	if t := os.Getenv("GTK_ICON_THEME"); t != "" {
		return t
	}
	settings := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "gtk-3.0", "settings.ini")
	if os.Getenv("XDG_CONFIG_HOME") == "" {
		settings = filepath.Join(homeDir(), ".config", "gtk-3.0", "settings.ini")
	}
	if data, err := os.ReadFile(settings); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if value, ok := strings.CutPrefix(line, "gtk-icon-theme-name="); ok {
				if value != "" {
					return value
				}
			}
		}
	}
	return "hicolor"
}

// build scans the icon directories once and records every candidate path.
func (idx *iconIndex) build() {
	idx.index = map[string][]string{}
	roots := iconRoots()
	idx.roots = roots

	themes := append([]string{currentIconTheme()}, iconThemePreference...)
	rank := map[string]int{}
	for i, t := range themes {
		if _, ok := rank[t]; !ok {
			rank[t] = i
		}
	}
	sizeRank := map[string]int{}
	for i, s := range iconSizePreference {
		sizeRank[s] = i
	}

	best := map[string][]iconCandidate{}

	for _, root := range roots {
		themeEntries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, themeEntry := range themeEntries {
			themeName := themeEntry.Name()
			themeRank, known := rank[themeName]
			if !known {
				themeRank = len(themes) + 1
			}
			if !themeEntry.IsDir() {
				continue
			}
			// Walk theme/<size>/<category>/<name>.<ext>
			sizes, err := os.ReadDir(filepath.Join(root, themeName))
			if err != nil {
				continue
			}
			for _, sizeEntry := range sizes {
				if !sizeEntry.IsDir() {
					continue
				}
				sizeName := sizeEntry.Name()
				sizeIdx, ok := sizeRank[sizeName]
				if !ok {
					continue
				}
				categories, err := os.ReadDir(filepath.Join(root, themeName, sizeName))
				if err != nil {
					continue
				}
				for _, cat := range categories {
					if !cat.IsDir() {
						continue
					}
					files, err := os.ReadDir(filepath.Join(root, themeName, sizeName, cat.Name()))
					if err != nil {
						continue
					}
					for _, file := range files {
						if file.IsDir() {
							continue
						}
						ext := strings.ToLower(filepath.Ext(file.Name()))
						if ext != ".svg" && ext != ".png" && ext != ".xpm" {
							continue
						}
						name := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
						full := filepath.Join(root, themeName, sizeName, cat.Name(), file.Name())
						best[name] = append(best[name], iconCandidate{full, themeRank, sizeIdx})
					}
				}
			}
		}
	}

	for name, list := range best {
		sortCandidates(list)
		paths := make([]string, 0, len(list))
		for _, c := range list {
			paths = append(paths, c.path)
		}
		idx.index[name] = paths
	}
}

// iconCandidate is one on-disk icon file with its preference ranks.
type iconCandidate struct {
	path string
	rank int
	size int
}

// sortCandidates orders candidates by theme rank then size preference using a
// small insertion sort, which is plenty for the handful of hits per icon.
func sortCandidates(list []iconCandidate) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0; j-- {
			a, b := list[j-1], list[j]
			if a.rank < b.rank || (a.rank == b.rank && a.size <= b.size) {
				break
			}
			list[j-1], list[j] = list[j], list[j-1]
		}
	}
}

// IconPath resolves a themed icon name to a file on disk. It accepts both a
// bare theme name ("firefox") and an absolute path.
func IconPath(name string) string {
	if name == "" {
		return ""
	}
	if filepath.IsAbs(name) {
		if _, err := os.Stat(name); err == nil {
			return name
		}
		return ""
	}
	globalIcons.once.Do(globalIcons.build)
	if candidates, ok := globalIcons.index[name]; ok && len(candidates) > 0 {
		return candidates[0]
	}
	// Try the common extension suffix used by some icon themes.
	for _, suffix := range []string{"-symbolic", ".symbolic"} {
		if candidates, ok := globalIcons.index[name+suffix]; ok && len(candidates) > 0 {
			return candidates[0]
		}
	}
	return ""
}
