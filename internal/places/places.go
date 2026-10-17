// Package places discovers well-known user directories, mounted filesystems
// and removable drives so the sidebar can present a familiar layout.
package places

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/R0N1n-dev/mura-files/internal/fsx"
	"github.com/R0N1n-dev/mura-files/internal/model"
)

// Home returns the current user's home directory.
func Home() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return "/"
}

// ConfigDir returns the XDG configuration directory for the application.
func ConfigDir() string {
	if c := os.Getenv("XDG_CONFIG_HOME"); c != "" {
		return filepath.Join(c, "mura-files")
	}
	return filepath.Join(Home(), ".config", "mura-files")
}

// DataDir returns the XDG data directory for the application.
func DataDir() string {
	if c := os.Getenv("XDG_DATA_HOME"); c != "" {
		return filepath.Join(c, "mura-files")
	}
	return filepath.Join(Home(), ".local", "share", "mura-files")
}

// CacheDir returns the XDG cache directory for the application.
func CacheDir() string {
	if c := os.Getenv("XDG_CACHE_HOME"); c != "" {
		return filepath.Join(c, "mura-files")
	}
	return filepath.Join(Home(), ".cache", "mura-files")
}

// wellKnownSpec maps the XDG user-dirs key to a sidebar presentation.
type wellKnownSpec struct {
	Key  string
	Name string
	Icon string
	Kind string
}

var wellKnownSpecs = []wellKnownSpec{
	{"XDG_DESKTOP_DIR", "Desktop", "desktop", "desktop"},
	{"XDG_DOCUMENTS_DIR", "Documents", "documents", "document"},
	{"XDG_DOWNLOAD_DIR", "Downloads", "download", "download"},
	{"XDG_MUSIC_DIR", "Music", "music", "music"},
	{"XDG_PICTURES_DIR", "Pictures", "pictures", "picture"},
	{"XDG_VIDEOS_DIR", "Videos", "videos", "video"},
	{"XDG_PUBLICSHARE_DIR", "Public", "public", "public"},
	{"XDG_TEMPLATES_DIR", "Templates", "templates", "template"},
}

// userDirs parses ~/.config/user-dirs.dirs, falling back to conventional names.
func userDirs() map[string]string {
	out := map[string]string{}
	path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "user-dirs.dirs")
	if os.Getenv("XDG_CONFIG_HOME") == "" {
		path = filepath.Join(Home(), ".config", "user-dirs.dirs")
	}
	f, err := os.Open(path)
	if err == nil {
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, value, found := strings.Cut(line, "=")
			if !found {
				continue
			}
			value = strings.Trim(value, `"`)
			value = strings.ReplaceAll(value, "$HOME", Home())
			out[strings.TrimSpace(key)] = filepath.Clean(value)
		}
	}
	home := Home()
	fallbacks := map[string]string{
		"XDG_DESKTOP_DIR":     filepath.Join(home, "Desktop"),
		"XDG_DOCUMENTS_DIR":   filepath.Join(home, "Documents"),
		"XDG_DOWNLOAD_DIR":    filepath.Join(home, "Downloads"),
		"XDG_MUSIC_DIR":       filepath.Join(home, "Music"),
		"XDG_PICTURES_DIR":    filepath.Join(home, "Pictures"),
		"XDG_VIDEOS_DIR":      filepath.Join(home, "Videos"),
		"XDG_PUBLICSHARE_DIR": filepath.Join(home, "Public"),
		"XDG_TEMPLATES_DIR":   filepath.Join(home, "Templates"),
	}
	for k, v := range fallbacks {
		if _, ok := out[k]; !ok {
			out[k] = v
		}
	}
	return out
}

// WellKnown returns the sidebar "Places" list.
func WellKnown() []model.Place {
	dirs := userDirs()
	home := Home()
	out := []model.Place{{
		Name:    "Home",
		Path:    home,
		Kind:    "home",
		Icon:    "home",
		Present: true,
	}}
	for _, spec := range wellKnownSpecs {
		p := dirs[spec.Key]
		if p == "" {
			continue
		}
		_, err := os.Stat(p)
		out = append(out, model.Place{
			Name:    spec.Name,
			Path:    p,
			Kind:    spec.Kind,
			Icon:    spec.Icon,
			Present: err == nil,
		})
	}
	out = append(out, model.Place{
		Name:    "Trash",
		Path:    "trash:///",
		Kind:    "trash",
		Icon:    "trash",
		Present: true,
	})
	return out
}

