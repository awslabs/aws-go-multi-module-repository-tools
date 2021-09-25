package gomod

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// ModuleTree provides a tree for organizing Go modules with a path tree
// structure.
type ModuleTree struct {
	options    ModuleTreeOptions
	subModules []*ModuleTreeNode
}

// ModuleTreeOption provides the options for the ModuleTree's behavior.
type ModuleTreeOptions struct {
	// Sets the root directory path that all modules must be nested within. Any
	// module attempted to be inserted into the tree that is outside of this
	// root path will cause Insert to return an error.
	//
	// If set, ModuleTreeNode.PathRel will return the relative path of the
	// module from this root path.
	RootPath string
}

// NewModuleTree returns a initialized tree container for modules.
//
// List of sub modules must be and sorted ascending list of unique paths.
func NewModuleTree(optFns ...func(o *ModuleTreeOptions)) *ModuleTree {
	var options ModuleTreeOptions
	for _, fn := range optFns {
		fn(&options)
	}
	return &ModuleTree{
		options: options,
	}
}

func (t *ModuleTree) InsertRel(relModulePath string, attributes ...string) (*ModuleTreeNode, error) {
	return t.Insert(filepath.Join(t.options.RootPath, relModulePath), attributes...)
}

// Insert adds a new module to the tree. Nesting the new module within the
// parent modules if any.
func (t *ModuleTree) Insert(modulePath string, attributes ...string) (newNode *ModuleTreeNode, err error) {
	moduleRelPath := modulePath
	if t.options.RootPath != "" {
		moduleRelPath, err = filepath.Rel(t.options.RootPath, modulePath)
		if err != nil {
			return nil, fmt.Errorf("module %q is not nested within %q, %w",
				modulePath, t.options.RootPath, err)
		}
	}

	if m := t.Get(moduleRelPath); m != nil {
		return nil, fmt.Errorf("module already exists with relative path, %v, %v",
			modulePath, moduleRelPath)
	}

	// search tree for nodes that have the prefix of this node, when they are
	// found walk down to the next layer to find the next node with a more
	// specific prefix.
	nodes := &t.subModules
	for {
		var nextNodes *[]*ModuleTreeNode
		for _, m := range *nodes {
			if m.AncestorOf(moduleRelPath) {
				nextNodes = &m.subModules
				break
			}
		}

		// Walk the module tree until there is no more direct ancestors. Then
		// insert the module at that layer. If the new module is it self an
		// ancestor of some other nodes at this layer, re-parent the children
		// nodes.
		if nextNodes == nil {
			// Copy attributes to ensure list cannot be mutated outside of the node.
			if len(attributes) != 0 {
				attributes = append([]string{}, attributes...)
			}

			newNode = &ModuleTreeNode{
				absPath:    modulePath,
				relPath:    moduleRelPath,
				attributes: attributes,
			}

			// Before adding the new node to the parent, check if there are any
			// existing sub modules of this parent have this new node as their
			// parent.
			for i := 0; i < len(*nodes); i++ {
				if !newNode.AncestorOf((*nodes)[i].Path()) {
					continue
				}

				newNode.subModules = append(newNode.subModules, (*nodes)[i])
				sort.Sort(sortableModuleTreeNodes(newNode.subModules))
				*nodes = cutSubModule(*nodes, i)
				i--
			}

			*nodes = append(*nodes, newNode)
			sort.Sort(sortableModuleTreeNodes(*nodes))
			return newNode, nil
		}

		nodes = nextNodes
	}
}

func cutSubModule(nodes []*ModuleTreeNode, i int) []*ModuleTreeNode {
	j := i + 1
	copy(nodes[i:], nodes[j:])
	for k, n := len(nodes)-j+i, len(nodes); k < n; k++ {
		nodes[k] = nil // or the zero values of T
	}

	return nodes[:len(nodes)-j+i]
}

// Search returns the nearest module ancestor for the path.
//
// Uses relative path of module from the tree's root. If the tree does not have
// a root specified the value will be the absolute path of the module when it
// was inserted into the tree.
func (t *ModuleTree) Search(path string) *ModuleTreeNode {
	return searchModuleTreeNodes(path, t.subModules)
}

