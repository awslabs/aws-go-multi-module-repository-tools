package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	repotools "github.com/awslabs/aws-go-multi-module-repository-tools"
	"github.com/awslabs/aws-go-multi-module-repository-tools/changelog"
	"github.com/awslabs/aws-go-multi-module-repository-tools/git"
	"github.com/awslabs/aws-go-multi-module-repository-tools/gomod"
	"github.com/awslabs/aws-go-multi-module-repository-tools/release"
)

var (
	getUnreleasedVersion bool
	preview              preReleaseFlag
)

func init() {
	flag.BoolVar(&getUnreleasedVersion, "unreleased", false,
		"Returns the version the projected version the module will be at after the next release")
	flag.Var(&preview, "preview",
		"Indicates a semver pre-release should be calculated when specified with the -unreleased flag.")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `Usage of %s [-unreleased] <module>
  module
	The relative path of the module to get the version of.
`, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if len(flag.Args()) != 1 {
		flag.Usage()
		log.Fatalf("no module specified")
	}
	moduleToCheck := flag.Args()[0]

	repoRoot, err := repotools.GetRepoRoot()
	if err != nil {
		log.Fatalf("failed to get repository root: %v", err)
	}

	config, err := repotools.LoadConfig(repoRoot)
	if err != nil {
		log.Fatalf("failed to load repotools config: %v", err)
	}

	discoverer := gomod.NewDiscoverer(repoRoot)

	if err := discoverer.Discover(); err != nil {
		log.Fatalf("failed to discover repository modules: %v", err)
	}

	tags, err := git.Tags(repoRoot)
	if err != nil {
		log.Fatalf("failed to get git tags: %v", err)
	}

	taggedModules := git.ParseModuleTags(tags)

	annotations, err := changelog.GetAnnotations(repoRoot)
	if err != nil {
		log.Fatal(err)
	}

	checkedModules, err := release.Calculate(discoverer, taggedModules, config, annotations)
	if err != nil {
		log.Fatalf("failed to check repo modules, %v", err)
	}

	if getUnreleasedVersion {
		id := release.NextReleaseID(tags)
		manifest, err := release.BuildReleaseManifest(discoverer.Modules(), id, checkedModules, false, preview.String())
		if err != nil {
			log.Fatalf("failed to build release manifest, %v", err)
		}

		if m, ok := manifest.Modules[moduleToCheck]; ok {
			moduleTag, err := git.ToModuleTag(moduleToCheck, m.To)
			if err != nil {
				log.Fatalf("failed to get module %v tag, %v", moduleToCheck, err)
			}
			fmt.Println(moduleTag)
			return
		}
	}

	checkedModule, ok := release.FindModuleViaRelativeRepoPath(checkedModules, moduleToCheck)
	if !ok {
		log.Fatalf("failed to find version for module, %v", moduleToCheck)
	}

	moduleVersion := checkedModule.Latest
	if checkedModule.Latest == "" {
		moduleVersion = "v0.0.0-00010101000000-000000000000"
	}
	fmt.Println(moduleVersion)
}

type preReleaseFlag string

func (p *preReleaseFlag) String() string {
	return string(*p)
}

func (p *preReleaseFlag) Set(s string) error {
	*p = preReleaseFlag(s)
	return nil
}
