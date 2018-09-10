# vulcanizer
GitHub's ops focused Elasticsearch library

This project is a golang library for interacting with an Elasticsearch cluster. It's goal is to provide a high level API to help with common tasks that are associated with operating an Elasticsearch cluster such as querying health status of the cluster, migrating data off of nodes, updating cluster settings, etc.

This project does not aim to be a fully-featured API client for querying or indexing to Elasticsearch.

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
  setting     Interact with cluster settings.
  settings    Display all the settings of the cluster.
  snapshot    Interact with a specific snapshot.
  snapshots   Display the snapshots of the cluster.

Flags:
  -c, --cluster string      Cluster to connect to defined in config file
  -f, --configFile string   Configuration file to read in (default to "~/.vulcanizer.yaml")
  -h, --help                help for vulcanizer
      --host string         Host to connect to (default "localhost")
  -p, --port int            Port to connect to (default 9200)

Use "vulcanizer [command] --help" for more information about a command.
```

#### Commands to be implemented

* Listing repositories
* Verifying repositories
* Taking snapshots
* Deleting snapshots
* Displaying index settings
* Updating index settings

#### Configuration and connection information 

All commands take `--cluster <name>` to look up information in a configuration file in `~/.vulcanizer.yaml`. The configuration should be in the form of 

```
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

To be determined

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

This project is released under the [MIT LICENSE](LICENSE). Please note it includes 3rd party dependencies release under their own licenses; these are found under [vendor](https://github.com/github/vulcanizer/tree/master/vendor).

### Authors

Authored by GitHub Engineering
