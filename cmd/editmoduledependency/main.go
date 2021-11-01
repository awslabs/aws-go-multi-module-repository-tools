package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	repotools "github.com/awslabs/aws-go-multi-module-repository-tools"
)

var (
	setModule    string
	deleteModule string
	version      string
)

func init() {
	flag.StringVar(&setModule, "s", "",
		"Sets the `module` version into the repositories module management file. (Requires version)")
	flag.StringVar(&deleteModule, "d", "",
		"Deletes the `module` from the repositories module management file.")
	flag.StringVar(&version, "v", "", "The `version` of the Go module dependency set. (Only usable with set mode)")

	flag.Usage = func() {
		baseFilename := filepath.Base(os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Usages:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Set:\n  %s -s <module> -v <version>\n", baseFilename)
		fmt.Fprintf(flag.CommandLine.Output(), "Delete:\n  %s -d <module>\n", baseFilename)
		fmt.Fprintf(flag.CommandLine.Output(), "\nOptions:\n")
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if !((setModule == "" || deleteModule == "") && !(setModule == "" && deleteModule == "")) {
		flag.Usage()
		log.Fatalf("Use either set or delete mode")
	}
	if setModule != "" && version == "" {
		flag.Usage()
		log.Fatalf("Set mode requires both module and version")
	}
	if deleteModule != "" && version != "" {
		flag.Usage()
		log.Fatalf("Delete mode cannot be use with version")
	}

	repoRoot, err := repotools.GetRepoRoot()
	if err != nil {
		log.Fatalf("Failed to get repository root: %v", err)
	}

	config, err := repotools.LoadConfig(repoRoot)
	if err != nil {
		log.Fatalf("Failed to load repotools config: %v", err)
	}

	if setModule != "" {
		config, err = setModuleDependency(config, setModule, version)
	} else {
		config, err = deleteModuleDependency(config, deleteModule)
	}
	if err != nil {
		log.Fatalf("Failed to modify module dependency, %v", err)
	}

	if err = repotools.WriteConfig(repoRoot, config); err != nil {
		log.Fatalf("Failed to write module management file, %v", err)
	}
}

func setModuleDependency(config repotools.Config, module, verison string) (repotools.Config, error) {
	if v, ok := config.Dependencies[module]; ok {
		log.Printf("Updating module dependency %v: %v, to %v: %v", module, v, module, version)
	} else {
		log.Printf("Adding module dependency %v: %v", module, version)
	}

	config.Dependencies[module] = version

	return config, nil
}

func deleteModuleDependency(config repotools.Config, module string) (repotools.Config, error) {
	if v, ok := config.Dependencies[module]; ok {
		log.Printf("Deleting module dependency %v: %v", module, v)
	} else {
		return repotools.Config{}, fmt.Errorf("module %v is not a dependency", module)
	}

	delete(config.Dependencies, module)

	return config, nil
}
