// Copyright 2022 The Okteto Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

func TestManifestExpandEnvs(t *testing.T) {
	tests := []struct {
		name            string
		envs            map[string]string
		manifest        []byte
		expectedErr     bool
		expectedCommand string
	}{
		{
			name: "expand envs on command",
			envs: map[string]string{
				"OKTETO_GIT_COMMIT": "dev",
			},
			manifest: []byte(`icon: https://apps.okteto.com/movies/icon.png
deploy:
  - okteto build -t okteto.dev/api:${OKTETO_GIT_COMMIT} api
  - okteto build -t okteto.dev/frontend:${OKTETO_GIT_COMMIT} frontend
  - helm upgrade --install movies chart --set tag=${OKTETO_GIT_COMMIT}
devs:
  - api/okteto.yml
  - frontend/okteto.yml`),
			expectedCommand: "okteto build -t okteto.dev/api:${OKTETO_GIT_COMMIT} api",
		},
		{
			name: "expand envs on command without env var set",
			envs: map[string]string{},
			manifest: []byte(`icon: https://apps.okteto.com/movies/icon.png
deploy:
  - okteto build -t okteto.dev/api:${OKTETO_GIT_COMMIT:=dev} api
  - okteto build -t okteto.dev/frontend:${OKTETO_GIT_COMMIT} frontend
  - helm upgrade --install movies chart --set tag=${OKTETO_GIT_COMMIT}
devs:
  - api/okteto.yml
  - frontend/okteto.yml`),
			expectedCommand: "okteto build -t okteto.dev/api:${OKTETO_GIT_COMMIT:=dev} api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envs {
				os.Setenv(k, v)
			}
			m, err := Read(tt.manifest)
			assert.NoError(t, err)

			err = m.ExpandEnvVars()
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				assert.Equal(t, tt.expectedCommand, m.Deploy.Commands[0].Command)
			}

		})
	}
}

func Test_validateDivert(t *testing.T) {
	tests := []struct {
		name        string
		divert      DivertDeploy
		expectedErr error
	}{
		{
			name: "divert-ok-with-port",
			divert: DivertDeploy{
				Namespace:  "namespace",
				Service:    "service",
				Port:       8080,
				Deployment: "deployment",
			},
			expectedErr: nil,
		},
		{
			name: "divert-ok-without-port",
			divert: DivertDeploy{
				Namespace:  "namespace",
				Service:    "service",
				Deployment: "deployment",
			},
			expectedErr: nil,
		},
		{
			name: "divert-ko-without-namespace",
			divert: DivertDeploy{
				Namespace:  "",
				Service:    "service",
				Port:       8080,
				Deployment: "deployment",
			},
			expectedErr: fmt.Errorf("the field 'deploy.divert.namespace' is mandatory"),
		},
		{
			name: "divert-ko-without-service",
			divert: DivertDeploy{
				Namespace:  "namespace",
				Service:    "",
				Port:       8080,
				Deployment: "deployment",
			},
			expectedErr: fmt.Errorf("the field 'deploy.divert.service' is mandatory"),
		},
		{
			name: "divert-ko-without-deployment",
			divert: DivertDeploy{
				Namespace:  "namespace",
				Service:    "service",
				Port:       8080,
				Deployment: "",
			},
			expectedErr: fmt.Errorf("the field 'deploy.divert.deployment' is mandatory"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manifest{
				Deploy: &DeployInfo{
					Divert: &tt.divert,
				},
			}
			assert.Equal(t, m.validateDivert(), tt.expectedErr)
		})
	}
}

