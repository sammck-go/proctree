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

[![Releases](https://img.shields.io/github/release/sammck-go/proctree.svg)](https://github.com/sammck-go/proctree/releases) [![Releases](https://img.shields.io/github/downloads/sammck-go/proctree/total.svg)](https://github.com/sammck-go/proctree/releases)

See [the latest release](https://github.com/sammck-go/proctree/releases/latest)


**Source**

```sh
$ go get -v github.com/sammck-go/proctree
```


### Commandline Usage

<!-- render these help texts by hand,
  or use https://github.com/jpillora/md-tmpl
    with $ md-tmpl -w README.md -->

<!--tmpl,code=plain:echo "$ proctree --help" && go run cmd/proctree/proctree.go --help -->
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
exit status 2
```
<!--/tmpl-->

### Package Usage


<!--tmpl:echo && godocdown -template ./.godocdown.template -->
Error parsing template "./.godocdown.template": open ./.godocdown.template: no such file or directory<!--/tmpl-->

### Caveats

- In order so provide reliable file locking, a parallel file with a ".lock" extension is created in the same directory
  as the proctree. This lockfile is not deleted as that would defeat its purpose.  The locking feature
  can be disabled with an option, with the consequence that two processes may claim the same proctree, with only the last one's
  pid actually being readable.

### Contributing

- http://golang.org/doc/code.html
- http://golang.org/doc/effective_go.html
- `github.com/sammck-go/proctree/proctree.go` contains the importable package
- `github.com/sammck-go/proctree/cmd/with-proctree.go` contains the command-line wrapper tool

### Changelog

- `1.0` - Initial release
