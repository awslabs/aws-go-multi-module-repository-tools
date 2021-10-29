package release

import (
	"fmt"
	"strings"
	"testing"
	"time"

	repotools "github.com/awslabs/aws-go-multi-module-repository-tools"
	"github.com/awslabs/aws-go-multi-module-repository-tools/changelog"
	"github.com/awslabs/aws-go-multi-module-repository-tools/gomod"
	"github.com/google/go-cmp/cmp"

	"golang.org/x/mod/modfile"
)

type mockFinder struct {
	RootPath string
	Modules  map[string][]string
}

func (m *mockFinder) Root() string {
	return m.RootPath
}

func (m *mockFinder) ModulesRel() (map[string][]string, error) {
	return m.Modules, nil
}

func TestCalculateNextVersion(t *testing.T) {
	type args struct {
		modulePath  string
		latest      string
		config      repotools.ModuleConfig
		annotations []changelog.Annotation
	}
	tests := map[string]struct {
		args     args
		wantNext string
		wantErr  bool
	}{
		"new module v1 major": {
			args: args{
				modulePath: "github.com/aws/aws-sdk-go-v2/service/shinynew",
			},
			wantNext: "v1.0.0-preview",
		},
		"new module v1 major with release annotation": {
			args: args{
				modulePath: "github.com/aws/aws-sdk-go-v2/service/shinynew",
				annotations: []changelog.Annotation{{
					Type: changelog.ReleaseChangeType,
				}},
			},
			wantNext: "v1.0.0",
		},
		"new module v2 or higher major": {
			args: args{
				modulePath: "github.com/aws/aws-sdk-go-v2/service/shinynew/v2",
			},
			wantNext: "v2.0.0-preview",
		},
		"new module v2 or higher with release annotation": {
			args: args{
				modulePath: "github.com/aws/aws-sdk-go-v2/service/shinynew/v2",
				annotations: []changelog.Annotation{{
					Type: changelog.ReleaseChangeType,
				}},
			},
			wantNext: "v2.0.0",
		},
		"existing module version, not pre-release, no annotation": {
			args: args{
				modulePath: "github.com/aws/aws-sdk-go-v2/service/existing",
				latest:     "v1.0.0",
			},
			wantNext: "v1.0.1",
		},
		"existing module version, not pre-release, with patch semver annotation": {
			args: args{
				modulePath: "github.com/aws/aws-sdk-go-v2/service/existing",
				latest:     "v1.0.0",
				annotations: []changelog.Annotation{
					{Type: changelog.BugFixChangeType},
				},
			},
			wantNext: "v1.0.1",
		},
		"existing module version, not pre-release, with minor semver annotation": {
			args: args{
				modulePath: "github.com/aws/aws-sdk-go-v2/service/existing",
				latest:     "v1.0.1",
				annotations: []changelog.Annotation{
					{Type: changelog.FeatureChangeType},
				},
			},
			wantNext: "v1.1.0",
		},
		"existing module version, set for pre-release": {
			args: args{
				modulePath: "github.com/aws/aws-sdk-go-v2/service/existing",
				latest:     "v1.0.1",
				config:     repotools.ModuleConfig{PreRelease: "rc"},
			},
			wantNext: "v1.1.0-rc",
		},
		"existing module preview version": {
			args: args{
				modulePath: "github.com/aws/aws-sdk-go-v2/service/existing",
				latest:     "v1.1.0-preview",
				config:     repotools.ModuleConfig{PreRelease: "preview"},
			},
			wantNext: "v1.1.0-preview.1",
		},
		"existing module preview version, with non-release annotation types": {
			args: args{
				modulePath: "github.com/aws/aws-sdk-go-v2/service/existing",
				latest:     "v1.1.0-preview.1",
				config:     repotools.ModuleConfig{PreRelease: "preview"},
				annotations: []changelog.Annotation{{
					Type: changelog.FeatureChangeType,
				}},
			},
			wantNext: "v1.1.0-preview.2",
		},
		"existing module preview version, with new pre-release tag": {
			args: args{
				modulePath: "github.com/aws/aws-sdk-go-v2/service/existing",
				latest:     "v1.1.0-preview.2",
				config:     repotools.ModuleConfig{PreRelease: "rc"},
				annotations: []changelog.Annotation{{
					Type: changelog.FeatureChangeType,
				}},
			},
			wantNext: "v1.1.0-rc",
		},
		"existing module preview version, with new invalid pre-release tag": {
			args: args{
				modulePath: "github.com/aws/aws-sdk-go-v2/service/existing",
				latest:     "v1.1.0-rc.5",
				config:     repotools.ModuleConfig{PreRelease: "alpha"},
				annotations: []changelog.Annotation{{
					Type: changelog.FeatureChangeType,
				}},
			},
			wantErr: true,
		},
		"existing module preview version, with release annotation": {
			args: args{
				modulePath: "github.com/aws/aws-sdk-go-v2/service/existing",
				latest:     "v1.1.0-rc.5",
				annotations: []changelog.Annotation{{
					Type: changelog.ReleaseChangeType,
				}},
			},
			wantNext: "v1.1.0",
		},
		"invalid latest tag": {
			args: args{
				modulePath: "github.com/aws/aws-sdk-go-v2/service/existing",
				latest:     "1.1.0",
			},
			wantErr: true,
		},
		"module tag with build metadata": {
			args: args{
				modulePath: "github.com/aws/aws-sdk-go-v2/service/existing",
				latest:     "v1.1.0+build.12345",
			},
			wantNext: "v1.1.1",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotNext, err := CalculateNextVersion(tt.args.modulePath, tt.args.latest, tt.args.config, tt.args.annotations)
			if (err != nil) != tt.wantErr {
				t.Errorf("CalculateNextVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotNext != tt.wantNext {
				t.Errorf("CalculateNextVersion() gotNext = %v, want %v", gotNext, tt.wantNext)
			}
		})
	}
}

