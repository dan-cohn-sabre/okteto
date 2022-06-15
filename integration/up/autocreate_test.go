//go:build integration
// +build integration

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

package up

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"testing"

	"github.com/okteto/okteto/integration"
	"github.com/okteto/okteto/integration/commands"
	"github.com/okteto/okteto/pkg/model"
	"github.com/stretchr/testify/require"
)

const (
	autocreateManifest = `
name: autocreate
image: python:alpine
command:
  - sh
  - -c
  - "echo -n $VAR > var.html && python -m http.server 8080"
environment:
  - VAR=value1
workdir: /usr/src/app
sync:
- .:/usr/src/app
forward:
- 8080:8080
autocreate: true
`
	autocreateManifestV2 = `
dev:
  autocreate:
    image: python:alpine
    command:
    - sh
    - -c
    - "echo -n $VAR > var.html && python -m http.server 8080"
    environment:
    - VAR=value1
    workdir: /usr/src/app
    sync:
    - .:/usr/src/app
    forward:
    - 8081:8080
    autocreate: true
`
)

func TestUpAutocreate(t *testing.T) {
	t.Parallel()
	// Prepare environment
	dir := t.TempDir()
	oktetoPath, err := integration.GetOktetoPath()
	require.NoError(t, err)

	testNamespace := integration.GetTestNamespace("TestUpAutocreateV1", user)
	require.NoError(t, commands.RunOktetoCreateNamespace(oktetoPath, testNamespace))
	defer commands.RunOktetoDeleteNamespace(oktetoPath, testNamespace)

	indexPath := filepath.Join(dir, "index.html")
	require.NoError(t, writeFile(indexPath, testNamespace))
	log.Printf("original 'index.html' content: %s", testNamespace)

	require.NoError(t, writeFile(filepath.Join(dir, "okteto.yml"), autocreateManifest))
	require.NoError(t, writeFile(filepath.Join(dir, ".stignore"), "venv"))

	upOptions := &commands.UpOptions{
		Name:         "autocreate",
		Namespace:    testNamespace,
		Workdir:      dir,
		ManifestPath: filepath.Join(dir, "okteto.yml"),
	}
	upResult, err := commands.RunOktetoUp(oktetoPath, upOptions)
	require.NoError(t, err)

	require.NoError(t, integration.WaitForDeployment(kubectlBinary, testNamespace, model.DevCloneName("autocreate"), 1, timeout))

	varLocalEndpoint := "http://localhost:8080/var.html"
	indexLocalEndpoint := "http://localhost:8080/index.html"
	indexRemoteEndpoint := fmt.Sprintf("https://autocreate-%s.%s/index.html", testNamespace, appsSubdomain)

	// Test that environment variable is injected correctly
	require.Equal(t, integration.GetContentFromURL(varLocalEndpoint, timeout), "value1")

	// Test that the same content is on the remote and on local endpoint
	require.NotEmpty(t, integration.GetContentFromURL(indexLocalEndpoint, timeout))
	require.Equal(t, integration.GetContentFromURL(indexLocalEndpoint, timeout), testNamespace)
	require.Equal(t, integration.GetContentFromURL(indexLocalEndpoint, timeout), integration.GetContentFromURL(indexRemoteEndpoint, timeout))

	// Test that making a change gets reflected on remote
	localupdatedContent := fmt.Sprintf("%s-updated-content", testNamespace)
	require.NoError(t, writeFile(indexPath, localupdatedContent))
	require.NoError(t, waitUntilUpdatedContent(indexLocalEndpoint, localupdatedContent, timeout, upResult.ErrorChan))

	// Test that stignore has been created
	require.NoError(t, checkStignoreIsOnRemote(testNamespace, filepath.Join(dir, "okteto.yml"), oktetoPath))

	// Test kill syncthing reconnection
	require.NoError(t, killLocalSyncthing(upResult.Pid.Pid))
	localSyncthingKilledContent := fmt.Sprintf("%s-kill-syncthing", testNamespace)
	require.NoError(t, writeFile(indexPath, localSyncthingKilledContent))
	require.NoError(t, waitUntilUpdatedContent(indexLocalEndpoint, localSyncthingKilledContent, timeout, upResult.ErrorChan))

	// Test destroy pod reconnection
	require.NoError(t, integration.DestroyPod(context.Background(), testNamespace, "app=autocreate"))
	destroyPodContent := fmt.Sprintf("%s-destroy-pod", testNamespace)
	require.NoError(t, writeFile(indexPath, destroyPodContent))
	require.NoError(t, waitUntilUpdatedContent(indexLocalEndpoint, destroyPodContent, timeout, upResult.ErrorChan))

	// Test okteto down command
	downOpts := &commands.DownOptions{
		Namespace:    testNamespace,
		ManifestPath: upOptions.ManifestPath,
		Workdir:      dir,
	}
	require.NoError(t, commands.RunOktetoDown(oktetoPath, downOpts))

	require.True(t, commands.HasUpCommandFinished(upResult.Pid.Pid))
}

