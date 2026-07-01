package bench

import (
	"os"
	"os/exec"
)

// Runtime is a JavaScript or TypeScript runtime the harness can drive. Args are
// the fixed arguments that come before the workload path, for runtimes that need
// a subcommand (Deno wants "run") or flags to stay quiet.
type Runtime struct {
	Name string
	Bin  string
	Args []string
}

// DefaultRuntimes returns the four runtimes the harness knows about. bento is
// located through BENTO_BIN when set so CI can point at a freshly built binary,
// otherwise it is looked up on PATH like the others.
func DefaultRuntimes() []Runtime {
	bento := os.Getenv("BENTO_BIN")
	if bento == "" {
		bento = "bento"
	}
	return []Runtime{
		{Name: "node", Bin: "node"},
		{Name: "deno", Bin: "deno", Args: []string{"run", "--quiet", "--allow-read", "--allow-write", "--allow-env"}},
		{Name: "bun", Bin: "bun", Args: []string{"run"}},
		{Name: "bento", Bin: bento, Args: []string{"run"}},
	}
}

// Available reports whether the runtime's binary can be found and executed.
func (r Runtime) Available() bool {
	if _, err := exec.LookPath(r.Bin); err == nil {
		return true
	}
	// BENTO_BIN may be an absolute path that is not on PATH.
	if info, err := os.Stat(r.Bin); err == nil && !info.IsDir() {
		return true
	}
	return false
}

// command builds the exec arguments to run a workload file under this runtime.
func (r Runtime) command(workload string) (string, []string) {
	args := make([]string, 0, len(r.Args)+1)
	args = append(args, r.Args...)
	args = append(args, workload)
	return r.Bin, args
}
