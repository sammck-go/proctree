/*
Package proctree provides tools for inspecting, monitoring, and manipulating the system process tree.
*/
package proctree

import (
	"fmt"
	"sort"
	"sync"

	gops "github.com/mitchellh/go-ps"
)

// ProcTree represents a session that inspects, monitors, and manipulates the system process tree
type ProcTree struct {
	// lock is a general-purpose mutex for the proctree, used for updating the tree.
	lock sync.Mutex

	// Config is the immutable configuration provided at New time.
	cfg *Config

	// pidMap is a map of all known pids an their associated processes. Includes Processes excluded by configuration and unpruned tombstones.
	pidMap map[int]*Process

	// absProcs is a slice of all Process objects, sorted by pid.  Includes Processes excluded by configuration and unpruned tombstones.
	absProcs []*Process

	// absRootProcs is a slice of all Process objects that are roots of the absolute process tree.  Includes roots excluded by configuration and unpruned tombstones.
	absRootProcs []*Process

	// cfgRootProcs is a slice of all Process objects that were explicitly configured roots, sorted by pid.  Includes unpruned tombstones.
	cfgRootProcs []*Process

	// includedProcs is a slice of all Process objects that are roots or descendants of rootsof the process tree, sorted by pid.
	// Includes unpruned tombstones. If roots were not provided and config time, this will be identical to procs.
	includedProcs []*Process

	// includedRootProcs is a slice of all Process objects that are roots of the included process tree, sorted by pid.  Includes unpruned tombstones. If
	// explicit roots were not configured, these will be the true roots of the absolute process tree. If explicitRoots were configured with includeAncestors,
	// these will be the roots of the absolute process tree that are ancestors of at least one configured root.
	includedRootProcs []*Process
}

// New creates a new process tree management object and populates it with an initial snapshot
func New(opts ...ConfigOption) (*ProcTree, error) {
	cfg := NewConfig(opts...)

	pt := &ProcTree{
		cfg:               cfg,
		pidMap:            make(map[int]*Process),
		absProcs:          nil,
		absRootProcs:      nil,
		cfgRootProcs:      nil,
		includedProcs:     nil,
		includedRootProcs: nil,
	}

	err := pt.Update(false)
	if err != nil {
		return nil, err
	}

	return pt, nil
}

func (pt *ProcTree) plock() {
	pt.lock.Lock()
}

func (pt *ProcTree) punlock() {
	pt.lock.Unlock()
}

func (pt *ProcTree) lockedSortProcessesByPid(procs []*Process) {
	sort.Slice(procs, func(i, j int) bool { return procs[i].lockedPid() < procs[j].lockedPid() })
}

// SortProcessesByPid sorts a slice of Processes in increasing pid order.
func (pt *ProcTree) SortProcessesByPid(procs []*Process) {
	pt.plock()
	defer pt.punlock()
	pt.lockedSortProcessesByPid(procs)
}

const kthreadPid = 2

