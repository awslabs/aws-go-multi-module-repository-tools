package release

import (
	"testing"

	"github.com/awslabs/aws-go-multi-module-repository-tools/gomod"
)

func Test_isModuleCarvedOut1(t *testing.T) {
	tests := map[string]struct {
		files   []string
		module  *gomod.ModuleTreeNode
		want    bool
		wantErr bool
	}{
		"tombstone, no go.mod, has nested go source": {
			module: func() *gomod.ModuleTreeNode {
				tree := gomod.NewModuleTree()
				tree.Insert(".")
				tree.Insert("a", "tombstone")
				return tree.List()[0]
			}(),
			files: []string{
				"a/c/foo.go",
			},
			want: false,
		},
		"tombstone and submodule, no go.mod, has nested go source": {
			module: func() *gomod.ModuleTreeNode {
				tree := gomod.NewModuleTree()
				tree.Insert(".")
				tree.Insert("a", "tombstone")
				tree.Insert("a/c")
				return tree.Get("a/c")
			}(),
			files: []string{
				"a/c/foo.go",
			},
			want: true,
		},
		"no submodules, has go.mod, has go source": {
			module: func() *gomod.ModuleTreeNode {
				tree := gomod.NewModuleTree()
				tree.Insert(".")
				return tree.List()[0]
			}(),
			files: []string{
				"a/go.mod",
				"a/foo.go",
			},
			want: false,
		},
		"no submodules, no go.mod, has go source": {
			module: func() *gomod.ModuleTreeNode {
				tree := gomod.NewModuleTree()
				tree.Insert(".")
				return tree.List()[0]
			}(),
			files: []string{
				"a/foo.go",
			},
			want: true,
		},
		"no submodules, no files": {
			want: false,
		},
		"submodules, no go.mod, no go source": {
			files: []string{
				"a/b/go.mod",
				"a/b/foo.go",
				"a/c/go.mod",
				"a/c/bar.go",
			},
			module: func() *gomod.ModuleTreeNode {
				tree := gomod.NewModuleTree()
				tree.Insert(".")
				tree.Insert("a/b")
				tree.Insert("a/c")
				return tree.List()[0]
			}(),
			want: false,
		},
		"submodules, has go.mod, no go source": {
			files: []string{
				"a/b/go.mod",
				"a/b/foo.go",
				"a/c/go.mod",
				"a/c/bar.go",
				"a/go.mod",
			},
			module: func() *gomod.ModuleTreeNode {
				tree := gomod.NewModuleTree()
				tree.Insert(".")
				tree.Insert("a/b")
				tree.Insert("a/c")
				return tree.List()[0]
			}(),
			want: false,
		},
		"submodules, no go.mod, has go source": {
			files: []string{
				"a/b/go.mod",
				"a/b/foo.go",
				"a/c/go.mod",
				"a/c/bar.go",
				"a/foo.go",
			},
			module: func() *gomod.ModuleTreeNode {
				tree := gomod.NewModuleTree()
				tree.Insert(".")
				tree.Insert("a/b")
				tree.Insert("a/c")
				return tree.List()[0]
			}(),
			want: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := isModuleCarvedOut(tt.module, tt.files)
			if (err != nil) != tt.wantErr {
				t.Errorf("isModuleCarvedOut() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("isModuleCarvedOut() got = %v, want %v", got, tt.want)
			}
		})
	}
}
