package process

import "github.com/iamgilwell/aura/internal/monitor"

// DependencyTree maps parent PIDs to their children.
type DependencyTree struct {
	children map[int][]int
	parents  map[int]int
}

// BuildDependencyTree creates a process dependency tree.
func BuildDependencyTree(procs []*monitor.ProcessInfo) *DependencyTree {
	dt := &DependencyTree{
		children: make(map[int][]int),
		parents:  make(map[int]int),
	}
	for _, p := range procs {
		dt.parents[p.PID] = p.PPid
		dt.children[p.PPid] = append(dt.children[p.PPid], p.PID)
	}
	return dt
}

// ChildrenOf returns all direct children of a PID.
func (dt *DependencyTree) ChildrenOf(pid int) []int {
	return dt.children[pid]
}

// AllDescendants returns all descendants of a PID recursively.
func (dt *DependencyTree) AllDescendants(pid int) []int {
	var result []int
	queue := dt.children[pid]
	visited := map[int]bool{pid: true}

	for len(queue) > 0 {
		child := queue[0]
		queue = queue[1:]
		if visited[child] {
			continue
		}
		visited[child] = true
		result = append(result, child)
		queue = append(queue, dt.children[child]...)
	}
	return result
}

// SafeTerminationOrder returns PIDs in order for safe termination
// (children first, then parents).
func (dt *DependencyTree) SafeTerminationOrder(pid int) []int {
	descendants := dt.AllDescendants(pid)
	// Reverse order: deepest children first
	result := make([]int, len(descendants)+1)
	for i, d := range descendants {
		result[len(descendants)-1-i] = d
	}
	result[len(descendants)] = pid
	return result
}

// ParentOf returns the parent PID.
func (dt *DependencyTree) ParentOf(pid int) int {
	return dt.parents[pid]
}

// WouldOrphan returns PIDs that would become orphaned if pid is terminated.
func (dt *DependencyTree) WouldOrphan(pid int) []int {
	return dt.children[pid]
}
