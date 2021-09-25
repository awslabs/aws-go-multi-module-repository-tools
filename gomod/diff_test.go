package gomod

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFilterModuleFiles(t *testing.T) {
	tests := map[string]struct {
		module     *ModuleTreeNode
		submodules []string
		changes    []string
		expect     []string
	}{
		"no submodules": {
			module: &ModuleTreeNode{
				absPath: ".", relPath: ".",
			},
			changes: []string{
				"sub3/foo.go",
				"sub2/bar.go",
				"sub1/baz.go",
				"foo.go",
			},
			expect: []string{
				"foo.go",
				"sub1/baz.go",
				"sub2/bar.go",
				"sub3/foo.go",
			},
		},
		"no submodules, no go changes": {
			module: &ModuleTreeNode{
				absPath: ".", relPath: ".",
			},
			changes: []string{
				"foo.java",
			},
		},
		"go.mod considered": {
			module: &ModuleTreeNode{
				absPath: ".", relPath: ".",
			},
			changes: []string{
				"go.mod",
			},
			expect: []string{
				"go.mod",
			},
		},
		"repo root with submodules": {
			module: &ModuleTreeNode{
				absPath: ".", relPath: ".",
				subModules: []*ModuleTreeNode{
					{absPath: "sub1", relPath: "sub1"},
					{absPath: "sub2", relPath: "sub2"},
				},
			},
			changes: []string{
				"sub3/foo.go",
				"sub2/bar.go",
				"sub1/baz.go",
				"foo.go",
			},
			expect: []string{
				"foo.go",
				"sub3/foo.go",
			},
		},
		"submodule directory, no submodules, no changes": {
			module: &ModuleTreeNode{
				absPath: "sub1", relPath: "sub1",
			},
			changes: []string{
				"sub3/foo.go",
				"sub2/bar.go",
				"foo.go",
			},
		},
		"submodule directory, no submodules, changes": {
			module: &ModuleTreeNode{
				absPath: "sub1", relPath: "sub1",
			},
			changes: []string{
				"sub3/foo.go",
				"sub2/bar.go",
				"sub1/bar.go",
				"foo.go",
			},
			expect: []string{
				"sub1/bar.go",
			},
		},
		"submodule directory, submodules, no changes": {
			module: &ModuleTreeNode{
				absPath: "sub1", relPath: "sub1",
				subModules: []*ModuleTreeNode{
					{absPath: "sub1/subsub1", relPath: "sub1/subsub1"},
					{absPath: "sub1/subsub2", relPath: "sub1/subsub2"},
				},
			},
			changes: []string{
				"sub3/foo.go",
				"sub2/bar.go",
				"sub1/subsub1/foo.go",
				"sub1/subsub1/bar.go",
				"sub1/subsub2/bar.go",
				"foo.go",
			},
		},
		"submodule directory, submodules, changes": {
			module: &ModuleTreeNode{
				absPath: "sub1", relPath: "sub1",
				subModules: []*ModuleTreeNode{
					{absPath: "sub1/subsub1", relPath: "sub1/subsub1"},
					{absPath: "sub1/subsub2", relPath: "sub1/subsub2"},
				},
			},
			changes: []string{
				"sub3/foo.go",
				"sub2/bar.go",
				"sub1/subsub1/foo.go",
				"sub1/subsub1/bar.go",
				"sub1/subsub2/bar.go",
				"sub1/notsub/foo.go",
				"foo.go",
			},
			expect: []string{
				"sub1/notsub/foo.go",
			},
		},
		"module with no changes, but shares common prefix with a changed module": {
			module: &ModuleTreeNode{
				absPath: "foobar", relPath: "foobar",
				subModules: []*ModuleTreeNode{
					{absPath: "foobar/sub1", relPath: "foobar/sub1"},
				},
			},
			changes: []string{
				"foobarbaz/baz.go",
				"foobar/sub1/sub1.go",
			},
		},
		"tombstone submodule": {
			module: func() *ModuleTreeNode {
				tree := NewModuleTree()
				tree.Insert(".")
				tree.Insert("a", "tombstone")
				return tree.Get(".")
			}(),
			changes: []string{
				"a/go.mod",
			},
		},
		"tombstone module": {
			module: func() *ModuleTreeNode {
				tree := NewModuleTree()
				tree.Insert(".")
				tree.Insert("a", "tombstone")
				return tree.Get("a")
			}(),
			changes: []string{
				"a/go.mod",
			},
			expect: []string{
				"a/go.mod",
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			actual, err := FilterModuleFiles(tt.module, tt.changes)
			if err != nil {
				t.Errorf("expect no error, got %v", err)
				return
			}

			if diff := cmp.Diff(tt.expect, actual); diff != "" {
				t.Errorf("expect changes match\n%s\n", diff)
			}
		})
	}
}

