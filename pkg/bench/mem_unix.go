//go:build unix

package bench

import (
	"os"
	"runtime"
	"syscall"
)

// maxRSSBytes reads the peak resident set size of a finished process from its
// rusage, in bytes. The kernel reports the high-water mark of physical memory
// the process held, which is the number a user cares about: how much RAM the
// run actually took at its worst, not a sampled average that can miss a spike.
//
// The unit of ru_maxrss differs by platform, so it is normalized here: Linux
// reports kilobytes, the BSDs and macOS report bytes. A zero is returned when
// the process state carries no rusage, so a platform or a failure that leaves it
// empty reads as "unknown" rather than a false zero the report would trust.
func maxRSSBytes(ps *os.ProcessState) int64 {
	if ps == nil {
		return 0
	}
	ru, ok := ps.SysUsage().(*syscall.Rusage)
	if !ok || ru == nil {
		return 0
	}
	maxrss := int64(ru.Maxrss)
	if runtime.GOOS == "linux" {
		return maxrss * 1024
	}
	return maxrss
}
