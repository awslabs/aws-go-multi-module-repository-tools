package gomod

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestModuleTreeList(t *testing.T) {
	cases := map[string]struct {
		tree   *ModuleTree
		expect []string
	}{
		"single module": {
			tree: func() *ModuleTree {
				tree := NewModuleTree()
				tree.Insert("a")
				return tree
			}(),
			expect: []string{
				"a",
			},
		},
		"sibling module": {
			tree: func() *ModuleTree {
				tree := NewModuleTree()
				tree.Insert("a")
				tree.Insert("b")
				return tree
			}(),
			expect: []string{
				"a",
				"b",
			},
		},
		"nested module": {
			tree: func() *ModuleTree {
				tree := NewModuleTree()
				tree.Insert("a")
				tree.Insert("a/c")
				tree.Insert("b")
				tree.Insert("b/c")
				return tree
			}(),
			expect: []string{
				"a",
				"a/c",
				"b",
				"b/c",
			},
		},
		"root module": {
			tree: func() *ModuleTree {
				tree := NewModuleTree()
				tree.Insert(".")
				return tree
			}(),
			expect: []string{
				".",
			},
		},
		"root with nested module": {
			tree: func() *ModuleTree {
				tree := NewModuleTree()
				tree.Insert(".")
				tree.Insert("a")
				tree.Insert("a/c")
				tree.Insert("b")
				tree.Insert("b/c")
				return tree
			}(),
			expect: []string{
				".",
				"a",
				"a/c",
				"b",
				"b/c",
			},
		},
		"with root module": {
			tree: func() *ModuleTree {
				tree := NewModuleTree(func(o *ModuleTreeOptions) {
					o.RootPath = "/foo/bar"
				})
				tree.Insert("/foo/bar")
				tree.Insert("/foo/bar/a")
				tree.Insert("/foo/bar/a/c")
				tree.Insert("/foo/bar/b")
				tree.Insert("/foo/bar/b/c")
				return tree
			}(),
			expect: []string{
				".",
				"a",
				"a/c",
				"b",
				"b/c",
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			modules := tt.tree.List()
			var actual []string
			for _, module := range modules {

				actual = append(actual, module.Path())
			}
			if diff := cmp.Diff(tt.expect, actual); diff != "" {
				t.Errorf("expect modules match\n%s", diff)
			}
		})
	}
}

func TestModuleTreeIterator(t *testing.T) {
	cases := map[string]struct {
		tree   *ModuleTree
		expect []string
	}{
		"single module": {
			tree: func() *ModuleTree {
				tree := NewModuleTree()
				tree.Insert("a")
				return tree
			}(),
			expect: []string{
				"a",
			},
		},
		"sibling module": {
			tree: func() *ModuleTree {
				tree := NewModuleTree()
				tree.Insert("a")
				tree.Insert("b")
				return tree
			}(),
			expect: []string{
				"a",
				"b",
			},
		},
		"nested module": {
			tree: func() *ModuleTree {
				tree := NewModuleTree()
				tree.Insert("a")
				tree.Insert("a/c")
				tree.Insert("b")
				tree.Insert("b/c")
				return tree
			}(),
			expect: []string{
				"a",
				"a/c",
				"b",
				"b/c",
			},
		},
		"root module": {
			tree: func() *ModuleTree {
				tree := NewModuleTree()
				tree.Insert(".")
				return tree
			}(),
			expect: []string{
				".",
			},
		},
		"root with nested module": {
			tree: func() *ModuleTree {
				tree := NewModuleTree()
				tree.Insert(".")
				tree.Insert("a")
				tree.Insert("a/c")
				tree.Insert("b")
				tree.Insert("b/c")
				return tree
			}(),
			expect: []string{
				".",
				"a",
				"a/c",
				"b",
				"b/c",
			},
		},
		"with root module": {
			tree: func() *ModuleTree {
				tree := NewModuleTree(func(o *ModuleTreeOptions) {
					o.RootPath = "/foo/bar"
				})
				tree.Insert("/foo/bar")
				tree.Insert("/foo/bar/a")
				tree.Insert("/foo/bar/a/c")
				tree.Insert("/foo/bar/b")
				tree.Insert("/foo/bar/b/c")
				return tree
			}(),
			expect: []string{
				".",
				"a",
				"a/c",
				"b",
				"b/c",
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			var actual []string
			for it := tt.tree.Iterator(); ; {
				module := it.Next()
				if module == nil {
					break
				}

				actual = append(actual, module.Path())
			}
			if diff := cmp.Diff(tt.expect, actual); diff != "" {
				t.Errorf("expect modules match\n%s", diff)
			}
		})
	}
}

