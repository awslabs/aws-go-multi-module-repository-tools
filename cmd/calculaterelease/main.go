package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	repotools "github.com/awslabs/aws-go-multi-module-repository-tools"
	"github.com/awslabs/aws-go-multi-module-repository-tools/changelog"
	"github.com/awslabs/aws-go-multi-module-repository-tools/git"
	"github.com/awslabs/aws-go-multi-module-repository-tools/gomod"
	"github.com/awslabs/aws-go-multi-module-repository-tools/release"
)

var verbose bool
var outputFile string

func init() {
	flag.BoolVar(&verbose, "v", false, "output with verbose changes")
	flag.StringVar(&outputFile, "o", "", "output file")
}

func main() {
	flag.Parse()

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

	log.Println("Calculating module changes")
	modulesForRelease, err := release.Calculate(discoverer, taggedModules, config, annotations)
	if err != nil {
		log.Fatal(err)
	}

	id := release.NextReleaseID(tags)

	manifest, err := release.BuildReleaseManifest(id, modulesForRelease, verbose)
	if err != nil {
		log.Fatal(err)
	}

	marshal, err := json.MarshalIndent(manifest, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	if len(outputFile) == 0 {
		fmt.Printf("%v\n", string(marshal))
		return
	}

	file, err := os.OpenFile(outputFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	if _, err = io.Copy(file, bytes.NewReader(marshal)); err != nil {
		log.Fatal(err)
	}
}