// Get returns if the tree contains a module with the relative path.
//
// If no tree root is specified, path will search for exact path the node was
// created with.
func (t *ModuleTree) Get(path string) *ModuleTreeNode {
	if node := searchModuleTreeNodes(path, t.subModules); node != nil && node.Path() == path {
		return node
	}
	return nil
}

// Iterator returns an iterator for walking the tree.
func (t *ModuleTree) Iterator() *ModuleTreeIterator {
	return newModuleTreeIterator(t)
}

// List returns a list of all nodes in the tree in sorted order.
func (t *ModuleTree) List() (list []*ModuleTreeNode) {
	for _, n := range t.subModules {
		list = append(list, n)
		list = append(list, n.List()...)
	}

	return list
}

// ListPaths returns a list of all node paths in the tree in sorted order.
//
// Uses relative path of module from the tree's root. If the tree does not have
// a root specified the value will be the absolute path of the module when it
// was inserted into the tree.
func (t *ModuleTree) ListPaths() (list []string) {
	for _, n := range t.subModules {
		list = append(list, n.Path())
		list = append(list, n.ListPaths()...)
	}

	return list
}

// ModuleTreeNode provides the module node of a ModuleTree.
type ModuleTreeNode struct {
	absPath    string
	relPath    string
	subModules []*ModuleTreeNode
	attributes []string
}

// HasAttribute returns if the node has the attribute requested.
func (n *ModuleTreeNode) HasAttribute(attribute string) bool {
	for _, attrib := range n.attributes {
		if attrib == attribute {
			return true
		}
	}
	return false
}

// Attributes returns a list of the attributes associated with this node.
func (n *ModuleTreeNode) Attributes() []string {
	return append([]string{}, n.attributes...)
}

// Path returns the module path.
//
// Uses relative path of module from the tree's root. If the tree does not have
// a root specified the value will be the absolute path of the module when it
// was inserted into the tree.
func (n *ModuleTreeNode) Path() string {
	return n.relPath
}

// AbsPath returns the absolute path of the module when it was inserted into
// the tree.
func (n *ModuleTreeNode) AbsPath() string {
	return n.absPath
}

// AncestorOf returns true if this module is an ancestor of the path. Not
// matching paths that are siblings of the module with common name prefix.
//
// Uses relative path of module from the tree's root. If the tree does not have
// a root specified the value will be the absolute path of the module when it
// was inserted into the tree.
func (n *ModuleTreeNode) AncestorOf(path string) bool {
	if n.Path() == "." || n.Path() == path {
		return true
	}

	modulePath := n.Path() + "/"
	return strings.HasPrefix(path, modulePath)
}

// ParentOf returns true if this module is a direct parent of the path
// specified, and no other sub module is also an ancestor of it. If the path is
// a sub module of this module, ParentOf will return false. Use Search to find
// sub modules.
//
// Uses relative path of module from the tree's root. If the tree does not have
// a root specified the value will be the absolute path of the module when it
// was inserted into the tree.
func (n *ModuleTreeNode) ParentOf(path string) bool {
	if !n.AncestorOf(path) {
		return false
	}
	return n.Search(path) == nil
}

// Search returns the module that is the closet ancestor of the path. Returns
// nil if no ancestor found.
//
// Uses relative path of module from the tree's root. If the tree does not have
// a root specified the value will be the absolute path of the module when it
// was inserted into the tree.
func (n *ModuleTreeNode) Search(path string) *ModuleTreeNode {
	return searchModuleTreeNodes(path, n.subModules)
}

// Get returns the module that is the closet ancestor of the path. Returns
// nil if the module does not exist exactly.
//
// Uses relative path of module from the tree's root. If the tree does not have
// a root specified the value will be the absolute path of the module when it
// was inserted into the tree.
func (n *ModuleTreeNode) Get(path string) *ModuleTreeNode {
	if n.Path() == path {
		return n
	}

	if node := searchModuleTreeNodes(path, n.subModules); node != nil && node.Path() == path {
		return node
	}
	return nil
}

