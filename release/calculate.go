package release

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"

	repotools "github.com/awslabs/aws-go-multi-module-repository-tools"
	"github.com/awslabs/aws-go-multi-module-repository-tools/changelog"
	"github.com/awslabs/aws-go-multi-module-repository-tools/git"
	"github.com/awslabs/aws-go-multi-module-repository-tools/gomod"
)

// ModuleFinder is a type that searches for modules
type ModuleFinder interface {
	// Absolute Path of the root directory all modules are nested within.
	Root() string

	// Returns a tree of the known modules.
	Modules() *gomod.ModuleTree
}

const tombstonedModuleAttrib = "tombstone"

// Calculate calculates the modules to be released and their next versions
// based on the Git history, previous tags, module configuration, and
// associated changelog annotations.
func Calculate(finder ModuleFinder, tags git.ModuleTags, config repotools.Config, annotations []changelog.Annotation) (map[string]*Module, error) {
	rootDir := finder.Root()

	repositoryModules := finder.Modules()

	moduleAnnotations := make(map[string][]changelog.Annotation)
	for _, annotation := range annotations {
		for _, am := range annotation.Modules {
			moduleAnnotations[am] = append(moduleAnnotations[am], annotation)
		}
	}

	// Add modules to the tree that have been tombstoned, and removed.
	for moduleTag := range tags {
		if m := repositoryModules.Get(moduleTag); m == nil {
			if _, err := repositoryModules.InsertRel(moduleTag, tombstonedModuleAttrib); err != nil {
				return nil, fmt.Errorf("failed to insert tombstone module, %w", err)
			}
		}
	}

	checkedModules := map[string]*Module{}
	for it := repositoryModules.Iterator(); ; {
		module := it.Next()
		if module == nil {
			break
		}

		var latestVersion string
		var hasChanges bool
		var changes []string

		// Tombstone modules must have no files, (excludes submodules).
		if module.HasAttribute(tombstonedModuleAttrib) {
			files, err := listRelFiles(rootDir, module.AbsPath())
			if err != nil {
				return nil, fmt.Errorf("failed to list tombstone module files, %w", err)
			}

			files, err = gomod.FilterModuleFiles(module, files)
			if err != nil {
				return nil, fmt.Errorf("failed to filter tombstone module files, %w", err)
			}

			if len(files) != 0 {
				return nil, fmt.Errorf("tombstone module has go source files, %v", files)
			}
			continue
		}

		moduleFile, err := gomod.LoadModuleFile(module.AbsPath(), nil, true)
		if err != nil {
			return nil, fmt.Errorf("failed to load module file: %w", err)
		}
		modulePath, err := gomod.GetModulePath(moduleFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read module path: %w", err)
		}

		latestVersion, ok := tags.Latest(module.Path())
		if ok {
			startTag, err := git.ToModuleTag(module.Path(), latestVersion)
			if err != nil {
				log.Fatalf("failed to convert module path and version to tag: %v", err)
			}

			changes, err = git.Changes(finder.Root(), startTag, "HEAD", module.Path())
			if err != nil {
				log.Fatalf("failed to get git changes: %v", err)
			}

			// Only consider changes that are specific to this module. Other
			// module changes will be considered separately.
			changes, err = gomod.FilterModuleFiles(module, changes)
			if err != nil {
				return nil, fmt.Errorf("failed to determine module changes: %w", err)
			}
			hasChanges = len(changes) != 0

			if !hasChanges {
				// Check if any of the submodules have been "carved out" of
				// this module since the last tagged release
				for it := module.Iterator(); ; {
					subModule := it.Next()
					if subModule == nil {
						break
					}

					// Ignore Tombstoned modules, since they no longer exist locally.
					if module.HasAttribute(tombstonedModuleAttrib) {
						continue
					}

					// Is an existing submodule?
					//  - yes, skip existing modules
					//  - no, check if new modules is a carve out
					if _, ok := tags.Latest(subModule.Path()); ok {
						continue
					}

					// Did parent module contain this path previously in its tree?
					treeFiles, err := git.LsTree(rootDir, startTag, subModule.Path())
					if err != nil {
						return nil, fmt.Errorf("failed to list git tree: %v", err)
					}

					carvedOut, err := isModuleCarvedOut(subModule, treeFiles)
					if err != nil {
						return nil, err
					}
					if carvedOut {
						hasChanges = true
						break
					}
				}
			}
		}

		var changeReason ModuleChange
		if hasChanges && len(latestVersion) > 0 {
			// Has changes and is an existing module
			changeReason |= SourceChange
		} else if len(latestVersion) == 0 {
			// New module with changes.
			changeReason |= NewModule
		}

		checkedModules[modulePath] = &Module{
			File:              moduleFile,
			RelativeRepoPath:  module.Path(),
			Latest:            latestVersion,
			Changes:           changeReason,
			FileChanges:       changes,
			ChangeAnnotations: moduleAnnotations[module.Path()],
			ModuleConfig:      config.Modules[module.Path()],
		}
	}

	if err := CalculateDependencyUpdates(checkedModules); err != nil {
		return nil, err
	}

	for modulePath := range checkedModules {
		if checkedModules[modulePath].Changes == 0 || config.Modules[modulePath].NoTag {
			delete(checkedModules, modulePath)
		}
	}

	return checkedModules, nil
}

// isModuleCarvedOut takes a list of files for a (new) submodule directory. The
// list of files are the files that are located in the submodule directory path
// from the parent's previous tagged release. Returns true the new submodule
// has been carved out of the parent module directory it is located under. This
// is determined by looking through the file list and determining if Go source
// is present but no `go.mod` file existed.
func isModuleCarvedOut(module *gomod.ModuleTreeNode, files []string) (carveOut bool, err error) {
	files, err = gomod.FilterModuleFiles(module, files)
	if err != nil {
		return false, fmt.Errorf("failed to filter tree files, %v", err)
	}

	var hasGoSource, hasGoMod bool
	for _, file := range files {
		dir, fileName := path.Split(file)
		dir = path.Clean(dir)

		if gomod.IsGoMod(fileName) {
			hasGoMod = true
		}
		if gomod.IsGoSource(fileName) {
			hasGoSource = true
		}

		if hasGoMod && hasGoSource {
			break
		}
	}

	// Carved out modules are those that were a Go package originally, then had
	// a go module added.
	return !hasGoMod && hasGoSource, nil
}

func listRelFiles(rootPath, modulePath string) (files []string, err error) {
	err = filepath.Walk(modulePath, func(path string, info os.FileInfo, err error) error {
		if err != nil && os.IsNotExist(err) {
			return filepath.SkipDir
		}
		if err != nil || info.IsDir() {
			return err
		}

		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(files); i++ {
		files[i], err = filepath.Rel(rootPath, files[i])
		if err != nil {
			return nil, fmt.Errorf("unable to get module file relative path, %w", err)
		}
	}

	return files, nil
}
