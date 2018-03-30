# Prometheus ssh exporter

The ssh exporter is a [Prometheus exporter][prom-exporter] developed by [Nordstorm][nord-gh] for running ssh commands on remote hosts and collecting statistics about the output of those commands.

## Use with caution

*This tool was built for very specific edge case applications where you need to quickly get the results of some existing test script into Prometheus and existing exporters are not flexible enough.*
*Before deciding to use this exporter, consider using a more specialized exporter insted.*

Any time you're executing arbitrary code on a host you should be careful.

Double check that your commands are not liable to crash your systems, especially considering that **the commands will be run in parallel ssh connections**.

## Usage

### Pre-requisites

You'll need a go-lang environment to build the `ssh_exporter` binary as well as the following go imports:

```go
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

	"gopkg.in/yaml.v2"
	"golang.org/x/crypto/ssh"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)
```

### Building the ssh exporter

Clone the repository via go ```go get github.com/Nordstrom/ssh_exporter``` or git (if cloning the repo by hand you will
have to update your $GOPATH) and `cd` into the directory.
Then build the `ssh_exporter` binary with the following commands.

```
$ go build
```

If any packages are not installed, use `go get` to download them.

### Configuration Options

#### \<version\>

The version of the config file format. Currently supports one value
```v0```

#### \<scripts\>

A list of scripts which might be executed by the exporter.

##### \<name\>

A name for the script to be executed. This is what is matched by the pattern URL variable.
For example with the followig config:

```yaml
version: v0
scripts:
  - name: echo_output
    script: echo "output script"
    timeout: 5s
    pattern: 'output [matches|does not match] a regex'
    credentials:
    - host: myhost.example.ext
      port: 22
      user: someuser
      keyfile: /path/to/private/key
    - host: second.host.example.net
  - name: ls_var_tmp
    script: ls /var/tmp
    ...
```

This request:
```curl http://localhost:9428/probe?pattern=echo_output```

Would execute
```echo "output script"```

And this request:
```curl http://localhost:9428/probe?pattern=ls_var_tmp```

Would likewise execute
```ls /var/tmp```

##### \<script\>

The script to execute on the remote host

##### \<timeout\>

How long to wait for the command to complete.

##### \<pattern\>

A regex pattern to match against the command output. The normal model for scraping with Prometheus
is to have the endpoint being scraped return statistical data which is stored rather than return a 
pass/fail status. Then alerts or reports can be generated against that data. ssh_exporter is
intended for edge case applications where you need to quickly get the results of some existing
test into Prometheus. It is intended to aid organizations who are migrating from some other monitoring 
solution to Prometheus. And since Prometheus stores numeric data and not text results the ssh_exporter
compares \<pattern\> against the output of the command and returns a true or false.

##### \<credentials\>

A list of endpoints upon which the command specified in \<script\> will be executed.

###### \<host\>

The host name or IP address upon which to run the test.

###### \<port\>

The port upon which an ssh daemon is running on the remote host.

###### \<user\>

The user to connect as and run the command.

###### \<keyfile\>

The ssh private key to use for authentication.

NOTE: ssh_exporter currently only supports private keys with no passphrase.

#### Example

**ssh_exporter config**
```yaml
version: v0
scripts:
  - name: echo_output
    script: echo "This is my output!"
    timeout: 5s
    pattern: '.*output!'
    credentials:
    - host: host1.example.com
      port: 22
      user: someuser
      keyfile: /path/to/private/key
  - name: check_var_temp_for_tars
    script: ls /var/tmp
    timeout: 5s
    pattern: '.*tgz'
    credentials:
    - host: myhost.example.com
      port: 22
      user: someuser
      keyfile: /path/to/private/key
    - host: host2.example.com
      port: 22
      user: someotheruser
      keyfile: /path/to/other/private/key
```

**Prometheus config**

```
scrape_configs:
  - job_name: 'ssh_exporter_check_output'
    static_configs:
      - targets: ['localhost:9428']
    metrics_path: /probe
    params:
      pattern: ['echo_output']

  - job_name: 'ssh_exporter_check_var_tmp'
    static_configs:
      - targets: ['localhost:9428']
    metrics_path: /probe
    params:
      pattern: ['check_var_temp_for_tars']
```

### Running

The config allows one to specify a list of scripts (with timeouts and match patterns) and a list of hosts to run that script on.
Scripts are run in parallel with concurrent ssh connections on all configured hosts.

The default configuration file path is `./config.yml`.
The `--config` flag overrides this option.

The default port `ssh_exporter` hosts its data on is `9428`; the `--port` flag overrides this option.

After you have created a config file, start the endpoint server:

```
$ ./ssh_exporter/ssh_exporter --port=8888 --config=custom_config.yaml
```

This will start the web server on `localhost:8888`.

- `localhost:8888/`: a human readable navigation page
- `localhost:8888/probe?pattern=<regex-matcher-for-script-names>`: statistics based on the scripts in the configuration file
- `localhost:8888/metrics`: meta-statics about the app itself.

### Prometheus Configuration

```
scrape_configs:
  - job_name: 'ssh_exporter'
    static_configs:
      - targets: ['localhost:9428']
    metrics_path: /probe
    params:
      pattern: ['.*']
```


## Contributing

There's a lot of work that can be done on the ssh exporter.

If you find an issue with ssh exporter don't want to make the changes yourself, search for the problem on the repos issues page.
If the issue or feature request is undocumented, make a new issue.

If you would like to contribute code or documentation, follow these steps:

1. Clone a local copy.
2. Make your changes on a uniquely named branch.
3. Comment those changes.
4. Test those changes (do as we say not as we do).
5. Push your branch to a fork and create a Pull Request.

## Testing

The tests are split into `Unit` and `Integration`.

### Unit tests

To run just the `Unit` tests run the following command:

```
$ go test -run 'Unit'
```

The unit tests require the file `test/config.yml` to exist at the base of the repository.

### Integration tests

The Integration tests require a host to run scripts on
A `Vagrantfile` has been provided for you to spin up and quickly use.
To use this first install [Vagrant][vagrant] and [Virtualbox][vbox] on your local host.

To run just the `Integration` tests, first spin up the host on which to run the scripts.
If you are using the provided Vagrant host, first run the following:

```
$ vagrant up
```

This will take a while.

If you would like to configure a *different* host to run the integration tests on, edit `test/config.yml` to reflect the changed:

- Host
- Port
- Username
- Keyfile

**WARNING** Make sure not to `git commit` any changes to `test/config.yml`.

Once you have configured / spun-up the testing host, run the integration tests with the following:

```
$ go test -run 'Integration'
```

To destroy the local vagrant host run the following:

```
$ vagrant destroy   # Respond yes at the prompt
```

### All tests

For all tests, run the following:

```
$ go test
```

## Future work

Some improvements that come to mind:

- Safeguards ought to be implemented on the commands being run (beyond just timeout).
- Addition of `script_files` to (more easily) run multi-command scripts.
- Tests! Figure out a better integration test method.

## Author

Nordstrom, Inc.

## License

Copyright 2017 Nordstrom, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

[prom-exporter]: https://prometheus.io/docs/instrumenting/exporters/
[nord-gh]: https://github.com/Nordstrom
[vbox]: https://www.virtualbox.org/
[vagrant]: https://vagrantup.com
[caution]: #use-with-caution
