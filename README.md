# vulcanizer

GitHub's ops focused Elasticsearch library

[![build status](https://github.com/github/vulcanizer/workflows/Vulcanizer%20CI/badge.svg)](https://github.com/github/vulcanizer/actions) [![GoDoc](https://godoc.org/github.com/github/vulcanizer?status.svg)](https://godoc.org/github.com/github/vulcanizer) [![Go Report Card](https://goreportcard.com/badge/github.com/github/vulcanizer)](https://goreportcard.com/report/github.com/github/vulcanizer) [![release](https://img.shields.io/github/release/github/vulcanizer.svg)](https://github.com/github/vulcanizer/releases)

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
$ vulcanizer help
Usage:
  vulcanizer [command]

Available Commands:
  aliases         Interact with aliases of the cluster.
  allocation      Set shard allocation on the cluster.
  analyze         Analyze text given an analyzer or a field and index.
  drain           Drain a server or see what servers are draining.
  fill            Fill servers with data, removing shard allocation exclusion rules.
  health          Display the health of the cluster.
  heap            Display the node heap stats.
  help            Help about any command
  hotthreads      Display the current hot threads by node in the cluster.
  indices         Display the indices of the cluster.
  mappings        Display the mappings of the specified index.
  nodeallocations Display the nodes of the cluster and their disk usage/allocation.
  nodes           Display the nodes of the cluster.
  repository      Interact with the configured snapshot repositories.
  setting         Interact with cluster settings.
  settings        Display all the settings of the cluster.
  shards          Get shard data by cluster node(s).
  snapshot        Interact with a specific snapshot.

Flags:
      --cacert string       Path to the certificate to check the cluster certificates against
      --cert string         Path to the certificate to use for client certificate authentication
  -c, --cluster string      Cluster to connect to defined in config file
  -f, --configFile string   Configuration file to read in (default to "~/.vulcanizer.yaml")
  -h, --help                help for vulcanizer
      --host string         Host to connect to (default "localhost")
      --key string          Path to the key to use for client certificate authentication
      --password string     Password to use during authentication
      --path string         Path to prepend to queries, in case Elasticsearch is behind a reverse proxy
  -p, --port int            Port to connect to (default 9200)
      --protocol string     Protocol to use when querying the cluster. Either 'http' or 'https'. Defaults to 'http' (default "http")
  -k, --skipverify string   Skip verifying server's TLS certificate. Defaults to 'false', ie. verify the server's certificate (default "false")
      --user string         User to use during authentication

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

This project is released under the [MIT LICENSE](LICENSE). Please note it includes 3rd party dependencies release under their own licenses; dependencies are listed in our [go.mod](https://github.com/github/vulcanizer/blob/main/go.mod) file. When using the GitHub logos, be sure to follow the [GitHub logo guidelines](https://github.com/logos).

### Authors

Authored by GitHub Engineering
