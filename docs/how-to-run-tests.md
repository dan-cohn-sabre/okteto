# Okteto CLI tests

On this document we will cover how to run unit tests and e2e tests locally

- [Okteto CLI tests](#okteto-cli-tests)
  - [How to run unit tests locally?](#how-to-run-unit-tests-locally)
    - [Requirements:](#requirements)
    - [Run all unit tests](#run-all-unit-tests)
    - [Run package tests](#run-package-tests)
    - [Run specific test](#run-specific-test)
  - [How to run e2e tests locally?](#how-to-run-e2e-tests-locally)
    - [Requirements:](#requirements-1)
    - [Run all e2e tests](#run-all-e2e-tests)
    - [Run specific e2e tests](#run-specific-e2e-tests)

## How to run unit tests locally?

Unit test will run against the code you have on your workspace

### Requirements:

You don't need to have any special prerrequisite to run unit tests locally.

### Run all unit tests

You can run all tests by running the following command:

``` bash
make test
```

### Run package tests

You can run all tests by running the following command:

``` bash
go test packageName
# for example
go test github.com/okteto/okteto/cmd/deploy
```

### Run specific test

You can run all tests by running the following command:

``` bash
go test -run testRegex packageName
# for example
go test -run ^(TestDeployWithErrorChangingKubeConfig)$ github.com/okteto/okteto/cmd/deploy
```

## How to run e2e tests locally?

Unit tests will run against a okteto cluster that you must be logged in

### Requirements:

You will need to set some environment variables to start running e2e tests

- `OKTETO_USER`: This is your okteto username. For example: `cindylopez`
- `OKTETO_PATH`: The path of the okteto binary (It will default to `/usr/bin/okteto`).
- `OKTETO_APPS_SUBDOMAIN`: The subdomain of the okteto cluster. For example: `cloud.okteto.net`

### Run all e2e tests

You can run all tests by running the following command:

``` bash
make integration
```

### Run specific e2e tests

There are different e2e tests that can be run individually:

- Run actions: Run all e2e tests for actions

``` bash
    make integration-actions # which is equivalent to run go test github.com/okteto/okteto/integration/actions -tags="actions" --count=1 -v -timeout 10m
```

- Run build: Run all e2e tests that builds

``` bash
    make integration-build # which is equivalent to run go test github.com/okteto/okteto/integration/build -tags="integration" --count=1 -v -timeout 10m
```

- Run deploy: Run all e2e tests that deploys

``` bash
    make integration-deploy # which is equivalent to run go test github.com/okteto/okteto/integration/deploy -tags="integration" --count=1 -v -timeout 20m
```

- Run okteto: Run all e2e tests that are only valid on okteto clusters

``` bash
    make integration-okteto # which is equivalent to run go test github.com/okteto/okteto/integration/okteto -tags="integration" --count=1 -v -timeout 30m
```

- Run up: Run all e2e tests that uses okteto up as main command

``` bash
    make integration-up # which is equivalent to run go test github.com/okteto/okteto/integration/up -tags="integration" --count=1 -v -timeout 45m
```

- Run okteto: Run all e2e tests that are deprecated

``` bash
    make integration-deprecated # which is equivalent to run go test github.com/okteto/okteto/integration/deprecated/push -tags="integration" --count=1 -v -timeout 15m && go test github.com/okteto/okteto/integration/deprecated/stack -tags="integration" --count=1 -v -timeout 15m
```
