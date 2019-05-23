# Roadmap for vulcanizer features

This is not a guarantee of any feature, but a rough plan on where we will develop further.

## v0.1.0 - [Released](https://github.com/github/vulcanizer/releases/tag/v0.1.0)

Release of basic functionality for the basic client with a idiomatic API containing structs for data returned and a possible error.

Functionality:
* Get health status of a cluster
* Get nodes of a cluster
* Get indices of a cluster
* Get settings of a cluster
* Get snapshots of a cluster
* Get details of a snapshot
* Drain a server - set shard allocation rules so that data moves off of the server
* Fill a server - remove shard allocations rules so that data moves on to the server
* Fill all servers - remove all shards allocation exclusion rules
* Set a cluster setting
* Enable or disable cluster allocation entirely


## v0.2.0 - [Released](https://github.com/github/vulcanizer/releases/tag/v0.2.0)

Handle more cases around repositories and snapshots.

Functionality:
* Verify a repository
* Delete a snapshot

## v0.3.0 - [Released](https://github.com/github/vulcanizer/releases/tag/v0.3.0)

Even more cases around repositories and snapshots.

Functionality:
* List repositories
* Take snapshots
* Restore snapshots

## v0.4.0 - [Released](https://github.com/github/vulcanizer/releases/tag/v0.4.0)

Add functionality for managing indices.

Functionality:
* Delete index
* Get pretty index settings
* Get machine readable index settings
* Set index settings
* Get pretty index mappings
* Analyze text with built in analyzers
* Analyze text based on a field
* Additional client options for HTTPS, timeout, HTTP basic auth, and TLS configuration

## v0.5.0 - [Released](https://github.com/github/vulcanizer/releases/tag/v0.5.0)

Functionality:
* Open index
* Close index
* List/add/update/remove aliases
* List shards
* Display shard recovery information
* New client configuration options: user, password, path, protocol, TLS verification