// ---------------------------------------------------------------------------
// Mounted filesystems
// ---------------------------------------------------------------------------

// pseudoFilesystems are kernel-internal mounts that should not clutter the UI.
var pseudoFilesystems = map[string]bool{
	"proc": true, "sysfs": true, "devtmpfs": true, "devpts": true,
	"cgroup": true, "cgroup2": true, "securityfs": true,
	"pstore": true, "autofs": true, "debugfs": true, "tracefs": true,
	"configfs": true, "fusectl": true, "mqueue": true, "hugetlbfs": true,
	"binfmt_misc": true, "bpf": true, "nsfs": true, "ramfs": true,
	"overlay": true, "squashfs": true, "efivarfs": true, "rpc_pipefs": true,
	"fuse.portal": true, "portal": true,
}

// networkFilesystems are remote mounts shown with a network badge.
var networkFilesystems = map[string]bool{
	"nfs": true, "nfs4": true, "cifs": true, "smb3": true, "smbfs": true,
	"sshfs": true, "fuse.sshfs": true, "davfs": true, "fuse.rclone": true,
	"9p": true, "afs": true, "ceph": true, "fuse.s3fs": true,
}

func unescapeMount(s string) string {
	repl := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`)
	return repl.Replace(s)
}

type mountRecord struct {
	device string
	path   string
	fsType string
	opts   string
}

func readMounts() []mountRecord {
	var out []mountRecord
	for _, file := range []string{"/proc/self/mounts", "/proc/mounts"} {
		f, err := os.Open(file)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			fields := strings.Fields(sc.Text())
			if len(fields) < 4 {
				continue
			}
			out = append(out, mountRecord{
				device: unescapeMount(fields[0]),
				path:   unescapeMount(fields[1]),
				fsType: fields[2],
				opts:   fields[3],
			})
		}
		f.Close()
		break
	}
	return out
}

// Statfs reports total and free space for the filesystem holding path.
func Statfs(path string) (total, free, avail uint64, err error) {
	var st syscall.Statfs_t
	if err = syscall.Statfs(path, &st); err != nil {
		return 0, 0, 0, err
	}
	bs := uint64(st.Bsize)
	return st.Blocks * bs, st.Bfree * bs, st.Bavail * bs, nil
}

// Usage returns a model.Usage for a path.
func Usage(path string) model.Usage {
	u := model.Usage{Path: path}
	total, free, _, err := Statfs(path)
	if err != nil {
		return u
	}
	u.Total = total
	u.Free = free
	if total > 0 {
		u.Used = total - free
		u.Percent = float64(u.Used) / float64(total) * 100
	}
	for _, m := range readMounts() {
		if m.path == path {
			u.FsType = m.fsType
			break
		}
	}
	return u
}

// Drives returns the mount points worth showing in the sidebar.
func Drives() []model.Drive {
	seen := map[string]bool{}
	var out []model.Drive
	for _, m := range readMounts() {
		// Kernel-internal mounts are hidden, but the root filesystem and
		// anything the user explicitly plugged in are always shown, even when
		// they happen to use a normally-internal filesystem type (a live USB
		// on tmpfs, a container root on overlay, and so on).
		if pseudoFilesystems[m.fsType] && m.path != "/" && !isRemovableMountPath(m.path) {
			continue
		}
		if seen[m.path] {
			continue
		}
		if _, err := os.Stat(m.path); err != nil {
			continue
		}
		seen[m.path] = true

		d := model.Drive{
			Path:      m.path,
			Device:    m.device,
			FsType:    m.fsType,
			ReadOnly:  strings.Contains(m.opts, "ro") && !strings.Contains(m.opts, "rw"),
			Network:   networkFilesystems[m.fsType],
			Removable: isRemovable(m),
			Mounted:   true,
		}
		d.Label = labelFor(m)
		d.Name = d.Label
		if d.Name == "" {
			if m.path == "/" {
				d.Name = "Filesystem"
			} else {
				d.Name = filepath.Base(m.path)
			}
		}
		d.System = isSystemMount(m.path)
		if total, free, _, err := Statfs(m.path); err == nil && total > 0 {
			d.Total = total
			d.Free = free
			d.Used = total - free
			d.Percent = float64(d.Used) / float64(total) * 100
		}
		out = append(out, d)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].System != out[j].System {
			return out[i].System
		}
		if out[i].Removable != out[j].Removable {
			return !out[i].Removable
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func isSystemMount(path string) bool {
	switch {
	case path == "/":
		return true
	case strings.HasPrefix(path, "/boot"):
		return true
	case strings.HasPrefix(path, "/usr"), strings.HasPrefix(path, "/var"),
		strings.HasPrefix(path, "/home"), strings.HasPrefix(path, "/opt"),
		strings.HasPrefix(path, "/srv"), strings.HasPrefix(path, "/nix"):
		return true
	case strings.HasPrefix(path, "/media"), strings.HasPrefix(path, "/run/media"),
		strings.HasPrefix(path, "/mnt"):
		return false
	}
	return true
}

// isRemovableMountPath reports whether a mount point is one of the
// conventional auto-mount locations for removable media.
func isRemovableMountPath(path string) bool {
	return strings.HasPrefix(path, "/media/") || strings.HasPrefix(path, "/run/media/") ||
		strings.HasPrefix(path, "/mnt/")
}

func isRemovable(m mountRecord) bool {
	if isRemovableMountPath(m.path) {
		return true
	}
	switch {
	case strings.HasPrefix(m.device, "/dev/sd"), strings.HasPrefix(m.device, "/dev/mmcblk"),
		strings.HasPrefix(m.device, "/dev/nvme"), strings.HasPrefix(m.device, "/dev/vd"):
		return false
	case strings.HasPrefix(m.device, "/dev/"):
		return true
	}
	return false
}

// labelFor derives a friendly name for a mount, preferring filesystem labels.
func labelFor(m mountRecord) string {
	if m.device == "" || !strings.HasPrefix(m.device, "/dev/") {
		return ""
	}
	base := filepath.Base(m.device)
	if label := readDeviceLabel(base); label != "" {
		return label
	}
	return base
}

// readDeviceLabel reads the filesystem label from udev's by-label directory.
func readDeviceLabel(dev string) string {
	entries, err := os.ReadDir("/dev/disk/by-label")
	if err != nil {
		return ""
	}
	for _, e := range entries {
		target, err := filepath.EvalSymlinks(filepath.Join("/dev/disk/by-label", e.Name()))
		if err != nil {
			continue
		}
		if filepath.Base(target) == dev {
			return e.Name()
		}
	}
	return ""
}

// Eject unmounts a removable drive using udisksctl when available and falls
// back to plain umount.
func Eject(path string) error {
	if _, err := exec.LookPath("udisksctl"); err == nil {
		if err := exec.Command("udisksctl", "unmount", "-b", path).Run(); err == nil {
			return nil
		}
	}
	if _, err := exec.LookPath("gio"); err == nil {
		if err := exec.Command("gio", "mount", "-u", path).Run(); err == nil {
			return nil
		}
	}
	return exec.Command("umount", path).Run()
}

// Mount asks udisksctl to mount a device or mount point.
func Mount(path string) error {
	if _, err := exec.LookPath("udisksctl"); err == nil {
		if err := exec.Command("udisksctl", "mount", "-b", path).Run(); err == nil {
			return nil
		}
	}
	return exec.Command("mount", path).Run()
}

// IsRemovablePath reports whether a path lives on a removable mount.
func IsRemovablePath(path string) bool {
	path = fsx.Abs(path)
	for _, d := range Drives() {
		if d.Removable && fsx.IsSameOrInside(d.Path, path) {
			return true
		}
	}
	return false
}

// FormatBytes renders a byte count in a compact human readable form.
func FormatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return strconv.FormatInt(n, 10) + " B"
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return strconv.FormatFloat(float64(n)/float64(div), 'f', 1, 64) + " " + string("KMGTPE"[exp]) + "B"
}