func TestNextReleaseID(t *testing.T) {
	origNowTime := nowTime
	defer func() {
		nowTime = origNowTime
	}()

	type args struct {
		tags []string
	}
	tests := map[string]struct {
		args     args
		nowTime  func() time.Time
		wantNext string
	}{
		"no tags": {
			wantNext: "2021-05-06",
		},
		"other tags": {
			args:     args{tags: []string{"v1.2.0", "release/foo/v2"}},
			wantNext: "2021-05-06",
		},
		"older tags": {
			args:     args{tags: []string{"release-2021-05-04", "release-2021-05-04.2"}},
			wantNext: "2021-05-06",
		},
		"second release": {
			args:     args{tags: []string{"release-2021-05-06"}},
			wantNext: "2021-05-06.2",
		},
		"third release": {
			args:     args{tags: []string{"release-2021-05-06", "release-2021-05-06.2"}},
			wantNext: "2021-05-06.3",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if tt.nowTime == nil {
				nowTime = func() time.Time {
					return time.Date(2021, 5, 6, 7, 8, 9, 10, time.UTC)
				}
			} else {
				nowTime = tt.nowTime
			}

			if gotNext := NextReleaseID(tt.args.tags); gotNext != tt.wantNext {
				t.Errorf("NextReleaseID() = %v, want %v", gotNext, tt.wantNext)
			}
		})
	}
}

