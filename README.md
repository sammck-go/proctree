# proctree

proctree is a simple package for working with a tree of processes.

### Features

- Easy to use with sensible defaults and golang option pattern
- Thread safe refresh mechanism
- Identity of Process objects preserved across refreshes
- Filters out kernel threads by default
- Can work with a subset of processes with provided root pids
- A command-line wrapper is included in cmd/proctree that allows you to display a process tree

### Install

**Binaries**

```sh
$ go install github.com/sammck-go/proctree/...
```


**Source**

```sh
$ go get -v github.com/sammck-go/proctree/...
```


### Commandline Usage

<!-- render these help texts by hand,
  or use https://github.com/jpillora/md-tmpl
    with $ md-tmpl -w README.md -->

<!--tmpl,code=plain:echo "$ proctree --help" && ( go build ./cmd/proctree && ./proctree --help ) -->
``` plain 
$ proctree --help
Usage: proctree [<option>...]

Print process tree details.

Options:
  -a, --include-ancestors        Include ancestors of roots. No effect if roots not provided.
                                 Disabled by default.
  -k, --include-kernel-threads   Include kernel threads. Disabled by default.
  -r, --root strings             Provides a pid to use as a root of the tree. May be repeated.
                                 By default, all orphaned processes are roots.
pflag: help requested
```
<!--/tmpl-->

### Package Usage


<!--tmpl:echo && godocdown -template ./.godocdown.template -->
```go
import "github.com/sammck-go/proctree"
```

Package proctree provides tools for inspecting, monitoring, and manipulating the
system process tree.

## Usage

#### type Config

```go
type Config struct {
}
```

Config provides configuration options for contruction of a ProcTree. The
constructed object is immutable after it is constructed by NewConfig.

#### func  NewConfig

```go
func NewConfig(opts ...ConfigOption) *Config
```
NewConfig creates a proctree Config object from provided options. The resulting
object can be passed to New using WithConfig.

#### func (*Config) Refine

```go
func (cfg *Config) Refine(opts ...ConfigOption) *Config
```
Refine creates a new Config object by applying ConfigOptions to an existing
config.

#### type ConfigOption

```go
type ConfigOption func(*Config)
```

ConfigOption is an opaque configuration option setter created by one of the With
functions. It follows the Golang "options" pattern.

#### func  WithConfig

```go
func WithConfig(other *Config) ConfigOption
```
WithConfig allows initialization of a new configuration object starting with an
existing one, and incremental initialization of configuration separately from
initialization of the PidFile. If provided, this option should be appear first
in the option list, since it replaces all configuration values.

#### func  WithKernelThreads

```go
func WithKernelThreads() ConfigOption
```
WithKernelThreads enables inclusion of kernel threads (children of pid 2). By
default, kernel threads are excluded.

#### func  WithRootAncestors

```go
func WithRootAncestors() ConfigOption
```
WithRootAncestors enables inclusion of all Processes that are ancestors of the
pids configured with WithRootPid. Has no effect if WithRootPid is not specified.
By default, root ancestors are excluded and the root Processes appear at the
first level of the tree.

#### func  WithRootPid

```go
func WithRootPid(pid int) ConfigOption
```
WithRootPid adds a pid to the set of pids to be included as roots of the tree.
By default, all orphaned processes are included as roots.

#### func  WithoutKernelThreads

```go
func WithoutKernelThreads() ConfigOption
```
WithoutKernelThreads disables inclusion of kernel threads (children of pid 2).
This is the default option.

#### func  WithoutRootAncestors

```go
func WithoutRootAncestors() ConfigOption
```
WithoutRootAncestors disables inclusion of Processes that are ancestors of the
pids configured with WithRootPid. Has no effect if WithRootPid is not specified.
Root ancestors are excluded and the root Processes appear at the first level of
the tree. This is the default setting.

#### func  WithoutRootPid

```go
func WithoutRootPid() ConfigOption
```
WithoutRootPid removes all pids added with WithRootPid, restoring config the
default, which is to include all orphaned processses.

#### type ProcTree

```go
type ProcTree struct {
}
```

ProcTree represents a session that inspects, monitors, and manipulates the
system process tree

#### func  New

```go
func New(opts ...ConfigOption) (*ProcTree, error)
```
New creates a new process tree management object and populates it with an
initial snapshot

#### func (*ProcTree) Close

```go
func (pt *ProcTree) Close() error
```
Close implements io.Closer. Shuts down the ProcTree and releases resources

#### func (*ProcTree) PidProcess

```go
func (pt *ProcTree) PidProcess(pid int) (proc *Process, ok bool)
```
PidProcess looks up a Process in the current snapshot by PID. If there is no
process with the provided PID, (nil, false) is returned.

