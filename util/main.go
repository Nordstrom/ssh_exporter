//
// Copyright 2017 Nordstrom. All rights reserved.
//

//
// ssh_exporter.util provides helper functions and types for ssh_exporter.
//
package util

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
)

//
// Configuration file datastructure overview:
//
// version: v0
// scripts:
//   - name: 'name'
//     script: 'script'
//     timeout: 1s
//     credentials:
//     - host: 'host'
//       user: 'user'
//       keyfile: '/path/to/keyfile'
//
// Also includes internal data structures used to achieve a more sane
// data-flow.
//
type Config struct {
	Version string         `yaml:"version"`
	Scripts []ScriptConfig `yaml:"scripts"`
}

//
// ScriptConfig stores information about a given script including the name
// (which can be used to filter results on the /probe endpoint), Script,
// Timeout, Pattern (which determines if the script failed or not), and
// Credentials (described below).
//
// In addition to the above, ParsedTimeout is the go Duration of the user
// provided Timeout value and Ignored stores weather or not the script is run
// for a given request; it is ignored if Name does not match the query parameter
// "pattern"
//
type ScriptConfig struct {
	Name          string             `yaml:"name"`
	Script        string             `yaml:"script"`
	Timeout       string             `yaml:"timeout"`
	Pattern       string             `yaml:"pattern"`
	Credentials   []CredentialConfig `yaml:"credentials"`
	ParsedTimeout time.Duration      // For internal use only
	Ignored       bool               // For internal use only
}

//
// CredentialConfig stores information about each host a script is to be run
// on; at runtime the structure also stores the result of a given on that host.
//
// User assigned values include host, port, user, and keyfile.
//
// Runtime determined ScriptResult, ScriptReturnCode, ScriptError, and
// ResultPatternMatch
//
type CredentialConfig struct {
	Host               string `yaml:"host"`
	Port               string `yaml:"port"`
	User               string `yaml:"user"`
	KeyFile            string `yaml:"keyfile"`
	ScriptResult       string // For internal use only
	ScriptReturnCode   int    // For internal use only
	ScriptError        string // For internal use only
	ResultPatternMatch int8   // For internal use only
}

//
// FatalCheck exits the program if e is non-nil. Used for startup errors that
// should kill the server.
//
func FatalCheck(e error) {

	if e != nil {
		log.Fatal("error: ", e)
	}
}

//
// SoftCheck logs non-nil errors to stderr. Used for runtime errors that should
// not kill the server.
//
func SoftCheck(e error) bool {

	if e != nil {
		LogMsg(fmt.Sprintf("%v", e))
		return true
	} else {
		return false
	}
}

//
// LogMsg logs a string to stdout with timestamp.
//
func LogMsg(s string) {

	log.Printf("ssh_exporter :: %s", fmt.Sprintf("%s", s))
}

//
// ParseFlags parses the given commandline arguments and returns config and
// port as a tuple.
//
func ParseFlags(c, p *string) (*string, *string) {

	flag.StringVar(c, "config", "config.yml", "Path to your ssh_exporter config file")
	flag.StringVar(p, "port", "9428", "Port probed metrics are served on.")

	flag.Parse()

	return c, p
}

func ParseConfig(c string) (Config, error) {

	raw, err := ioutil.ReadFile(c)
	FatalCheck(err)

	initialConfig := Config{}
	err = yaml.Unmarshal([]byte(raw), &initialConfig)
	SoftCheck(err)

	finalConfig, err := adjustConfig(initialConfig)
	SoftCheck(err)

	return finalConfig, err
}

//
// ParseQuery parses HTTP query parameters for the 'pattern' query. Returns the
// compiled regex pattern or an error.
//
func ParseQuery(w http.ResponseWriter, r *http.Request) (*regexp.Regexp, error) {

	if r.URL.Query().Get("pattern") == "" {
		return nil, errors.New("Probe endpoint was hit, but pattern parameter was not passed.")
	}

	p, err := regexp.Compile(string(r.URL.Query().Get("pattern")))
	return p, err
}

//
// BatchExecute runs the scripts described in the provided configuration file.
//
// Conceptual overview (because this is a little complicated):
//
// A channel 't' is created as well as a sync.WaitGroup. These are used to
// communicate between goroutines and the main thread.
//
// The main thread spawns each goroutine and then waits with done.Wait().  In
// each goroutine the size of our sync.WaitGroup is incremented by 1. Once that
// thread is done executing it's assigned script, it calls done(), unblocking
// the WaitGroup.
//
// Once that stops blocking BatchExecute returns.
//
func BatchExecute(c *Config, p *regexp.Regexp) (Config, error) {

	var done sync.WaitGroup
	t := make(chan bool)

	for i, v := range c.Scripts {
		if p.MatchString(v.Name) != true {
			c.Scripts[i].Ignored = true
		} else {
			go executeScript(v.Script, v.Pattern, &c.Scripts[i].Credentials, &done, t)
		}
	}

	done.Wait()

	for _, v := range c.Scripts {
		if !v.Ignored {
			for _, _ = range v.Credentials {
				select {
				case <-time.After(v.ParsedTimeout):
				case <-t:
				}
			}
		}
	}

	return *c, nil
}