func TestModuleTreeParentOf(t *testing.T) {
	cases := map[string]struct {
		module *ModuleTreeNode
		path   string
		expect bool
	}{
		"child": {
			module: &ModuleTreeNode{
				absPath: "a",
				relPath: "a",
				subModules: []*ModuleTreeNode{
					{absPath: "a/b", relPath: "a/b"},
					{absPath: "a/f/g", relPath: "a/f/g"},
				},
			},
			path:   "a/foo",
			expect: true,
		},
		"match submodule": {
			module: &ModuleTreeNode{
				absPath: "a",
				relPath: "a",
				subModules: []*ModuleTreeNode{
					{absPath: "a/b", relPath: "a/b"},
					{absPath: "a/f/g", relPath: "a/f/g"},
				},
			},
			path:   "a/b",
			expect: false,
		},
		"submodule child": {
			module: &ModuleTreeNode{
				absPath: "a",
				relPath: "a",
				subModules: []*ModuleTreeNode{
					{absPath: "a/b", relPath: "a/b"},
					{absPath: "a/f/g", relPath: "a/f/g"},
				},
			},
			path:   "a/b/a",
			expect: false,
		},
		"submodule child root path": {
			module: &ModuleTreeNode{
				absPath: "/foo/bar/a",
				relPath: "a",
				subModules: []*ModuleTreeNode{
					{absPath: "/foo/bar/a/b", relPath: "a/b"},
					{absPath: "/foo/bar/a/f/g", relPath: "a/f/g"},
				},
			},
			path:   "a/b/a",
			expect: false,
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			actual := tt.module.ParentOf(tt.path)
			if e, a := tt.expect, actual; e != a {
				t.Errorf("expect %v, got %v", e, a)
			}
		})
	}
}