func TestInferFromStack(t *testing.T) {
	dirtest := filepath.Clean("/stack/dir/")
	devInterface := PrivilegedLocalhost
	if runtime.GOOS == "windows" {
		devInterface = Localhost
	}
	stack := &Stack{
		Services: map[string]*Service{
			"test": {
				Build: &BuildInfo{
					Name:       "",
					Context:    "test",
					Dockerfile: "Dockerfile",
				},
				Ports: []Port{
					{
						HostPort:      8080,
						ContainerPort: 8080,
					},
				},
			},
		},
	}
	tests := []struct {
		name             string
		currentManifest  *Manifest
		expectedManifest *Manifest
	}{
		{
			name: "infer from stack empty dev",
			currentManifest: &Manifest{
				Dev:   ManifestDevs{},
				Build: ManifestBuild{},
				Deploy: &DeployInfo{
					ComposeSection: &ComposeSectionInfo{
						Stack: &Stack{
							Services: map[string]*Service{
								"test": {
									Build: &BuildInfo{
										Name:       "test",
										Context:    filepath.Join(dirtest, "test"),
										Dockerfile: filepath.Join(filepath.Join(dirtest, "test"), "Dockerfile"),
									},
									Ports: []Port{
										{
											HostPort:      8080,
											ContainerPort: 8080,
										},
									},
								},
							},
						},
					},
				},
			},
			expectedManifest: &Manifest{
				Build: ManifestBuild{
					"test": &BuildInfo{
						Context:    "test",
						Dockerfile: "Dockerfile",
					},
				},
				Dev: ManifestDevs{},
				Deploy: &DeployInfo{
					ComposeSection: &ComposeSectionInfo{
						Stack: stack,
					},
				},
			},
		},
		{
			name: "infer from stack not overriding build",
			currentManifest: &Manifest{
				Dev: ManifestDevs{},
				Build: ManifestBuild{
					"test": &BuildInfo{
						Context:    "test-1",
						Dockerfile: filepath.Join("test-1", "Dockerfile"),
					},
				},
				Deploy: &DeployInfo{
					ComposeSection: &ComposeSectionInfo{
						Stack: &Stack{
							Services: map[string]*Service{
								"test": {
									Build: &BuildInfo{
										Name:       "test",
										Context:    filepath.Join(dirtest, "test"),
										Dockerfile: filepath.Join(filepath.Join(dirtest, "test"), "Dockerfile"),
									},
									Ports: []Port{
										{
											HostPort:      8080,
											ContainerPort: 8080,
										},
									},
								},
							},
						},
					},
				},
			},
			expectedManifest: &Manifest{
				Build: ManifestBuild{
					"test": &BuildInfo{
						Context:    "test-1",
						Dockerfile: filepath.Join("test-1", "Dockerfile"),
					},
				},
				Dev: ManifestDevs{},
				Deploy: &DeployInfo{
					ComposeSection: &ComposeSectionInfo{
						Stack: &Stack{
							Services: map[string]*Service{
								"test": {
									Build: &BuildInfo{
										Name:       "test",
										Context:    "test",
										Dockerfile: "Dockerfile",
									},
									Ports: []Port{
										{
											HostPort:      8080,
											ContainerPort: 8080,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "infer from stack not overriding dev",
			currentManifest: &Manifest{
				Dev: ManifestDevs{
					"test": &Dev{
						Name:      "one",
						Namespace: "test",
					},
				},
				Build: ManifestBuild{},
				Deploy: &DeployInfo{
					ComposeSection: &ComposeSectionInfo{
						Stack: &Stack{
							Services: map[string]*Service{
								"test": {
									Build: &BuildInfo{
										Name:       "test",
										Context:    "test",
										Dockerfile: "Dockerfile",
									},
									Ports: []Port{
										{
											HostPort:      8080,
											ContainerPort: 8080,
										},
									},
								},
							},
						},
					},
				},
			},
			expectedManifest: &Manifest{
				Build: ManifestBuild{
					"test": &BuildInfo{
						Context:    "test",
						Dockerfile: "Dockerfile",
					},
				},
				Dev: ManifestDevs{
					"test": &Dev{
						Name:      "one",
						Namespace: "test",
						Metadata: &Metadata{
							Labels:      Labels{},
							Annotations: Annotations{},
						},
						Selector:   Selector{},
						EmptyImage: true,
						Image: &BuildInfo{
							Context:    ".",
							Dockerfile: "Dockerfile",
						},
						Push: &BuildInfo{
							Context:    ".",
							Dockerfile: "Dockerfile",
						},
						ImagePullPolicy: apiv1.PullAlways,
						InitContainer:   InitContainer{Image: OktetoBinImageTag},
						Probes:          &Probes{},
						Lifecycle:       &Lifecycle{},
						Workdir:         "/okteto",
						SecurityContext: &SecurityContext{
							RunAsUser:  pointer.Int64(0),
							RunAsGroup: pointer.Int64(0),
							FSGroup:    pointer.Int64(0),
						},
						SSHServerPort: 2222,
						Volumes:       []Volume{},
						Timeout: Timeout{
							Default:   60 * time.Second,
							Resources: 120 * time.Second,
						},
						Command: Command{
							Values: []string{"sh"},
						},
						Interface: devInterface,
						Sync: Sync{
							RescanInterval: 300,
							Folders: []SyncFolder{
								{
									LocalPath:  ".",
									RemotePath: "/okteto",
								},
							},
						},
					},
				},
				Deploy: &DeployInfo{
					ComposeSection: &ComposeSectionInfo{
						Stack: stack,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.currentManifest.InferFromStack(filepath.Clean(dirtest))
			if result != nil {
				for _, d := range result.Dev {
					d.parentSyncFolder = ""
				}
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedManifest, result)
		})
	}
}

func TestSetManifestDefaultsFromDev(t *testing.T) {
	os.Setenv("my_key", "my_value")
	tests := []struct {
		name              string
		currentManifest   *Manifest
		expectedContext   string
		expectedNamespace string
	}{
		{
			name: "setting only manifest.Namespace",
			currentManifest: &Manifest{
				Dev: ManifestDevs{
					"test": &Dev{
						Namespace: "other-ns",
					},
				},
			},
			expectedContext:   "",
			expectedNamespace: "other-ns",
		},
		{
			name: "setting only manifest.Context",
			currentManifest: &Manifest{
				Dev: ManifestDevs{
					"test": &Dev{
						Context: "other-ctx",
					},
				},
			},
			expectedContext:   "other-ctx",
			expectedNamespace: "",
		},
		{
			name: "setting manifest.Context & manifest.Namespace",
			currentManifest: &Manifest{
				Dev: ManifestDevs{
					"test": &Dev{
						Context:   "other-ctx",
						Namespace: "other-ns",
					},
				},
			},
			expectedContext:   "other-ctx",
			expectedNamespace: "other-ns",
		},
		{
			name: "not overwrite if manifest has more than one dev",
			currentManifest: &Manifest{
				Namespace: "test",
				Context:   "test",
				Dev: ManifestDevs{
					"test": &Dev{
						Context: "other-ctx",
					},
					"test-2": &Dev{
						Context: "other-ctx",
					},
				},
			},
			expectedContext:   "test",
			expectedNamespace: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.currentManifest.setManifestDefaultsFromDev()
			assert.Equal(t, tt.expectedContext, tt.currentManifest.Context)
			assert.Equal(t, tt.expectedNamespace, tt.currentManifest.Namespace)
		})
	}
}

func TestSetBuildDefaults(t *testing.T) {

	tests := []struct {
		name              string
		currentBuildInfo  BuildInfo
		expectedBuildInfo BuildInfo
	}{
		{
			name:             "all empty",
			currentBuildInfo: BuildInfo{},
			expectedBuildInfo: BuildInfo{
				Context:    ".",
				Dockerfile: "Dockerfile",
			},
		},
		{
			name: "context empty",
			currentBuildInfo: BuildInfo{
				Dockerfile: "Dockerfile",
			},
			expectedBuildInfo: BuildInfo{
				Context:    ".",
				Dockerfile: "Dockerfile",
			},
		},
		{
			name: "dockerfile empty",
			currentBuildInfo: BuildInfo{
				Context: "buildName",
			},
			expectedBuildInfo: BuildInfo{
				Context:    "buildName",
				Dockerfile: "Dockerfile",
			},
		},
		{
			name: "context and Dockerfile filled",
			currentBuildInfo: BuildInfo{
				Context:    "buildName",
				Dockerfile: "Dockerfile",
			},
			expectedBuildInfo: BuildInfo{
				Context:    "buildName",
				Dockerfile: "Dockerfile",
			},
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			tt.currentBuildInfo.setBuildDefaults()

			assert.Equal(t, tt.expectedBuildInfo, tt.currentBuildInfo)
		})
	}
}

func TestGetManifestFromFile(t *testing.T) {
	tests := []struct {
		name          string
		manifestBytes []byte
		composeBytes  []byte
		expectedErr   bool
	}{
		{
			name:          "OktetoManifest does not exist and compose manifest is correct",
			manifestBytes: nil,
			composeBytes: []byte(`services:
  test:
    image: test`),
			expectedErr: false,
		},
		{
			name:          "OktetoManifest not contains any content and compose manifest does not exists",
			manifestBytes: []byte(``),
			composeBytes:  nil,
			expectedErr:   true,
		},
		{
			name:          "OktetoManifest is invalid and compose manifest does not exists",
			manifestBytes: []byte(`asdasa: asda`),
			composeBytes:  nil,
			expectedErr:   true,
		},
		{
			name: "OktetoManifestV2 is ok",
			manifestBytes: []byte(`dev:
  api:
    sync:
    - .:/usr`),
			composeBytes: nil,
			expectedErr:  false,
		},
		{
			name: "OktetoManifestV1 is ok",
			manifestBytes: []byte(`name: test
sync:
- .:/usr`),
			composeBytes: nil,
			expectedErr:  false,
		},
		{
			name:          "OktetoManifest and compose manifest does not exists",
			manifestBytes: nil,
			composeBytes:  nil,
			expectedErr:   true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			file := ""
			if tt.manifestBytes != nil {
				file = filepath.Join(dir, "okteto.yml")
				assert.NoError(t, os.WriteFile(filepath.Join(dir, "okteto.yml"), tt.manifestBytes, 0644))
			}
			if tt.composeBytes != nil {
				if file == "" {
					file = filepath.Join(dir, "docker-compose.yml")
				}
				assert.NoError(t, os.WriteFile(filepath.Join(dir, "docker-compose.yml"), tt.composeBytes, 0644))
			}
			_, err := getManifestFromFile(dir, file)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

		})
	}
}
