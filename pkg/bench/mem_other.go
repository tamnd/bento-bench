//go:build !unix

package bench

import "os"

// maxRSSBytes has no portable rusage source off unix, so it reports zero, which
// the report renders as "n/a". The harness runs on Linux in CI and on macOS in
// development, where the unix build carries the real measurement.
func maxRSSBytes(_ *os.ProcessState) int64 { return 0 }
