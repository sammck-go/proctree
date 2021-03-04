/*
Package proctree provides tools for inspecting, monitoring, and manipulating the system process tree.
*/
package proctree

import (
	"os"
	"testing"
)

func TestCurrentProcess(t *testing.T) {
	cfg := NewConfig()

	pt, err := New(WithConfig(cfg))

	if err != nil {
		t.Fatalf("proctree.New() returned error: %s", err)
	}

	myPid := os.Getpid()
	myParentPid := os.Getppid()

	myProc := pt.PidProcess(myPid)
	if myProc == nil {
		t.Errorf("Current process pid %d not found in process tree", myPid)
	} else {
		if myPid != myProc.Pid() {
			t.Errorf("Current process pid %d does not match myProc.Pid() %d", myPid, myProc.Pid())
		}
		if myProc.Executable() != "proctree.test" {
			t.Errorf("myProc executable name \"%s\" is not expected", myProc.Executable())
		}
		myParentProc := pt.PidProcess(myParentPid)
		if myParentProc == nil {
			t.Errorf("Current parent process pid %d not found in process tree", myParentPid)
		} else {
			if myParentPid != myParentProc.Pid() {
				t.Errorf("Current parent process pid %d does not match myParentProc.Pid() %d", myParentPid, myParentProc.Pid())
			}
			if myParentProc != myProc.Parent() {
				t.Error("myParentProc != myProc.Parent")
			}
			found := false
			for _, child := range myParentProc.Children() {
				if child == myProc {
					found = true
					break
				}
			}
			if !found {
				t.Error("myProc not in myParentProc.Children()")
			}
		}
	}

	err = pt.Close()
	if err != nil {
		t.Errorf("pt.Close() returned error: %f", err)
	}
}
