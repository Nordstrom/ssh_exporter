package main

//
// Copyright 2017 Nordstrom. All rights reserved.
//

//
// Provides an HTTP endpoint to be consumed by Prometheus
// which hosts pre-configured statistics found in config.yaml.
//
// Default endpoint: http://localhost:9382/probe?pattern=.*
//

import (
	"github.com/Nordstrom/ssh_exporter/util"

	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//
// Global variables
//
var configPath string
var servePort string
var loggingEnabled bool
var patternHelpText = `<p>Please include a valid <code>?pattern=[regex]</code>
query parameter in your URL. This should match the <bold>name</bold> of the
scripts you want to run (e.g., <code>?pattern=.*logs</code> matches
<code>chef_logs</code> and not <code>proc_status</code>)</p>.`

//
// Main fetches configuration file and assigns handlers for the http server.
//
func main() {

	util.ParseFlags(&configPath, &servePort)

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/probe", probeHandler)
	http.Handle("/metrics", promhttp.Handler())

	util.LogMsg(fmt.Sprintf("Listening on localhost:%s", servePort))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", servePort), nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {

	// Human readable navigation help.
	response := `<h1>ssh exporter</h1>
		<p><a href='/probe'>probe</a></p>
		<p><a href='/metrics'>metrics</a></p>`

	fmt.Fprintf(w, response)
}

func probeHandler(w http.ResponseWriter, r *http.Request) {

	conf, err := util.ParseConfig(configPath)
	util.SoftCheck(err)

	pattern, err := util.ParseQuery(w, r)
	if util.SoftCheck(err) {
		fmt.Fprintf(w, patternHelpText)
	} else {
		util.BatchExecute(&conf, pattern)

		response, _ := util.PrometheusFormatResponse(conf)

		fmt.Fprintf(w, response)
	}
}