#### func (*ProcTree) Processes

```go
func (pt *ProcTree) Processes() []*Process
```
Processes returns a snapshot of the list of Process objects the tree, sorted in
ascending PID order. If root pids were provided at configuration time, only
processes descended from the provided root Processes will be returned.

#### func (*ProcTree) Roots

```go
func (pt *ProcTree) Roots() []*Process
```
Roots returns a snapshot of the list of all included Process objects that are
toplevel roots, sorted in ascending PID order.

#### func (*ProcTree) SortProcessesByPid

```go
func (pt *ProcTree) SortProcessesByPid(procs []*Process)
```
SortProcessesByPid sorts a slice of Processes in increasing pid order.

#### func (*ProcTree) Update

```go
func (pt *ProcTree) Update(pruneTombstones bool) error
```
Update refreshes the ProcTree session with a new snapshot view of current
processes. Process objects from the previous snapshot are preserved, but may
become tombstoned.

#### func (*ProcTree) Walk

```go
func (pt *ProcTree) Walk(h ProcessHandler) error
```
Walk walks all subtrees starting at the configured root Process objects,
invoking a handler for each. Roots are walked in pid order; within each root
Processes are walked in depth-first order with children sorted in pid order.

#### func (*ProcTree) WalkFromRoots

```go
func (pt *ProcTree) WalkFromRoots(roots []*Process, h ProcessHandler) error
```
WalkFromRoots walks all subtrees starting at this provided root Process objects,
invoking a handler for each. Roots are walked in provided order; within each
root Processes are walked in depth-first order with children sorted in pid
order. It is the caller's responsibility to ensure that no root is a descendant
of another; otherwise the handler will be called multiple times for the same
Process.

#### type Process

```go
type Process struct {
}
```

Process repressents an abstraction of a single process within a ProcTree
session. It maintains its identity within a single session.

#### func (*Process) Children

```go
func (p *Process) Children() []*Process
```
Children returns an immutable snapshot slice of Processes known to be a child of
the Process. Only children that meet configured filter conditions (e.g., are in
configured root subtrees or ancestor paths) are included. This will include
tombstoned children that have been added since the last time tombstones were
pruned.

#### func (*Process) Depth

```go
func (p *Process) Depth() int
```
Depth computes the depth of this process in the process tree. 0 is returned for
root processes; 1 for their children; etc.

#### func (*Process) Executable

```go
func (p *Process) Executable() string
```
Executable returns the executable name associated with a process, without the
directory path

#### func (*Process) IsAncestorOf

```go
func (p *Process) IsAncestorOf(descendant *Process) bool
```
IsAncestorOf returns true if the Process is a known ancestor of a provided
descendant Process

#### func (*Process) IsDescendantOf

```go
func (p *Process) IsDescendantOf(ancestor *Process) bool
```
IsDescendantOf returns true if the Process is a known descendant of a provided
ancestor Process

#### func (*Process) OrigParent

```go
func (p *Process) OrigParent() *Process
```
OrigParent returns the Process that was the parent before this process became
detached (reattached to pid 1), if it is known. Otherwise, returns the parent at
the time the session was initialized, which may be null or pid 1.

#### func (*Process) Parent

```go
func (p *Process) Parent() *Process
```
Parent returns the Process that is the parent of this Process, or nil if the
Process does not have a parent that is configured for inclusion.

#### func (*Process) Pid

```go
func (p *Process) Pid() int
```
Pid returns the pid of a Process

#### func (*Process) WalkAncestry

```go
func (p *Process) WalkAncestry(h ProcessHandler) error
```
WalkAncestry walks the ancestry list starting at this process, up to the root,
invoking a handler for each. Only Processes enabled by configuration are
included

#### func (*Process) WalkSubtree

```go
func (p *Process) WalkSubtree(h ProcessHandler) error
```
WalkSubtree walks an entire subtree starting at this process as the root,
invoking a handler for each. Processes are walked in depth-first order with
children sorted in pid order. Only subtrees enabled by configuration are
included

#### type ProcessHandler

```go
type ProcessHandler func(*Process) error
```

ProcessHandler represents a function that is called back to act on a process.
Used for process tree walking operations.
<!--/tmpl-->

### Caveats

- Has not been tested on Windows. In particular, WithoutKernelThreads may cause real processes to be hidden.

### Contributing

- http://golang.org/doc/code.html
- http://golang.org/doc/effective_go.html
- `github.com/sammck-go/proctree/proctree.go` contains the importable package
- `github.com/sammck-go/proctree/cmd/proctree.go` contains the command-line wrapper tool

### Changelog

- `1.0` - Initial release
