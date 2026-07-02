package bench

import (
	"os"
	"os/exec"
	"strings"
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
// appends the output flag with the binary path and the workload path, so the
// full command is "<Bin> <Args...> <OutFlag> <binary> <workload>". OutFlag names
// the compiler's output option, which is "-o" for deno and bento but "--outfile"
// for bun; an empty OutFlag defaults to "-o".
type CompileStep struct {
	Bin     string
	Args    []string
	OutFlag string
}

// command builds the exec arguments for the compile step of a two-phase runtime.
func (c *CompileStep) command(workload, out string) (string, []string) {
	flag := c.OutFlag
	if flag == "" {
		flag = "-o"
	}
	args := make([]string, 0, len(c.Args)+3)
	args = append(args, c.Args...)
	args = append(args, flag, out, workload)
	return c.Bin, args
}

// AOTBuildArgs are the compiler arguments bento's ahead-of-time path is driven
// with. They are exported so a caller can override them as the build command
// settles, while the default names the current subcommand.
var AOTBuildArgs = []string{"build"}

// DefaultRuntimes returns the four runtimes the harness knows about. bento is
// located through BENTO_BIN when set so CI can point at a freshly built binary,
// otherwise it is looked up on PATH like the others.
//
// Three of the four are measured through a compile-to-binary path, so the run
// column and the binary-size column compare like with like: bento compiles each
// workload to a native Go binary with `bento build`, bun compiles to a single
// executable with `bun build --compile`, and deno compiles with `deno compile`.
// Each binary is what gets timed, with the compile step timed on its own.
//
// node stays the single-phase interpreter reference: its single-executable path
// (SEA) is experimental, needs the external postject tool, and does not accept
// TypeScript directly, so compiling a .ts workload to one node binary is not a
// clean single command the way the other three are. node therefore carries no
// binary-size cell rather than a number it did not earn the same way.
func DefaultRuntimes() []Runtime {
	bento := os.Getenv("BENTO_BIN")
	if bento == "" {
		bento = "bento"
	}
	return []Runtime{
		{Name: "node", Bin: "node"},
		{Name: "deno", Compile: &CompileStep{Bin: "deno", Args: []string{"compile", "--quiet", "--allow-read", "--allow-write", "--allow-env"}}},
		{Name: "bun", Compile: &CompileStep{Bin: "bun", Args: []string{"build", "--compile"}, OutFlag: "--outfile"}},
		{Name: "bento", Compile: &CompileStep{Bin: bento, Args: AOTBuildArgs}},
	}
}

// Without drops every runtime whose name appears in the comma-separated list,
// case-insensitively. It is how CI can leave the slow interpreter out of a run
// while the default local run still measures it, without touching the runtime
// table itself. An empty or blank list changes nothing.
func Without(runtimes []Runtime, skip string) []Runtime {
	drop := map[string]bool{}
	for name := range strings.SplitSeq(skip, ",") {
		if s := strings.ToLower(strings.TrimSpace(name)); s != "" {
			drop[s] = true
		}
	}
	if len(drop) == 0 {
		return runtimes
	}
	out := make([]Runtime, 0, len(runtimes))
	for _, rt := range runtimes {
		if !drop[strings.ToLower(rt.Name)] {
			out = append(out, rt)
		}
	}
	return out
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
