package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/sammck-go/proctree"
	flag "github.com/spf13/pflag"
	"github.com/xlab/treeprint"
)

func addProc(root treeprint.Tree, pidToTree map[int]treeprint.Tree, proc *proctree.Process) error {
	pid := proc.Pid()
	parentTree := root
	parentProc := proc.Parent()
	if parentProc != nil {
		parentPid := parentProc.Pid()
		var ok bool
		parentTree, ok = pidToTree[parentPid]
		if !ok {
			return fmt.Errorf("Process with pid %d has parent pid %d but it is not in treeprint map", pid, parentPid)
		}
	}
	nodeTree := parentTree.AddMetaBranch(pid, proc.Executable())
	pidToTree[pid] = nodeTree

	for _, childProc := range proc.Children() {
		addProc(root, pidToTree, childProc)
	}

	return nil
}

func run() int {

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [<option>...]\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "Print process tree details.\n\n")
		fmt.Fprintln(os.Stderr, "Options:")

		flag.PrintDefaults()
	}

	includeKernelThreads := false
	includeAncestors := false
	rootPidStrs := []string{}
	flag.BoolVarP(&includeKernelThreads, "include-kernel-threads", "k", false, "Include kernel threads. Disabled by default.")
	flag.BoolVarP(&includeAncestors, "include-ancestors", "a", false, "Include ancestors of roots. No effect if roots not provided.\nDisabled by default.")
	flag.StringSliceVarP(&rootPidStrs, "root", "r", []string{}, "Provides a pid to use as a root of the tree. May be repeated.\nBy default, all orphaned processes are roots.")

	flag.Parse()

	cfg := proctree.NewConfig()

	if len(rootPidStrs) > 0 {
		for _, pidStr := range rootPidStrs {
			pid, err := strconv.Atoi(pidStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "proctree: Invalid pid \"%s\" supplied to --root: %s\n", pidStr, err)
				return 1
			}
			cfg = cfg.Refine(proctree.WithRootPid(pid))
		}
	}

	if includeAncestors {
		cfg = cfg.Refine(proctree.WithRootAncestors())
	}

	if includeKernelThreads {
		cfg = cfg.Refine(proctree.WithKernelThreads())
	}

	if len(flag.Args()) != 0 {
		fmt.Fprintln(os.Stderr, "proctree: Too many command line arguments")
		fmt.Fprintln(os.Stderr)
		flag.Usage()
		return 1
	}

	pt, err := proctree.New(proctree.WithConfig(cfg))
	if err != nil {
		fmt.Fprintln(os.Stderr, "proctree: Could not build process tree: ", err)
		return 1
	}

	defer pt.Close()

	pidToTree := map[int]treeprint.Tree{}

	root := treeprint.New()

	for _, proc := range pt.Roots() {
		err = addProc(root, pidToTree, proc)
		if err != nil {
			fmt.Fprintln(os.Stderr, "proctree: Unable to build printable tree: ", err)
			return 1
		}
	}

	fmt.Println(root.String())

	return 0
}

func main() {
	exitCode := run()
	os.Exit(exitCode)
}
