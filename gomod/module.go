package gomod

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

const (
	goModuleFile   = "go.mod"
	testDataFolder = "testdata"
)

// GetModulePath retrieves the module path from the provide file description.
func GetModulePath(file *modfile.File) (string, error) {
	if file.Module == nil {
		return "", fmt.Errorf("module directive not present")
	}
	return file.Module.Mod.Path, nil
}

// LoadModuleFile loads the Go module file located at the provided directory path.
func LoadModuleFile(path string, fix modfile.VersionFixer, lax bool) (*modfile.File, error) {
	path = filepath.Join(path, goModuleFile)

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ReadModule(path, f, fix, lax)
}

// ReadModule parses the module file bytes from the provided reader.
func ReadModule(path string, f io.Reader, fix modfile.VersionFixer, lax bool) (parse *modfile.File, err error) {
	fBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	if lax {
		parse, err = modfile.ParseLax(path, fBytes, fix)
	} else {
		parse, err = modfile.Parse(path, fBytes, fix)
	}
	if err != nil {
		return nil, err
	}

	return parse, nil
}

// WriteModuleFile writes the Go module description to the provided directory path.
func WriteModuleFile(path string, file *modfile.File) (err error) {
	modPath := filepath.Join(path, goModuleFile)

	var mf *os.File
	mf, err = os.OpenFile(modPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		fErr := mf.Close()
		if fErr != nil && err == nil {
			err = fErr
		}
	}()

	var fb []byte
	fb, err = file.Format()
	if err != nil {
		return err
	}

	_, err = io.Copy(mf, bytes.NewReader(fb))

	return err
}

// Discoverer is used for discovering all modules and submodules at the provided path.
type Discoverer struct {
	path    string
	modules *ModuleTree
}

// NewDiscoverer constructs a new Discover for the given path.
func NewDiscoverer(path string) *Discoverer {
	return &Discoverer{
		path: path,
	}
}

// Root returns the root path of the module discovery.
func (d *Discoverer) Root() string {
	return d.path
}

// Modules returns the modules discovered after executing Discover.
func (d *Discoverer) Modules() *ModuleTree {
	return d.modules
}

// Discover will find all modules starting from the path provided when
// constructing the Discoverer. Does not iterate into testdata folders.
//
// Any previous modules discovered by Discovery will be reset.
func (d *Discoverer) Discover() error {
	d.modules = NewModuleTree(func(o *ModuleTreeOptions) {
		o.RootPath = d.path
	})

	return filepath.Walk(d.path, d.walkChildModules)
}

func (d *Discoverer) walkChildModules(path string, fs os.FileInfo, err error) error {
	if err != nil || !fs.IsDir() {
		return err
	}

	if fs.Name() == testDataFolder || strings.HasPrefix(fs.Name(), ".") {
		return filepath.SkipDir
	}

	hasGoMod, err := IsGoModPresent(path)
	if err != nil {
		return err
	}

	if !hasGoMod {
		return nil
	}

	if _, err = d.modules.Insert(path); err != nil {
		return fmt.Errorf("unable to insert discovered module, %w", err)
	}
	return nil
}

// IsGoModPresent returns whether there is a go.mod file located in the provided directory path
func IsGoModPresent(path string) (bool, error) {
	_, err := os.Stat(filepath.Join(path, goModuleFile))
	if err != nil && os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}