func TestUpAutocreateV2(t *testing.T) {
	t.Parallel()
	// Prepare environment
	dir := t.TempDir()
	oktetoPath, err := integration.GetOktetoPath()
	require.NoError(t, err)

	testNamespace := integration.GetTestNamespace("TestUpAutocreateV2", user)
	require.NoError(t, commands.RunOktetoCreateNamespace(oktetoPath, testNamespace))
	defer commands.RunOktetoDeleteNamespace(oktetoPath, testNamespace)

	indexPath := filepath.Join(dir, "index.html")
	require.NoError(t, writeFile(indexPath, testNamespace))
	log.Printf("original 'index.html' content: %s", testNamespace)

	require.NoError(t, writeFile(filepath.Join(dir, "okteto.yml"), autocreateManifestV2))
	require.NoError(t, writeFile(filepath.Join(dir, ".stignore"), "venv"))

	upOptions := &commands.UpOptions{
		Name:         "autocreate",
		Namespace:    testNamespace,
		Workdir:      dir,
		ManifestPath: filepath.Join(dir, "okteto.yml"),
	}
	upResult, err := commands.RunOktetoUp(oktetoPath, upOptions)
	require.NoError(t, err)

	require.NoError(t, integration.WaitForDeployment(kubectlBinary, testNamespace, model.DevCloneName("autocreate"), 1, timeout))

	varLocalEndpoint := "http://localhost:8081/var.html"
	indexLocalEndpoint := "http://localhost:8081/index.html"
	indexRemoteEndpoint := fmt.Sprintf("https://autocreate-%s.%s/index.html", testNamespace, appsSubdomain)

	// Test that environment variable is injected correctly
	require.NoError(t, waitUntilUpdatedContent(varLocalEndpoint, "value1", timeout, upResult.ErrorChan))

	// Test that the same content is on the remote and on local endpoint
	require.NotEmpty(t, integration.GetContentFromURL(indexLocalEndpoint, timeout))
	require.Equal(t, integration.GetContentFromURL(indexLocalEndpoint, timeout), testNamespace)
	require.Equal(t, integration.GetContentFromURL(indexLocalEndpoint, timeout), integration.GetContentFromURL(indexRemoteEndpoint, timeout))

	// Test that making a change gets reflected on remote
	localupdatedContent := fmt.Sprintf("%s-updated-content", testNamespace)
	require.NoError(t, writeFile(indexPath, localupdatedContent))
	require.NoError(t, waitUntilUpdatedContent(indexLocalEndpoint, localupdatedContent, timeout, upResult.ErrorChan))

	// Test that stignore has been created
	require.NoError(t, checkStignoreIsOnRemote(testNamespace, filepath.Join(dir, "okteto.yml"), oktetoPath))

	// Test kill syncthing reconnection
	require.NoError(t, killLocalSyncthing(upResult.Pid.Pid))
	localSyncthingKilledContent := fmt.Sprintf("%s-kill-syncthing", testNamespace)
	require.NoError(t, writeFile(indexPath, localSyncthingKilledContent))
	require.NoError(t, waitUntilUpdatedContent(indexLocalEndpoint, localSyncthingKilledContent, timeout, upResult.ErrorChan))

	// Test destroy pod reconnection
	require.NoError(t, integration.DestroyPod(context.Background(), testNamespace, "app=autocreate"))
	destroyPodContent := fmt.Sprintf("%s-destroy-pod", testNamespace)
	require.NoError(t, writeFile(indexPath, destroyPodContent))
	require.NoError(t, waitUntilUpdatedContent(indexLocalEndpoint, destroyPodContent, timeout, upResult.ErrorChan))

	// Test okteto down command
	downOpts := &commands.DownOptions{
		Namespace:    testNamespace,
		ManifestPath: upOptions.ManifestPath,
		Workdir:      dir,
	}
	require.NoError(t, commands.RunOktetoDown(oktetoPath, downOpts))

	require.True(t, commands.HasUpCommandFinished(upResult.Pid.Pid))
}
