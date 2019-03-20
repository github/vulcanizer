# vulcanizer

GitHub's ops focused Elasticsearch library

[![build status](https://travis-ci.org/github/vulcanizer.svg)](https://travis-ci.org/github/vulcanizer) [![GoDoc](https://godoc.org/github.com/github/vulcanizer?status.svg)](https://godoc.org/github.com/github/vulcanizer) [![Go Report Card](https://goreportcard.com/badge/github.com/github/vulcanizer)](https://goreportcard.com/report/github.com/github/vulcanizer) [![release](https://img.shields.io/github/release/github/vulcanizer.svg)](https://github.com/github/vulcanizer/releases)

This project is a golang library for interacting with an Elasticsearch cluster. It's goal is to provide a high level API to help with common tasks that are associated with operating an Elasticsearch cluster such as querying health status of the cluster, migrating data off of nodes, updating cluster settings, etc.

This project does not aim to be a fully-featured API client for querying or indexing to Elasticsearch.

### Go API

You can perform custom operations in your Go application.

```go
import "github.com/github/vulcanizer"

v = vulcanizer.NewClient("localhost", 9200)
oldSetting, newSetting, err := v.SetSetting("indices.recovery.max_bytes_per_sec", "1000mb")
```

### Command line application

This project produces a `vulcanizer` binary that is a command line application that can be used to manage your Elasticsearch cluster.

```
$ vulcanizer -h
Usage:
  vulcanizer [command]

Available Commands:
  allocation  Set shard allocation on the cluster.
  drain       Drain a server or see what servers are draining.
  fill        Fill servers with data, removing shard allocation exclusion rules.
  health      Display the health of the cluster.
  help        Help about any command
  indices     Display the indices of the cluster.
  nodes       Display the nodes of the cluster.
  repository  Interact with the configured snapshot repositories.
  setting     Interact with cluster settings.
  settings    Display all the settings of the cluster.
  shards      Get shard data by cluster node(s).
  snapshot    Interact with a specific snapshot.

Flags:
  -c, --cluster string      Cluster to connect to defined in config file
  -f, --configFile string   Configuration file to read in (default to "~/.vulcanizer.yaml")
  -h, --help                help for vulcanizer
      --host string         Host to connect to (default "localhost")
  -p, --port int            Port to connect to (default 9200)

Use "vulcanizer [command] --help" for more information about a command.
```

#### Roadmap and future releases

The proposed future for vulcanizer can be found in our [ROADMAP](ROADMAP.md).


#### Configuration and connection information 

All commands take `--cluster <name>` to look up information in a configuration file in `~/.vulcanizer.yaml`. The configuration should be in the form of 

```yml
local:
  host: localhost
  port: 9200
staging:
  host: 10.10.2.1
  port: 9201
production:
  host: 10.10.1.1
  port: 9202
```

Alternatively, all commands take `--host` and `--port` for the connection information.

For example:

```
# Query for cluster health on the "local" cluster
vulcanizer health --cluster local

# Query for nodes against the node 10.10.2.1 and port 9202
vulcanizer nodes --host 10.10.2.1 --port 9202
```

### Development

`./script/build` will compile the project and install the `vulcanizer` binary to `$GOPATH/bin`.

`./script/test` will run the tests in the project.

### Supported Elasticsearch versions

Integration tests are set up to run against the latest v5 and v6 versions of Elasticsearch.

### Name

[Vulcanization](https://en.wikipedia.org/wiki/Vulcanization) is the process of making rubber more elastic, so vulcanizer is the library that makes Elasticsearch easier to work with!

### Project status

This project is under active development.

### Contributing

This repository is [open to contributions](CONTRIBUTING.md). Please also see [code of conduct](CODE_OF_CONDUCT.md)

To get up and running, install the project into your $GOPATH and run the set up scripts.

```
go get github.com/github/vulcanizer

cd $GOPATH/src/github.com/github/vulcanizer

./script/bootstrap
./script/test
```

And the test suite should execute correctly.

### License

This project is released under the [MIT LICENSE](LICENSE). Please note it includes 3rd party dependencies release under their own licenses; these are found under [vendor](https://github.com/github/vulcanizer/tree/master/vendor). When using the GitHub logos, be sure to follow the [GitHub logo guidelines](https://github.com/logos).

### Authors

Authored by GitHub Engineering