func TestBuildReleaseManifest(t *testing.T) {
	const smithyGoRootGoMod = `module github.com/aws/smithy-go

require (
	github.com/google/go-cmp v0.5.6
)

go 1.15`
	const sdkRootGoMod = `module github.com/aws/aws-sdk-go-v2

require (
	github.com/aws/smithy-go v1.8.1
	github.com/google/go-cmp v0.5.6
	github.com/jmespath/go-jmespath v0.4.0
)

go 1.15`
	const configGoMod = `module github.com/aws/aws-sdk-go-v2/config

go 1.15

require (
	github.com/aws/aws-sdk-go-v2 v1.10.0
	github.com/google/go-cmp v0.5.6
)`
	cases := map[string]struct {
		ModuleTree *gomod.ModuleTree
		ID         string
		Modules    map[string]*Module
		Verbose    bool

		ExpectManifest Manifest
	}{
		"multi-module": {
			ID: "2021-10-27",
			ModuleTree: func() *gomod.ModuleTree {
				tree := gomod.NewModuleTree()
				tree.InsertRel("")
				tree.InsertRel("config")
				return tree
			}(),
			Modules: map[string]*Module{
				"github.com/aws/aws-sdk-go-v2": {
					File: func() *modfile.File {
						f, err := gomod.ReadModule("go.mod", strings.NewReader(sdkRootGoMod), nil, false)
						if err != nil {
							panic(fmt.Errorf("expect no error reading module, %v", err).Error())
						}
						return f
					}(),
					RelativeRepoPath: ".",
					Latest:           "v1.0.0",
				},
				"github.com/aws/aws-sdk-go-v2/config": {
					File: func() *modfile.File {
						f, err := gomod.ReadModule("config/go.mod", strings.NewReader(configGoMod), nil, false)
						if err != nil {
							panic(fmt.Errorf("expect no error reading module, %v", err).Error())
						}
						return f
					}(),
					RelativeRepoPath: "config",
					Latest:           "v1.0.0",
					Changes:          SourceChange,
					FileChanges: []string{
						"config/foo.go",
					},
				},
			},
			ExpectManifest: Manifest{
				ID:             "2021-10-27",
				WithReleaseTag: true,
				Modules: map[string]ModuleManifest{
					"config": {
						ModulePath: "github.com/aws/aws-sdk-go-v2/config",
						From:       "v1.0.0",
						To:         "v1.0.1",
						Changes:    SourceChange,
					},
				},
				Tags: []string{
					"config/v1.0.1",
				},
			},
		},
		"verbose multi-module": {
			ID:      "2021-10-27",
			Verbose: true,
			ModuleTree: func() *gomod.ModuleTree {
				tree := gomod.NewModuleTree()
				tree.InsertRel(".")
				tree.InsertRel("config")
				return tree
			}(),
			Modules: map[string]*Module{
				"github.com/aws/aws-sdk-go-v2": {
					File: func() *modfile.File {
						f, err := gomod.ReadModule("go.mod", strings.NewReader(sdkRootGoMod), nil, false)
						if err != nil {
							panic(fmt.Errorf("expect no error reading module, %v", err).Error())
						}
						return f
					}(),
					RelativeRepoPath: ".",
					Latest:           "v1.0.0",
				},
				"github.com/aws/aws-sdk-go-v2/config": {
					File: func() *modfile.File {
						f, err := gomod.ReadModule("config/go.mod", strings.NewReader(configGoMod), nil, false)
						if err != nil {
							panic(fmt.Errorf("expect no error reading module, %v", err).Error())
						}
						return f
					}(),
					RelativeRepoPath: "config",
					Latest:           "v1.0.0",
					Changes:          SourceChange,
					FileChanges: []string{
						"config/foo.go",
					},
				},
			},
			ExpectManifest: Manifest{
				ID:             "2021-10-27",
				WithReleaseTag: true,
				Modules: map[string]ModuleManifest{
					"config": {
						ModulePath: "github.com/aws/aws-sdk-go-v2/config",
						From:       "v1.0.0",
						To:         "v1.0.1",
						Changes:    SourceChange,
						FileChanges: []string{
							"config/foo.go",
						},
					},
				},
				Tags: []string{
					"config/v1.0.1",
				},
			},
		},
		"single-module": {
			ID: "2021-10-27",
			ModuleTree: func() *gomod.ModuleTree {
				tree := gomod.NewModuleTree()
				tree.InsertRel(".")
				return tree
			}(),
			Modules: map[string]*Module{
				"github.com/aws/smithy-go": {
					File: func() *modfile.File {
						f, err := gomod.ReadModule("go.mod", strings.NewReader(smithyGoRootGoMod), nil, false)
						if err != nil {
							panic(fmt.Errorf("expect no error reading module, %v", err).Error())
						}
						return f
					}(),
					RelativeRepoPath: ".",
					Latest:           "v1.2.3",
					Changes:          SourceChange,
					FileChanges: []string{
						"config/foo.go",
					},
				},
			},
			ExpectManifest: Manifest{
				ID:             "v1.2.4",
				WithReleaseTag: false,
				Modules: map[string]ModuleManifest{
					".": {
						ModulePath: "github.com/aws/smithy-go",
						From:       "v1.2.3",
						To:         "v1.2.4",
						Changes:    SourceChange,
					},
				},
				Tags: []string{
					"v1.2.4",
				},
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			manifest, err := BuildReleaseManifest(tt.ModuleTree, tt.ID, tt.Modules, tt.Verbose)
			if err != nil {
				t.Fatalf("expect no error, got %v", err)
			}

			if diff := cmp.Diff(tt.ExpectManifest, manifest); diff != "" {
				t.Errorf("expect manifest match, got\n%s", diff)
			}
		})
	}
}