// Iterator returns an depth first iterator for the tree starting with this
// node as the root of the tree.
func (n *ModuleTreeNode) Iterator() *ModuleTreeIterator {
	return newModuleTreeNodeIterator(n)
}

// List returns a depth first list of all modules starting at this node.
func (n *ModuleTreeNode) List() (list []*ModuleTreeNode) {
	for it := n.Iterator(); ; {
		node := it.Next()
		if node == nil {
			break
		}
		list = append(list, node)
	}

	return list
}

// ListPaths returns a list of all module paths under this node.
//
// Uses relative path of module from the tree's root. If the tree does not have
// a root specified the value will be the absolute path of the module when it
// was inserted into the tree.
func (n *ModuleTreeNode) ListPaths() (list []string) {
	for it := n.Iterator(); ; {
		node := it.Next()
		if node == nil {
			break
		}
		list = append(list, node.Path())
	}

	return list
}

func searchModuleTreeNodes(path string, nodes []*ModuleTreeNode) *ModuleTreeNode {
	var parent *ModuleTreeNode

	for {
		var nextNodes []*ModuleTreeNode
		for _, m := range nodes {
			if m.AncestorOf(path) {
				parent = m
				nextNodes = m.subModules
				break
			}
		}

		if nextNodes == nil {
			return parent
		}

		nodes = nextNodes
	}
}

type sortableModuleTreeNodes []*ModuleTreeNode

func (ns sortableModuleTreeNodes) Len() int { return len(ns) }
func (ns sortableModuleTreeNodes) Less(i, j int) bool {
	return ns[i].relPath < ns[j].relPath
}
func (ns sortableModuleTreeNodes) Swap(i, j int) {
	ns[i], ns[j] = ns[j], ns[i]
}

// moduleTreeNodeStack provides a stack implementation for tree nodes with
// reverse push ordering. This allows lists of nodes to be pushed, but their
// depth first search order maintained in the stack.
type moduleTreeNodeStack struct {
	stack []*ModuleTreeNode
}

func newModuleTreeNodeStack() *moduleTreeNodeStack {
	return &moduleTreeNodeStack{}
}
func (s *moduleTreeNodeStack) PushReverse(nodes ...*ModuleTreeNode) {
	if cap(s.stack) < len(s.stack)+len(nodes) {
		newStack := make([]*ModuleTreeNode, 0, len(nodes)+cap(s.stack))
		newStack = append(newStack, s.stack...)
		s.stack = newStack
	}
	for i := len(nodes) - 1; i >= 0; i-- {
		s.stack = append(s.stack, nodes[i])
	}
}
func (s *moduleTreeNodeStack) Pop() (n *ModuleTreeNode) {
	if len(s.stack) == 0 {
		return nil
	}

	n, s.stack = s.stack[len(s.stack)-1], s.stack[:len(s.stack)-1]
	return n
}

// ModuleTreeIterator provides an iterator for walking the module nodes in the tree.
type ModuleTreeIterator struct {
	stack *moduleTreeNodeStack
}

// Returns an iterator for the sub modules of the tree.
func newModuleTreeIterator(tree *ModuleTree) *ModuleTreeIterator {
	return newModuleTreeNodesIterator(tree.subModules)
}

// Returns an iterator for the sub modules of this node. Does not include the
// node provided it self.
func newModuleTreeNodeIterator(node *ModuleTreeNode) *ModuleTreeIterator {
	return newModuleTreeNodesIterator(node.subModules)
}

func newModuleTreeNodesIterator(nodes []*ModuleTreeNode) *ModuleTreeIterator {
	stack := newModuleTreeNodeStack()
	stack.PushReverse(nodes...)

	return &ModuleTreeIterator{
		stack: stack,
	}
}

// Next returns the next node in the tree. If there are no more nodes, nil will
// be returned.
func (it *ModuleTreeIterator) Next() *ModuleTreeNode {
	next := it.stack.Pop()
	if next == nil {
		return next
	}

	it.stack.PushReverse(next.subModules...)
	return next
}
