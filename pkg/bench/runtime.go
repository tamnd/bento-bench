package bench

import (
	"os"
	"os/exec"
)

// Runtime is a JavaScript or TypeScript runtime the harness can drive. Args are
// the fixed arguments that come before the workload path, for runtimes that need
// a subcommand (Deno wants "run") or flags to stay quiet.
//
// Compile, when set, makes this a two-phase runtime: the workload is compiled to
// a native binary and the binary is what gets timed, with the compile step timed
// on its own. bento's ahead-of-time path uses it so the run column reflects the
// compiled Go program rather than the interpreter, which is the number the AOT
// compiler exists to move.
type Runtime struct {
	Name    string
	Bin     string
	Args    []string
	Compile *CompileStep
}

// CompileStep describes how a two-phase runtime turns a workload into a native
// binary. Args are the fixed arguments before the output and input; the harness
// appends the "-o <binary>" output and the workload path, so the full command is
// "<Bin> <Args...> -o <binary> <workload>".
type CompileStep struct {
	Bin  string
	Args []string
}

// command builds the exec arguments for the compile step of a two-phase runtime.
func (c *CompileStep) command(workload, out string) (string, []string) {
	args := make([]string, 0, len(c.Args)+3)
	args = append(args, c.Args...)
	args = append(args, "-o", out, workload)
	return c.Bin, args
}

// AOTBuildArgs are the compiler arguments bento's ahead-of-time path is driven
// with. They are exported so a caller can override them as the build command
// settles, while the default names the current subcommand.
var AOTBuildArgs = []string{"build"}

// DefaultRuntimes returns the four runtimes the harness knows about. bento is
// located through BENTO_BIN when set so CI can point at a freshly built binary,
// otherwise it is looked up on PATH like the others. bentoAOT selects bento's
// ahead-of-time path: instead of the interpreter, the workload is compiled to a
// Go binary and the binary is timed, with the compile step timed separately.
func DefaultRuntimes(bentoAOT bool) []Runtime {
	bento := os.Getenv("BENTO_BIN")
	if bento == "" {
		bento = "bento"
	}
	bentoRT := Runtime{Name: "bento", Bin: bento, Args: []string{"run"}}
	if bentoAOT {
		bentoRT = Runtime{
			Name:    "bento",
			Compile: &CompileStep{Bin: bento, Args: AOTBuildArgs},
		}
	}
	return []Runtime{
		{Name: "node", Bin: "node"},
		{Name: "deno", Bin: "deno", Args: []string{"run", "--quiet", "--allow-read", "--allow-write", "--allow-env"}},
		{Name: "bun", Bin: "bun", Args: []string{"run"}},
		bentoRT,
	}
}

// Available reports whether the runtime can be found and executed. For a
// two-phase runtime the compiler binary is what must exist; for a single-phase
// runtime it is the interpreter binary.
func (r Runtime) Available() bool {
	bin := r.Bin
	if r.Compile != nil {
		bin = r.Compile.Bin
	}
	if _, err := exec.LookPath(bin); err == nil {
		return true
	}
	// BENTO_BIN may be an absolute path that is not on PATH.
	if info, err := os.Stat(bin); err == nil && !info.IsDir() {
		return true
	}
	return false
}

// runArgs builds the arguments to run a workload file under a single-phase
// runtime: the fixed args followed by the workload path.
func (r Runtime) runArgs(workload string) []string {
	args := make([]string, 0, len(r.Args)+1)
	args = append(args, r.Args...)
	args = append(args, workload)
	return args
}
