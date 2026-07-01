// The smallest possible program: it isolates process startup and teardown,
// which is the cost you pay on every single invocation of a runtime.
process.stdout.write("ok\n");
