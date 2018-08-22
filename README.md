# vulcanizer
GitHub's ops focused Elasticsearch library

This project is a golang library for interacting with an Elasticsearch cluster. It's goal is to provide a high level API to help with common tasks that are associated with operating an Elasticsearch cluster such as querying health status of the cluster, migrating data off of nodes, updating cluster settings, etc.

This project does not aim to be a fully-featured API client for querying or indexing to Elasticsearch.

### Command line application

This project produces a command line application that can be used to manage your Elasticsearch cluster:

* `allocation [disable|enable]`
* `drain server <name>`
* `drain status`
* `fill all`
* `fill server <name>`
* `health`
* `indices`
* `nodes`
* `settings`
* `setting update <setting> <value>`
* `snapshots`
* `snapshot status <snapshot name>`

All commands take:
* `--cluster <name>` to look up information in a config file
or
* `--host localhost` and `--port 9200` for the connection information

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