func TestIsModuleChanged(t *testing.T) {
	tests := map[string]struct {
		module     *ModuleTreeNode
		submodules []string
		changes    []string
		want       bool
		wantErr    bool
	}{
		"no submodules": {
			module: &ModuleTreeNode{
				absPath: ".", relPath: ".",
			},
			changes: []string{
				"sub3/foo.go",
				"sub2/bar.go",
				"sub1/baz.go",
				"foo.go",
			},
			want: true,
		},
		"no submodules, no go changes": {
			module: &ModuleTreeNode{
				absPath: ".", relPath: ".",
			},
			changes: []string{
				"foo.java",
			},
			want: false,
		},
		"go.mod considered": {
			module: &ModuleTreeNode{
				absPath: ".", relPath: ".",
			},
			changes: []string{
				"go.mod",
			},
			want: true,
		},
		"repo root with submodules": {
			module: &ModuleTreeNode{
				absPath: ".", relPath: ".",
				subModules: []*ModuleTreeNode{
					{absPath: "sub1", relPath: "sub1"},
					{absPath: "sub2", relPath: "sub2"},
				},
			},
			changes: []string{
				"sub3/foo.go",
				"sub2/bar.go",
				"sub1/baz.go",
				"foo.go",
			},
			want: true,
		},
		"submodule directory, no submodules, no changes": {
			module: &ModuleTreeNode{
				absPath: "sub1", relPath: "sub1",
			},
			changes: []string{
				"sub3/foo.go",
				"sub2/bar.go",
				"foo.go",
			},
		},
		"submodule directory, no submodules, changes": {
			module: &ModuleTreeNode{
				absPath: "sub1", relPath: "sub1",
			},
			changes: []string{
				"sub3/foo.go",
				"sub2/bar.go",
				"sub1/bar.go",
				"foo.go",
			},
			want: true,
		},
		"submodule directory, submodules, no changes": {
			module: &ModuleTreeNode{
				absPath: "sub1", relPath: "sub1",
				subModules: []*ModuleTreeNode{
					{absPath: "sub1/subsub1", relPath: "sub1/subsub1"},
					{absPath: "sub1/subsub2", relPath: "sub1/subsub2"},
				},
			},
			changes: []string{
				"sub3/foo.go",
				"sub2/bar.go",
				"sub1/subsub1/foo.go",
				"sub1/subsub1/bar.go",
				"sub1/subsub2/bar.go",
				"foo.go",
			},
		},
		"submodule directory, submodules, changes": {
			module: &ModuleTreeNode{
				absPath: "sub1", relPath: "sub1",
				subModules: []*ModuleTreeNode{
					{absPath: "sub1/subsub1", relPath: "sub1/subsub1"},
					{absPath: "sub1/subsub2", relPath: "sub1/subsub2"},
				},
			},
			changes: []string{
				"sub3/foo.go",
				"sub2/bar.go",
				"sub1/subsub1/foo.go",
				"sub1/subsub1/bar.go",
				"sub1/subsub2/bar.go",
				"sub1/notsub/foo.go",
				"foo.go",
			},
			want: true,
		},
		"module with no changes, but shares common prefix with a changed module": {
			module: &ModuleTreeNode{
				absPath: "foobar", relPath: "foobar",
				subModules: []*ModuleTreeNode{
					{absPath: "foobar/sub1", relPath: "foobar/sub1"},
				},
			},
			changes: []string{
				"foobarbaz/baz.go",
				"foobar/sub1/sub1.go",
			},
			want: false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := IsModuleChanged(tt.module, tt.changes)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsModuleChanged() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsModuleChanged() got = %v, want %v", got, tt.want)
			}
		})
	}
}
