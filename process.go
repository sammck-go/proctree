package proctree

import (
	gops "github.com/mitchellh/go-ps"
)

// Process repressents an abstraction of a single process within a ProcTree session. It
// maintains its identity within a single session.
type Process struct {
	pt                 *ProcTree
	gopsProcess        gops.Process
	isTombstone        bool
	parentProc         *Process
	origParentProc     *Process
	absChildProcs      []*Process
	includedChildProcs []*Process
	isIncluded         bool
}

func newProcess(pt *ProcTree, gopsProcess gops.Process) *Process {
	p := &Process{
		pt:                 pt,
		gopsProcess:        gopsProcess,
		isTombstone:        false,
		origParentProc:     nil,
		parentProc:         nil,
		absChildProcs:      nil,
		includedChildProcs: nil,
		isIncluded:         true,
	}

	return p
}

func (p *Process) plock() {
	p.pt.plock()
}

func (p *Process) punlock() {
	p.pt.punlock()
}

func (p *Process) lockedPid() int {
	return p.gopsProcess.Pid()
}

// Pid returns the pid of a Process
func (p *Process) Pid() int {
	p.plock()
	defer p.punlock()
	return p.lockedPid()
}

func (p *Process) lockedExecutable() string {
	return p.gopsProcess.Executable()
}

// Executable returns the executable name associated with a process, without the directory path
func (p *Process) Executable() string {
	p.plock()
	defer p.punlock()
	return p.lockedExecutable()
}

func (p *Process) lockedParent() *Process {
	if p.parentProc == nil || p.parentProc == p || !p.parentProc.isIncluded {
		return nil
	}
	return p.parentProc
}

// Parent returns the Process that is the parent of this Process, or nil if the Process does not have
// a parent that is configured for inclusion.
func (p *Process) Parent() *Process {
	p.plock()
	defer p.punlock()
	return p.lockedParent()
}

func (p *Process) lockedOrigParent() *Process {
	return p.origParentProc
}

// OrigParent returns the Process that was the parent before this process became detached (reattached
// to pid 1), if it is known. Otherwise, returns the parent at the time the session was initialized, which
// may be null or pid 1.
func (p *Process) OrigParent() *Process {
	p.plock()
	defer p.punlock()
	return p.lockedOrigParent()
}

// lockedChildren returns an immutable snapshot slice of Processes known to be a child of the Process. Only
// children that meet configured filter conditions (e.g., are in configured root subtrees or ancestor paths) are included.
// This will include tombstoned children that have been added since the last time tombstones were pruned.
func (p *Process) lockedChildren() []*Process {
	return p.includedChildProcs
}

// Children returns an immutable snapshot slice of Processes known to be a child of the Process. Only
// children that meet configured filter conditions (e.g., are in configured root subtrees or ancestor paths) are included.
// This will include tombstoned children that have been added since the last time tombstones were pruned.
func (p *Process) Children() []*Process {
	p.plock()
	defer p.punlock()
	result := make([]*Process, len(p.includedChildProcs))
	for i, child := range p.includedChildProcs {
		result[i] = child
	}
	return result
}

func (p *Process) lockedIsDescendantOf(ancestor *Process) bool {
	parent := p.parentProc
	return parent != nil && ancestor != nil && (parent == ancestor || parent.lockedIsDescendantOf(ancestor))
}

// IsDescendantOf returns true if the Process is a known descendant of a provided ancestor Process
func (p *Process) IsDescendantOf(ancestor *Process) bool {
	p.plock()
	defer p.punlock()
	return p.lockedIsDescendantOf(ancestor)
}

func (p *Process) lockedIsAncestorOf(descendant *Process) bool {
	return descendant != nil && descendant.lockedIsDescendantOf(p)
}

// IsAncestorOf returns true if the Process is a known ancestor of a provided descendant Process
func (p *Process) IsAncestorOf(descendant *Process) bool {
	p.plock()
	defer p.punlock()
	return p.lockedIsAncestorOf(descendant)
}

// ProcessHandler represents a function that is called back to act on a process. Used for process
// tree walking operations.
type ProcessHandler func(*Process) error

func (p *Process) lockedWalkFullSubtree(h ProcessHandler) error {
	err := h(p)
	if err != nil {
		return err
	}
	for _, child := range p.absChildProcs {
		err = child.lockedWalkFullSubtree(h)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Process) lockedWalkSubtree(h ProcessHandler) error {
	if p.isIncluded {
		err := h(p)
		if err != nil {
			return err
		}
		for _, child := range p.lockedChildren() {
			err = child.lockedWalkSubtree(h)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// WalkSubtree walks an entire subtree starting at this process as the root, invoking
// a handler for each. Processes are walked in depth-first order with children
// sorted in pid order. Only subtrees enabled by configuration are included
func (p *Process) WalkSubtree(h ProcessHandler) error {
	if p.isIncluded {
		err := h(p)
		if err != nil {
			return err
		}
		for _, child := range p.Children() {
			err = child.WalkSubtree(h)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *Process) lockedWalkFullAncestry(h ProcessHandler) error {
	err := h(p)
	if err != nil {
		return err
	}
	parent := p.parentProc
	if parent != nil && parent != p {
		err = parent.lockedWalkFullAncestry(h)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Process) lockedWalkAncestry(h ProcessHandler) error {
	var err error
	if p.isIncluded {
		err = h(p)
		if err != nil {
			return err
		}
	}
	parent := p.parentProc
	if parent != nil && parent != p {
		err = parent.lockedWalkAncestry(h)
		if err != nil {
			return err
		}
	}
	return nil
}

// WalkAncestry walks the ancestry list starting at this process, up to the root, invoking
// a handler for each. Only Processes enabled by configuration are included
func (p *Process) WalkAncestry(h ProcessHandler) error {
	var err error
	p.plock()
	isIncluded := p.isIncluded
	parent := p.parentProc
	p.punlock()
	if isIncluded {
		err = h(p)
		if err != nil {
			return err
		}
	}
	if parent != nil && parent != p {
		err = parent.WalkAncestry(h)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Process) lockedDepth() int {
	result := 0
	proc := p
	for proc.parentProc != nil && proc.parentProc != proc && proc.parentProc.isIncluded {
		proc = proc.parentProc
		result++
	}
	return result
}

// Depth computes the depth of this process in the process tree. 0 is returned for root processes; 1 for their children; etc.
func (p *Process) Depth() int {
	p.plock()
	defer p.punlock()
	return p.lockedDepth()
}
