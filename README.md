# ssh_exporter

ssh_exporter is a Prometheus exporter for running ssh commands on remote hosts.

## Usage

### Building the exporter

In a Unix terminal

```
$ pushd ssh_exporter
$ go build .
$ popd
```

If any packages are not installed use `go get` to download them.

### Configuring and starting the exporter

To start the server first create a config file with the following format:

```yaml
version: v0
scripts:
  - name: name_for_pattern_query
    script: echo "output script"
    timeout: 5s
    pattern: 'output [must|match|script]'
    credentials:
    - host: myhost.example.ext
      port: 22
      user: someuser
      keyfile: /path/to/private/key
```

The default configuration file path is `config.yml`; the `--config` flag overrides this option.

The default port is `8007`; the `--port` flag overrides this option.

After you have created a config file, start the endpoint server:

```
$ ./ssh_exporter/ssh_exporter --port 8888 --config custom_config.yaml
```

This will start the web server on `localhost:8888`.
- `localhost:8888/` provides a human readable navigation page
- `localhost:8888/probe?pattern=<regex>` provides statistics based on the scripts in the configuration file
- `localhost:8888/metrics` provides meta-statics about the app itself.

## Contributing

There's a lot of work that can be done on ssh_exporter.

To contribute:

1. Clone a local copy.
2. Make your changes on a uniquely named branch.
3. Comment those changes.
4. Test those changes (do as I say not as I do).
5. Push your branch to a fork and create a Pull Request.

## License

TBD
