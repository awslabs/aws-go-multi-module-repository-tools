package repotools

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml"
)

const toolingConfigFile = "modman.toml"

// ModuleConfig is the configuration for the repository module
type ModuleConfig struct {
	// Indicates that the given module should not be tagged (released)
	NoTag bool `toml:"no_tag,omitempty"`

	// The semver pre-release string for the module
	PreRelease string `toml:"pre_release,omitempty"`

	// The package alternative location relative to the module where the go_module_metadata.go should be written.
	// By default this file is written in the location of the module root where the `go.mod` is located.
	MetadataPackage string `toml:"metadata_package,omitempty"`
}

// Config is a configuration file for describing how modules and dependencies are managed.
type Config struct {
	Modules      map[string]ModuleConfig `toml:"modules,omitempty"`
	Dependencies map[string]string       `toml:"dependencies,omitempty"`
}

func newConfig() Config {
	return Config{
		Modules:      map[string]ModuleConfig{},
		Dependencies: map[string]string{},
	}
}

// LoadConfig loads the tooling configuration file located in the directory path.
func LoadConfig(path string) (Config, error) {
	file, err := os.Open(filepath.Join(path, toolingConfigFile))
	if err != nil && os.IsNotExist(err) {
		return newConfig(), nil
	} else if err != nil {
		return Config{}, err
	}
	defer file.Close()

	return ReadConfig(file)
}

// ReadConfig reads the tooling configuration from the given reader.
func ReadConfig(reader io.Reader) (Config, error) {
	all, err := ioutil.ReadAll(reader)
	if err != nil {
		return Config{}, err
	}

	c := newConfig()
	if err = toml.Unmarshal(all, &c); err != nil {
		return Config{}, err
	}

	return c, nil
}

// WriteConfig writes the tooling configuration to the given path.
func WriteConfig(path string, config Config) (err error) {
	var f *os.File
	f, err = os.OpenFile(filepath.Join(path, toolingConfigFile), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		fErr := f.Close()
		if fErr != nil && err == nil {
			err = fErr
		}
	}()

	return toml.NewEncoder(f).Order(toml.OrderAlphabetical).Encode(config)
}