func TestModuleTreeInsert(t *testing.T) {
	cases := map[string]struct {
		modules    []string
		expectTree *ModuleTree
	}{
		"basic": {
			modules: []string{"a/f/g", "a", "a/b", "c", "e/f/g"},
			expectTree: &ModuleTree{
				options: ModuleTreeOptions{},
				subModules: []*ModuleTreeNode{
					{
						absPath: "a",
						relPath: "a",
						subModules: []*ModuleTreeNode{
							{absPath: "a/b", relPath: "a/b"},
							{absPath: "a/f/g", relPath: "a/f/g"},
						},
					},
					{absPath: "c", relPath: "c"},
					{absPath: "e/f/g", relPath: "e/f/g"},
				},
			},
		},
		"with root": {
			modules: []string{"/foo/bar/a/f/g", "/foo/bar/a", "/foo/bar/a/b", "/foo/bar/c", "/foo/bar/e/f/g"},
			expectTree: &ModuleTree{
				options: ModuleTreeOptions{RootPath: "/foo/bar"},
				subModules: []*ModuleTreeNode{
					{
						absPath: "/foo/bar/a",
						relPath: "a",
						subModules: []*ModuleTreeNode{
							{absPath: "/foo/bar/a/b", relPath: "a/b"},
							{absPath: "/foo/bar/a/f/g", relPath: "a/f/g"},
						},
					},
					{absPath: "/foo/bar/c", relPath: "c"},
					{absPath: "/foo/bar/e/f/g", relPath: "e/f/g"},
				},
			},
		},
		"nested reorder": {
			modules: []string{
				".",
				"service/s3/internal/configtest",
				"service/s3",
			},
			expectTree: &ModuleTree{
				subModules: []*ModuleTreeNode{
					{
						absPath: ".",
						relPath: ".",
						subModules: []*ModuleTreeNode{
							{
								absPath: "service/s3", relPath: "service/s3",
								subModules: []*ModuleTreeNode{
									{
										absPath: "service/s3/internal/configtest",
										relPath: "service/s3/internal/configtest",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			tree := NewModuleTree(func(o *ModuleTreeOptions) {
				*o = c.expectTree.options
			})

			for _, modulePath := range c.modules {
				m, err := tree.Insert(modulePath)
				if err != nil {
					t.Errorf("failed to insert, %v", err)
				}
				if m == nil {
					t.Errorf("insert not return module, got nil")
				}
			}

			if diff := cmp.Diff(c.expectTree, tree, moduleTreeCmpOptions); diff != "" {
				t.Errorf("expect trees to match\n%s", diff)
			}
		})
	}
}

func TestModuleTreeSearch(t *testing.T) {
	cases := map[string]struct {
		searchPath string
		tree       *ModuleTree
		expectNode *ModuleTreeNode
	}{
		"submatch": {
			searchPath: "a/b/123",
			tree: &ModuleTree{
				options: ModuleTreeOptions{},
				subModules: []*ModuleTreeNode{
					{
						absPath: "a",
						relPath: "a",
						subModules: []*ModuleTreeNode{
							{absPath: "a/b", relPath: "a/b"},
						},
					},
					{absPath: "c", relPath: "c"},
					{absPath: "e/f/g", relPath: "e/f/g"},
				},
			},
			expectNode: &ModuleTreeNode{
				absPath: "a/b", relPath: "a/b",
			},
		},
		"nested module": {
			searchPath: "service/c/e/f",
			tree: &ModuleTree{
				subModules: []*ModuleTreeNode{
					{
						absPath: ".",
						relPath: ".",
						subModules: []*ModuleTreeNode{
							{
								absPath: "a", relPath: "a",
							},
							{
								absPath: "service/a", relPath: "service/a",
							},
							{
								absPath: "service/c", relPath: "service/c",
								subModules: []*ModuleTreeNode{
									{
										absPath: "service/c/e/f",
										relPath: "service/c/e/f",
									},
								},
							},
						},
					},
				},
			},
			expectNode: &ModuleTreeNode{
				absPath: "service/c/e/f",
				relPath: "service/c/e/f",
			},
		},
		"not found": {
			searchPath: "b/123",
			tree: &ModuleTree{
				options: ModuleTreeOptions{},
				subModules: []*ModuleTreeNode{
					{
						absPath: "a",
						relPath: "a",
						subModules: []*ModuleTreeNode{
							{absPath: "a/b", relPath: "a/b"},
						},
					},
					{absPath: "c", relPath: "c"},
					{absPath: "e/f/g", relPath: "e/f/g"},
				},
			},
			expectNode: nil,
		},
		"with root": {
			searchPath: "a/f/g/foo",
			tree: &ModuleTree{
				options: ModuleTreeOptions{RootPath: "/foo/bar"},
				subModules: []*ModuleTreeNode{
					{
						absPath: "/foo/bar/a",
						relPath: "a",
						subModules: []*ModuleTreeNode{
							{absPath: "/foo/bar/a/b", relPath: "a/b"},
							{absPath: "/foo/bar/a/f/g", relPath: "a/f/g"},
						},
					},
					{absPath: "/foo/bar/c", relPath: "c"},
					{absPath: "/foo/bar/e/f/g", relPath: "e/f/g"},
				},
			},
			expectNode: &ModuleTreeNode{
				absPath: "/foo/bar/a/f/g", relPath: "a/f/g",
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			node := c.tree.Search(c.searchPath)
			if diff := cmp.Diff(c.expectNode, node, moduleTreeCmpOptions); diff != "" {
				t.Errorf("expect trees to match\n%s", diff)
			}
		})
	}
}
func TestModuleTreeGet(t *testing.T) {
	cases := map[string]struct {
		searchPath string
		tree       *ModuleTree
		expectNode *ModuleTreeNode
	}{
		"submatch": {
			searchPath: "a/b",
			tree: &ModuleTree{
				options: ModuleTreeOptions{},
				subModules: []*ModuleTreeNode{
					{
						absPath: "a",
						relPath: "a",
						subModules: []*ModuleTreeNode{
							{absPath: "a/b", relPath: "a/b"},
						},
					},
					{absPath: "c", relPath: "c"},
					{absPath: "e/f/g", relPath: "e/f/g"},
				},
			},
			expectNode: &ModuleTreeNode{
				absPath: "a/b", relPath: "a/b",
			},
		},
		"nested module": {
			searchPath: "service/c/e/f",
			tree: &ModuleTree{
				subModules: []*ModuleTreeNode{
					{
						absPath: ".",
						relPath: ".",
						subModules: []*ModuleTreeNode{
							{
								absPath: "a", relPath: "a",
							},
							{
								absPath: "service/a", relPath: "service/a",
							},
							{
								absPath: "service/c", relPath: "service/c",
								subModules: []*ModuleTreeNode{
									{
										absPath: "service/c/e/f",
										relPath: "service/c/e/f",
									},
								},
							},
						},
					},
				},
			},
			expectNode: &ModuleTreeNode{
				absPath: "service/c/e/f",
				relPath: "service/c/e/f",
			},
		},
		"nested module with root": {
			searchPath: "service/c/e/f",
			tree: &ModuleTree{
				options: ModuleTreeOptions{RootPath: "/foo/bar"},
				subModules: []*ModuleTreeNode{
					{
						absPath: "/foo/bar",
						relPath: ".",
						subModules: []*ModuleTreeNode{
							{
								absPath: "/foo/bar/a", relPath: "a",
							},
							{
								absPath: "/foo/bar/service/a", relPath: "service/a",
							},
							{
								absPath: "/foo/bar/service/c", relPath: "service/c",
								subModules: []*ModuleTreeNode{
									{
										absPath: "/foo/bar/service/c/e/f",
										relPath: "service/c/e/f",
									},
								},
							},
						},
					},
				},
			},
			expectNode: &ModuleTreeNode{
				absPath: "/foo/bar/service/c/e/f",
				relPath: "service/c/e/f",
			},
		},
		"not found": {
			searchPath: "b/123",
			tree: &ModuleTree{
				options: ModuleTreeOptions{},
				subModules: []*ModuleTreeNode{
					{
						absPath: "a",
						relPath: "a",
						subModules: []*ModuleTreeNode{
							{absPath: "a/b", relPath: "a/b"},
						},
					},
					{absPath: "c", relPath: "c"},
					{absPath: "e/f/g", relPath: "e/f/g"},
				},
			},
			expectNode: nil,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			node := c.tree.Get(c.searchPath)
			if diff := cmp.Diff(c.expectNode, node, moduleTreeCmpOptions); diff != "" {
				t.Errorf("expect trees to match\n%s", diff)
			}
		})
	}
}

var moduleTreeType = reflect.TypeOf((*ModuleTree)(nil))
var moduleTreeNodeType = reflect.TypeOf((*ModuleTreeNode)(nil))

var moduleTreeCmpOptions = cmp.Options{
	cmp.Exporter(func(t reflect.Type) bool {
		switch t {
		case moduleTreeType, moduleTreeType.Elem(),
			moduleTreeNodeType, moduleTreeNodeType.Elem():
			return true
		default:
			return false
		}
	}),
}
