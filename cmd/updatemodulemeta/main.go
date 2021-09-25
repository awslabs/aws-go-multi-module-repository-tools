package main

import (
	"encoding/json"
	"flag"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	repotools "github.com/awslabs/aws-go-multi-module-repository-tools"
	"github.com/awslabs/aws-go-multi-module-repository-tools/git"
	"github.com/awslabs/aws-go-multi-module-repository-tools/gomod"
	"github.com/awslabs/aws-go-multi-module-repository-tools/release"
)

const metadataFile = "go_module_metadata.go"

var metadataTemplate = template.Must(template.New("metadata").
	Parse(`// Code generated by internal/repotools/cmd/updatemodulemeta DO NOT EDIT.

package {{ .Package }}

// goModuleVersion is the tagged release for this module
const goModuleVersion = {{ printf "%q" .Version }}
`))

var releaseFileName string

func init() {
	flag.StringVar(&releaseFileName, "release", "", "release manifest file path")
}

func main() {
	flag.Parse()

	repoRoot, err := repotools.GetRepoRoot()
	if err != nil {
		log.Fatalf("failed to get repository root: %v", err)
	}

	config, err := repotools.LoadConfig(repoRoot)
	if err != nil {
		log.Fatalf("failed to load repository config: %v", err)
	}

	discoverer := gomod.NewDiscoverer(repoRoot)

	if err = discoverer.Discover(); err != nil {
		log.Fatalf("failed to discover modules: %v", err)
	}

	tags, err := git.Tags(repoRoot)
	if err != nil {
		log.Fatalf("failed to retrieve git tags: %v", err)
	}

	moduleTags := git.ParseModuleTags(tags)

	if len(releaseFileName) > 0 {
		manifest, err := loadManifest(releaseFileName)
		if err != nil {
			log.Fatalf("failed to load release manifest file: %v", err)
		}
		for _, tag := range manifest.Tags {
			moduleTags.Add(tag)
		}
	}

	modules := discoverer.Modules()
	for it := modules.Iterator(); ; {
		module := it.Next()
		if module == nil {
			break
		}

		cfg := config.Modules[module.Path()]
		dirPath := module.AbsPath()
		if len(cfg.MetadataPackage) > 0 {
			pkgRel := filepath.Join(module.Path(), cfg.MetadataPackage)
			if m := module.Search(pkgRel); m != nil {
				log.Fatalf("%s metadata_package location must not be located in a sub-module",
					module.Path())
			}
			dirPath = filepath.Join(repoRoot, pkgRel)
		}
		goPackage, err := getModuleGoPackage(dirPath)
		if err != nil {
			log.Fatalf("failed to determine module go package: %v", err)
		}
		if len(goPackage) == 0 {
			log.Printf("[WARN] unable to determine go package for %v...skipping", module.Path())
			continue
		}
		latest, isTagged := moduleTags.Latest(module.Path())

		if cfg, ok := config.Modules[module.Path()]; (ok && cfg.NoTag) || !isTagged {
			latest = "tip"
		}

		if err := writeModuleMetadata(dirPath, goPackage, latest); err != nil {
			log.Fatalf("failed to write module metadata: %v", err)
		}
	}
}

func getModuleGoPackage(dir string) (string, error) {
	var inspectFile string
	{
		metaFile := filepath.Join(dir, metadataFile)
		if _, err := os.Stat(metaFile); err == nil {
			inspectFile = metaFile
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	if len(inspectFile) == 0 {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || path == dir {
				return err
			}

			if len(inspectFile) > 0 {
				return nil
			}

			if info.IsDir() {
				return filepath.SkipDir
			}

			name := info.Name()
			if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
				inspectFile = path
			}

			return nil
		})
		if err != nil {
			return "", err
		}
	}

	if len(inspectFile) == 0 {
		return "", nil
	}

	return readGoPackage(inspectFile)
}

func readGoPackage(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	parseFile, err := parser.ParseFile(token.NewFileSet(), f.Name(), f, parser.PackageClauseOnly)
	if err != nil {
		return "", err
	}

	return parseFile.Name.Name, nil
}

type metadata struct {
	Package string
	Version string
}

func writeModuleMetadata(dir string, goPackage string, version string) (err error) {
	f, err := os.OpenFile(filepath.Join(dir, metadataFile), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		fErr := f.Close()
		if err == nil && fErr != nil {
			err = fErr
		}
	}()

	return metadataTemplate.Execute(f, metadata{
		Package: goPackage,
		Version: strings.TrimPrefix(version, "v"),
	})
}

func loadManifest(path string) (v release.Manifest, err error) {
	fb, err := ioutil.ReadFile(path)
	if err != nil {
		return release.Manifest{}, err
	}

	if err = json.Unmarshal(fb, &v); err != nil {
		return release.Manifest{}, err
	}

	return v, nil
}