func (pt *ProcTree) lockedUpdate(pruneTombstones bool) error {
	fixedRoots := (len(pt.cfg.rootPids) > 0)

	gopsProcs, err := gops.Processes()
	if err != nil {
		return err
	}

	// All existing processes are tombstoned unless they are found again, and child lists are rederived on each update
	for _, proc := range pt.pidMap {
		proc.isTombstone = true
		proc.absChildProcs = []*Process{}
		proc.includedChildProcs = []*Process{}
	}

	// Create all new Processes, and refresh old ones
	for _, gopsProc := range gopsProcs {
		pid := gopsProc.Pid()
		ppid := gopsProc.PPid()
		if pt.cfg.includeKernelThreads || (pid != 2 && ppid != 2) {
			proc, ok := pt.pidMap[pid]
			if ok {
				// refresh existing process
				proc.gopsProcess = gopsProc
				proc.isTombstone = false
			} else {
				// add a new process
				proc = newProcess(pt, gopsProc)
				pt.pidMap[pid] = proc
				proc.isIncluded = !fixedRoots
			}
		}
	}

	if pruneTombstones {
		// Remove all Processes that were not rediscovered by this update
		for pid, proc := range pt.pidMap {
			if proc.isTombstone {
				delete(pt.pidMap, pid)
			}
		}
	}

	if fixedRoots && pt.cfgRootProcs == nil {
		// On the first update, build the list of configured root Processes. This will return
		// an error at New() time if one of the pids is not found.
		pt.cfgRootProcs = make([]*Process, 0, len(pt.cfg.rootPids))
		for _, pid := range pt.cfg.rootPids {
			proc, ok := pt.pidMap[pid]
			if !ok {
				pt.cfgRootProcs = nil
				return fmt.Errorf("Configured root pid %d does not exist", pid)
			}
			pt.cfgRootProcs = append(pt.cfgRootProcs, proc)
		}
	}

	// Fill in the absolute child lists for each process, Build a sorted list of absolute processes,
	// and build a sorted list of absolute root processes
	pt.absProcs = make([]*Process, len(pt.pidMap))
	pt.absRootProcs = []*Process{}
	i := 0
	for _, proc := range pt.pidMap {
		pt.absProcs[i] = proc
		i++
		ppid := proc.gopsProcess.PPid()
		var pproc *Process
		if ppid != 0 {
			var ok bool
			pproc, ok = pt.pidMap[ppid]
			if !ok {
				pproc = nil
			}
		}
		if pproc != nil {
			pproc.absChildProcs = append(pproc.absChildProcs, proc)
			proc.parentProc = pproc
			if proc.origParentProc == nil {
				proc.origParentProc = pproc
			}
		} else {
			pt.absRootProcs = append(pt.absRootProcs, proc)
		}
	}
	pt.lockedSortProcessesByPid(pt.absProcs)
	pt.lockedSortProcessesByPid(pt.absRootProcs)

	// Make sure each Process's child list is sorted in pid order
	for _, proc := range pt.absProcs {
		pt.lockedSortProcessesByPid(proc.absChildProcs)
	}

	if fixedRoots {
		// If we have configured roots, then by default everything is excluded. We will walk the subtree for each
		// root and enable all of the reachable processes
		err = pt.lockedFullWalkFromRoots(pt.cfgRootProcs, func(proc *Process) error {
			if !proc.isIncluded {
				proc.isIncluded = true
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("Unable to compute rooted tree subset: %s", err)
		}
		if pt.cfg.includeRootAncestors {
			// If we are including ancestors then we also need to walk up from each configured root and enable those processes
			for _, root := range pt.cfgRootProcs {
				err = root.lockedWalkFullAncestry(func(proc *Process) error {
					if !proc.isIncluded {
						proc.isIncluded = true
					}
					return nil
				})
				if err != nil {
					return fmt.Errorf("Unable to compute rooted tree subset: %s", err)
				}
			}
		}
	} else {
		// If we have configured roots, then by default everything is included.
		for _, proc := range pt.absProcs {
			proc.isIncluded = true
		}
	}

	// If requested, exclude kernel threads
	if !pt.cfg.includeKernelThreads {
		kProc, ok := pt.pidMap[kthreadPid]
		if ok {
			err = kProc.lockedWalkFullSubtree(func(proc *Process) error {
				proc.isIncluded = false
				return nil
			})
			if err != nil {
				return fmt.Errorf("Unable to compute disable kernel thread subtree: %s", err)
			}
		}

	}

	// Build the list of included processes and included root processes, and fill in included child list for
	// each Process
	pt.includedProcs = make([]*Process, 0, len(pt.absProcs))
	pt.includedRootProcs = []*Process{}
	for _, proc := range pt.absProcs {
		if proc.isIncluded {
			pt.includedProcs = append(pt.includedProcs, proc)
			pproc := proc.parentProc
			if pproc != nil {
				pproc.includedChildProcs = append(pproc.includedChildProcs, proc)
			}
			if pproc == nil || pproc == proc || !pproc.isIncluded {
				pt.includedRootProcs = append(pt.includedRootProcs, proc)
			}

		}
	}

	pt.lockedSortProcessesByPid(pt.includedProcs)
	pt.lockedSortProcessesByPid(pt.includedRootProcs)
	// Make sure each Process's included child list is sorted in pid order
	for _, proc := range pt.absProcs {
		pt.lockedSortProcessesByPid(proc.includedChildProcs)
	}

	return nil
}

// Update refreshes the ProcTree session with a new snapshot view of current processes. Process objects
// from the previous snapshot are preserved, but may become tombstoned.
func (pt *ProcTree) Update(pruneTombstones bool) error {
	pt.plock()
	defer pt.punlock()
	return pt.lockedUpdate(pruneTombstones)
}

// Close implements io.Closer. Shuts down the ProcTree and releases resources
func (pt *ProcTree) Close() error {
	return nil
}

// Processes returns a snapshot of the list of Process objects the tree, sorted in ascending PID order.
// If root pids were provided at configuration time, only processes descended from the provided root
// Processes will be returned.
func (pt *ProcTree) Processes() []*Process {
	pt.plock()
	defer pt.punlock()
	result := make([]*Process, len(pt.includedProcs))
	copy(result, pt.includedProcs)
	return result
}

// Roots returns a snapshot of the list of all included Process objects that are toplevel roots,
// sorted in ascending PID order.
func (pt *ProcTree) Roots() []*Process {
	pt.plock()
	defer pt.punlock()
	result := make([]*Process, len(pt.includedRootProcs))
	copy(result, pt.includedRootProcs)
	return result
}

// PidProcess looks up a Process in the current snapshot by PID. If there
// is no process with the provided PID, (nil, false) is returned.
func (pt *ProcTree) PidProcess(pid int) (proc *Process, ok bool) {
	pt.plock()
	defer pt.punlock()
	proc, ok = pt.pidMap[pid]
	return proc, ok
}

func (pt *ProcTree) lockedFullWalkFromRoots(roots []*Process, h ProcessHandler) error {
	for _, proc := range roots {
		err := proc.lockedWalkFullSubtree(h)
		if err != nil {
			return err
		}
	}
	return nil
}

func (pt *ProcTree) lockedWalkFromRoots(roots []*Process, h ProcessHandler) error {
	for _, proc := range roots {
		err := proc.lockedWalkSubtree(h)
		if err != nil {
			return err
		}
	}
	return nil
}

// WalkFromRoots walks all subtrees starting at this provided root Process objects, invoking
// a handler for each. Roots are walked in provided order; within each root Processes are walked in
// depth-first order with children sorted in pid order. It is the caller's responsibility to ensure
// that no root is a descendant of another; otherwise the handler will be called multiple
// times for the same Process.
func (pt *ProcTree) WalkFromRoots(roots []*Process, h ProcessHandler) error {
	for _, proc := range roots {
		err := proc.WalkSubtree(h)
		if err != nil {
			return err
		}
	}
	return nil
}

func (pt *ProcTree) lockedFullWalk(h ProcessHandler) error {
	return pt.lockedFullWalkFromRoots(pt.absRootProcs, h)
}

func (pt *ProcTree) lockedWalk(h ProcessHandler) error {
	return pt.lockedWalkFromRoots(pt.includedRootProcs, h)
}

// Walk walks all subtrees starting at the configured root Process objects, invoking
// a handler for each. Roots are walked in pid order; within each root Processes are walked in
// depth-first order with children sorted in pid order.
func (pt *ProcTree) Walk(h ProcessHandler) error {
	return pt.WalkFromRoots(pt.Roots(), h)
}
