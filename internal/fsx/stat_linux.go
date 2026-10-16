//go:build linux

package fsx

import (
	"os"
	"syscall"
)

// statExtra extracts the Linux specific stat fields we surface in the UI.
func statExtra(info os.FileInfo) (uid, gid, nlink uint32, atimeMs, ctimeMs int64, ok bool) {
	st, isStat := info.Sys().(*syscall.Stat_t)
	if !isStat {
		return 0, 0, 0, 0, 0, false
	}
	return st.Uid, st.Gid, uint32(st.Nlink),
		st.Atim.Sec*1000 + st.Atim.Nsec/1e6,
		st.Ctim.Sec*1000 + st.Ctim.Nsec/1e6,
		true
}
