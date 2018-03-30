package main

//
// Copyright 2017 Nordstrom. All rights reserved.
//

//
// Provides Unit and Integration tests for ssh_exporter.go
//
// TODO: More tests! Always more tests.
//

import (
	"github.com/Nordstrom/ssh_exporter/util"

	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"gopkg.in/yaml.v2"
)

const (
	// Used to parse a config and run integration tests
	config = "test/config.yml"

	// used to execute the local server
	binary = "./ssh_exporter"

	// Used to connect to the local server
	address = "localhost:9428"
)

//
// Compares a string and a file.
// They should be identical.
//
func compare(computed, file_path string, t *testing.T) bool {

	data, err := ioutil.ReadFile(file_path)
	if err != nil {
		t.Errorf("Error opening %s: %s", file_path, err)
	}

	expected := string(data)

	if expected != computed {
		t.Errorf("Expected output did not match computed output:\nExpected:\n%sGot:\n%s", expected, computed)
		return false
	}
	return true
}

//
// Tests that the config parser is working correctly.
//
// This is done because ParseConfig does more than just yaml.Unmarshal,
// it also adds internal fields.
//
// Should produce a marshalled config similar to `test/config.yml` with some additional fields.
//
func TestUnitParseConfig(t *testing.T) {

	fmt.Println("Running TestUnitParseConfig")

	// Ensure the config file exist before continuing
	if _, err := os.Stat(config); err != nil {
		t.Errorf("%s config not avaliable, add it to continue first: %s", config, err)
		t.Fail()
	}

	// Parse the config
	conf, err := util.ParseConfig(config)
	if err != nil {
		t.Errorf("There was an error parsing config %s: %s", config, err)
		t.Fail()
	}

	// Marshal the new config, should include more fields
	marshalled_conf, err := yaml.Marshal(&conf)
	if err != nil {
		t.Errorf("Error Marshaling loaded config file: %s", err)
		t.Fail()
	}

	// Compare to the test's source of truth
	compare(string(marshalled_conf), "test/parse_config.unit.txt", t)
}

//
// Tests that the we're able to output Prometheus data correctly.
//
// Should produce a string similar to the HTTP endpoint result.
//
func TestUnitPrometheusFormatResponse(t *testing.T) {

	fmt.Println("Running TestUnitPrometheusFormatResponse")

	parsedTime, _ := time.ParseDuration("1s")

	prom_conf := util.Config{
		Version: "v0",
		Scripts: []util.ScriptConfig{
			util.ScriptConfig{
				Name:    "scriptName",
				Script:  "echo foo",
				Timeout: "5s",
				Pattern: "foo",
				Credentials: []util.CredentialConfig{
					util.CredentialConfig{
						Host:               "localhost",
						Port:               "2222",
						User:               "username",
						KeyFile:            "/noop",
						ScriptResult:       "foo",
						ScriptReturnCode:   0,
						ScriptError:        "",
						ResultPatternMatch: 1,
					},
				},
				ParsedTimeout: parsedTime,
				Ignored:       false,
			},
		},
	}

	// PrometheusFormatResponse formats the output based on the modified Config
	output, err := util.PrometheusFormatResponse(prom_conf)
	if err != nil {
		t.Errorf("Error formatting output for Prometheus: %s", err)
		t.Fail()
	}

	// Compare the Prometheus formatted output we expect vs what we actually got
	compare(string(output), "test/prometheus_format.unit.txt", t)
}

//
// Simple integration test, ensuring the 'happy path' works
//
// NOTE: This requires a host to run tests on.
// A host is provided via Vagrant for local testing, however the host used for integration tests can be changed by editing `test/config.yml`.
//
func TestIntegrationHappyPath(t *testing.T) {

	fmt.Println("Running TestIntegrationHappyPath")

	// Make sure we have a binary to run
	if _, err := os.Stat(binary); err != nil {
		t.Errorf("%s binary not available, try to run `go build` first: %s", binary, err)
		t.Fail()
	}
	// Make sure we have a config to read
	if _, err := os.Stat(config); err != nil {
		t.Errorf("%s config not available, add it to run integration tests: %s", config, err)
		t.Fail()
	}

	// Run the exporter locally
	cmd := exec.Command(binary, "--config", config)
	cmd.Stdout = os.Stdout
	err := cmd.Start()
	if err != nil {
		t.Errorf("Failed to start exporter: %s", err)
		t.Fail()
	}

	// Wait for the exporter to startup
	select {
	case <-time.After(100 * time.Millisecond):
	}

	// Fetch the default "all" pattern for the metrics
	resp, err := http.Get(fmt.Sprintf("http://%s/probe?pattern=.*", address))
	if err != nil {
		t.Errorf("Error fetching endpoint: %s\nIs the integration test host running?", err)
		t.Fail()
	}

	// Read the body into a bytes variable
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Error parsing response body: %s", err)
		t.Fail()
	}

	// Close the response body
	if err := resp.Body.Close(); err != nil {
		t.Errorf("Error closing body: %s", err)
		t.Fail()
	}

	// Make sure the status is correct
	// If this fails we have weirder problems
	if want, have := http.StatusOK, resp.StatusCode; want != have {
		t.Errorf("Status code was not OK: %s != %s\n%s", want, have, string(data))
		t.Fail()
	}

	// Compare the body with what we want the body to be
	if !compare(string(data), "test/happy_path.integration.txt", t) {
		t.Error("Is the integration host running?")
		t.Fail()
	}
}
