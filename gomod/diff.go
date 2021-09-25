package gomod

import (
	"path/filepath"
	"sort"
	"strings"
)

// FilterModuleFiles will return a list of files that apply to this specific
// module. Any file that is not relevant to this module will be excluded from
// the returned list. List will be empty if there are no relevant files.
func FilterModuleFiles(module *ModuleTreeNode, files []string) ([]string, error) {
	type modDir struct {
		filepaths []string
		relevant  bool
	}
	dirCache := map[string]modDir{}

	// Iterate through all files, building up a cache of files that are
	// relevant. Filtering out directories that are not relevant to the current
	// module.
	for _, filepathName := range files {
		dir, fileName := filepath.Split(filepathName)
		dir = filepath.Clean(dir)

		// Only consider Go file or module files as relevant.
		if !(IsGoSource(fileName) || IsGoMod(fileName)) {
			continue
		}

		// Only need to consider paths for files that are relevant.
		if v, ok := dirCache[dir]; ok {
			if !v.relevant {
				continue
			}

			v.filepaths = append(v.filepaths, filepathName)
			dirCache[dir] = v
			continue
		}

		if !module.ParentOf(dir) {
			dirCache[dir] = modDir{}
			continue
		}

		dirCache[dir] = modDir{
			relevant:  true,
			filepaths: []string{filepathName},
		}
	}

	var relevantFiles []string
	for _, dir := range dirCache {
		if !dir.relevant {
			continue
		}
		relevantFiles = append(relevantFiles, dir.filepaths...)
	}

	sort.Strings(relevantFiles)
	return relevantFiles, nil
}

// IsModuleChanged returns whether the given set of changes applies to the
// module directly, and not any of its sub modules.
func IsModuleChanged(module *ModuleTreeNode, changes []string) (bool, error) {
	changes, err := FilterModuleFiles(module, changes)
	if err != nil {
		return false, err
	}

	return len(changes) != 0, nil
}

// IsGoSource returns whether a given file name is a Go source code file ending
// in `.go`
func IsGoSource(name string) bool {
	return !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
}

// IsGoMod returns whether a given file name is `go.mod`.
func IsGoMod(name string) bool {
	return name == "go.mod"
}
