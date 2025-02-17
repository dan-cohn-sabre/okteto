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

package okteto

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/okteto/okteto/integration"
	"github.com/okteto/okteto/integration/commands"
	"github.com/stretchr/testify/require"

	"github.com/okteto/okteto/pkg/k8s/kubeconfig"
	oktetoLog "github.com/okteto/okteto/pkg/log"
	"github.com/okteto/okteto/pkg/okteto"
)

const (
	composeTemplate = `services:
  app:
    build: app
    command: echo -n $RABBITMQ_PASS > var.html && python -m http.server 8080
    ports:
    - 8080
  nginx:
    image: nginx
    volumes:
    - ./nginx/nginx.conf:/tmp/nginx.conf
    command: /bin/bash -c "envsubst < /tmp/nginx.conf > /etc/nginx/conf.d/default.conf && nginx -g 'daemon off;'"
    environment:
    - FLASK_SERVER_ADDR=app:8080
    ports:
    - 80:80
    depends_on:
      app:
        condition: service_started
    container_name: web-svc
    healthcheck:
      test: service nginx status || exit 1
      interval: 45s
      timeout: 5m
      retries: 5
      start_period: 30s`
	appDockerfile = "FROM python:alpine"
	nginxConf     = `server {
  listen 80;
  location / {
    proxy_pass http://$FLASK_SERVER_ADDR;
  }
}`
)

func TestDeployOutput(t *testing.T) {
	integration.SkipIfNotOktetoCluster(t)
	t.Parallel()
	oktetoPath, err := integration.GetOktetoPath()
	require.NoError(t, err)

	dir := t.TempDir()
	require.NoError(t, createComposeScenario(dir))

	testNamespace := integration.GetTestNamespace("TestDeploy", user)
	namespaceOpts := &commands.NamespaceOptions{
		Namespace:  testNamespace,
		OktetoHome: dir,
		Token:      token,
	}
	require.NoError(t, commands.RunOktetoCreateNamespace(oktetoPath, namespaceOpts))
	require.NoError(t, commands.RunOktetoKubeconfig(oktetoPath, dir))
	defer commands.RunOktetoDeleteNamespace(oktetoPath, namespaceOpts)

	deployOptions := &commands.DeployOptions{
		Workdir:    dir,
		Namespace:  testNamespace,
		OktetoHome: dir,
		Token:      token,
	}
	require.NoError(t, commands.RunOktetoDeploy(oktetoPath, deployOptions))

	c, _, err := okteto.NewK8sClientProvider().Provide(kubeconfig.Get([]string{filepath.Join(dir, ".kube", "config")}))
	require.NoError(t, err)
	cmap, err := integration.GetConfigmap(context.Background(), testNamespace, fmt.Sprintf("okteto-git-%s", filepath.Base(dir)), c)
	require.NoError(t, err)

	uiOutput, err := base64.StdEncoding.DecodeString(cmap.Data["output"])
	require.NoError(t, err)

	var text oktetoLog.JSONLogFormat
	stageLines := map[string][]string{}
	prevLine := ""
	for _, l := range strings.Split(string(uiOutput), "\n") {
		if err := json.Unmarshal([]byte(l), &text); err != nil {
			if prevLine != "EOF" {
				t.Fatalf("not json format: %s", l)
			}
		}
		if _, ok := stageLines[text.Stage]; ok {
			stageLines[text.Stage] = append(stageLines[text.Stage], text.Message)
		} else {
			stageLines[text.Stage] = []string{text.Message}
		}
		prevLine = text.Message
	}

	stagesToTest := []string{"Load manifest", "Building service app", "Deploying compose", "done"}
	for _, ss := range stagesToTest {
		if _, ok := stageLines[ss]; !ok {
			t.Fatalf("deploy didn't have the stage '%s'", ss)
		}
		if strings.HasPrefix(ss, "Building service") {
			if len(stageLines[ss]) < 5 {
				t.Fatalf("Not sending build output on stage %s. Output:%s", ss, stageLines[ss])
			}
		}

	}
}

func createComposeScenario(dir string) error {
	if err := os.Mkdir(filepath.Join(dir, "nginx"), 0700); err != nil {
		return err
	}

	nginxPath := filepath.Join(dir, "nginx", "nginx.conf")
	nginxContent := []byte(nginxConf)
	if err := os.WriteFile(nginxPath, nginxContent, 0644); err != nil {
		return err
	}

	if err := createAppDockerfile(dir); err != nil {
		return err
	}

	composePath := filepath.Join(dir, "docker-compose.yml")
	composeContent := []byte(composeTemplate)
	if err := os.WriteFile(composePath, composeContent, 0644); err != nil {
		return err
	}

	return nil
}

func createAppDockerfile(dir string) error {
	if err := os.Mkdir(filepath.Join(dir, "app"), 0700); err != nil {
		return err
	}

	appDockerfilePath := filepath.Join(dir, "app", "Dockerfile")
	appDockerfileContent := []byte(appDockerfile)
	if err := os.WriteFile(appDockerfilePath, appDockerfileContent, 0644); err != nil {
		return err
	}
	return nil
}
