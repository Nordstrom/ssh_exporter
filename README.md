# Prometheus ssh exporter

The ssh exporter is a [Prometheus exporter][prom-exporter] developed by [Nordstorm][nord-gh] for running ssh commands on remote hosts and collecting statistics about the output of those commands.

*This tool was built for very specific use-cases when the snmp_exporter, node_exporter, and the Prometheus pushgateway couldn't cut it.*
*Before deciding to use this exporter, consider using a more specialized exporter insted.*
*For more about this, check out the [Use with caution][caution] section of this README.*

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

In a Unix terminal clone this repo and `cd` into the directory.
Then build the `ssh_exporter` binary with the following commands.

```
$ go build
```

If any packages are not installed, use `go get` to download them.

### Configuration and execution

To start the exporter, first create a config file with the following format:

```yaml
version: v0
scripts:
  - name: name_for_pattern_query
    script: echo "output script"
    timeout: 5s
    pattern: 'output [matches|does not match] a regex'
    credentials:
    - host: myhost.example.ext
      port: 22
      user: someuser
      keyfile: /path/to/private/key
    - host: second.host.example.net
      ...
  - name: other_query
    ...
```

The config allows one to specify a list of scripts (with timeouts and match patterns) and a list of hosts to run that script on.
Scripts are run in parallel with concurrent ssh connections on all configured hosts.

The default configuration file path is `./config.yml`.
The `--config` flag overrides this option.

The default port `ssh_exporter` hosts its data on is `9382`; the `--port` flag overrides this option.

After you have created a config file, start the endpoint server:

```
$ ./ssh_exporter/ssh_exporter --port=8888 --config=custom_config.yaml
```

This will start the web server on `localhost:8888`.

- `localhost:8888/`: a human readable navigation page
- `localhost:8888/probe?pattern=<regex-matcher-for-script-names>`: statistics based on the scripts in the configuration file
- `localhost:8888/metrics`: meta-statics about the app itself.

## Use with caution

Any time you're executing arbitrary code on a host you should be careful.

Double check that your commands are not liable to crash your systems, especially considering that **the commands will be run in parallel ssh connections**.

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