//
// PrometheusFormatResponse converts the config struct to a
// Prometheus-digestable format.
//
func PrometheusFormatResponse(c Config) (string, error) {

	var response string
	exitStatusFormatStr := "ssh_exporter_%s_exit_status{name=\"%s\",host=\"%s\",user=\"%s\",script=\"%s\",exit_status=\"%d\"} %d"
	patternMatchFormatStr := "ssh_exporter_%s_pattern_match{name=\"%s\",host=\"%s\",user=\"%s\",script=\"%s\",regex=\"%s\"} %d"

	exitStatusHelpStr := "# HELP ssh_exporter_%s_exit_status Integer exit status of commands and metadata about the command's execution.\n# TYPE ssh_exporter gauge"
	patternMatchHelpStr := "# HELP ssh_exporter_%s_pattern_match Boolean match of regex on output of script of commands and metadata about the command's execution.\n# TYPE ssh_exporter gauge"

	for _, i := range c.Scripts {
		if i.Ignored != true {
			exitedDoc := fmt.Sprintf(exitStatusHelpStr, i.Name)
			matchedDoc := fmt.Sprintf(patternMatchHelpStr, i.Name)

			response = fmt.Sprintf("%s%s", response, exitedDoc)
			for _, j := range i.Credentials {
				s := fmt.Sprintf(exitStatusFormatStr, i.Name, i.Name, j.Host, j.User, i.Script, j.ScriptReturnCode, j.ScriptReturnCode)
				response = fmt.Sprintf("%s\n%s", response, s)
			}
			response = fmt.Sprintf("%s\n%s", response, matchedDoc)
			for _, j := range i.Credentials {
				m := fmt.Sprintf(patternMatchFormatStr, i.Name, i.Name, j.Host, j.User, i.Script, i.Pattern, j.ResultPatternMatch)
				response = fmt.Sprintf("%s\n%s", response, m)
			}
			response = fmt.Sprintf("%s\n", response)
		}
	}

	return response, nil
}

//
// AdjustConfig makes small changes to ensure the config file provided is
// consistent.
//
func adjustConfig(c Config) (Config, error) {

	for c_i, v_i := range c.Scripts {
		for c_j, v_j := range v_i.Credentials {
			if v_j.Port == "" {
				c.Scripts[c_i].Credentials[c_j].Port = "22"
			}
		}

		tmp, err := time.ParseDuration(c.Scripts[c_i].Timeout)
		if !SoftCheck(err) {
			c.Scripts[c_i].ParsedTimeout = tmp
		} else {
			LogMsg(fmt.Sprintf("Failed to parse `timeout` for %s. Default to 10s", c.Scripts[c_i].Name))
			c.Scripts[c_i].ParsedTimeout, _ = time.ParseDuration("10s")
		}
	}

	return c, nil
}

//
// executeScript runs a given script on each assigned host, spawning a
// goroutine for each host in the CredentialConfig provided.
//
// TLDR executeScript runs the  given script in parallel on all hosts.
//
func executeScript(script, pattern string, creds *[]CredentialConfig, done *sync.WaitGroup, t chan bool) {

	match, _ := regexp.Compile(pattern)

	for i, c := range *creds {
		done.Add(1)
		go func() {
			result, status, err := executeScriptOnHost(c.Host, c.Port, c.User, c.KeyFile, script)

			(*creds)[i].ScriptReturnCode = status
			(*creds)[i].ScriptResult = result

			if err != nil {
				(*creds)[i].ScriptError = fmt.Sprintf("%v", err)
			}

			if match.MatchString(result) {
				(*creds)[i].ResultPatternMatch = 1
			} else {
				(*creds)[i].ResultPatternMatch = 0
			}

			t <- true

			done.Done()
		}()
	}
}

//
// executeScriptOnHost executes a given script on a given host.
//
func executeScriptOnHost(host, port, user, keyfile, script string) (string, int, error) {

	client, session, err := sshConnectToHost(host, port, user, keyfile)
	if SoftCheck(err) {
		return "", -1, err
	}

	out, err := session.CombinedOutput(script)
	if SoftCheck(err) {
		var errorStatusCode int
		fmt.Sscanf(fmt.Sprintf("%v", err), "Process exited with status %d", &errorStatusCode)
		if errorStatusCode != 0 {
			return "", errorStatusCode, err
		} else {
			return "", -1, err
		}
	}
	defer client.Close()
	defer session.Close()

	return literalFormat(string(out)), 0, nil

}

//
// sshConnectToHost connects to a given host with the given keyfile.
//
func sshConnectToHost(host, port, user, keyfile string) (*ssh.Client, *ssh.Session, error) {

	key, err := getKeyFile(keyfile)
	SoftCheck(err)

	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	sshConfig.SetDefaults()

	fullHost := fmt.Sprintf("%s:%s", host, port)
	client, err := ssh.Dial("tcp", fullHost, sshConfig)
	if err != nil {
		return nil, nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, nil, err
	}

	return client, session, nil
}

//
// getKeyFile provides an ssh.Signer for the given keyfile (path to a private key).
//
func getKeyFile(keyfile string) (ssh.Signer, error) {

	buf, err := ioutil.ReadFile(keyfile)
	SoftCheck(err)

	key, err := ssh.ParsePrivateKey(buf)
	SoftCheck(err)

	return key, nil
}

//
// literalFormat formats a string to be included in an endpoint to be scraped by Prometheus.
//
// Turns newline characters into '\n' characters.
//
func literalFormat(input string) string {

	return strings.Replace(input, "\n", "\\n", -1)
}
