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

#### v0.2.1 - Proposed

Functionality:
* List repositories
* Create a repository

### v0.3.0 - Proposed

Show more information around shard allocation and recovery.

Functionality:
* List shards on a node
* List shards moving to / from a node
* Show recovery information in a friendly manner, like percentages and maybe calculate an estimated time
* Show allocation explain information in a friendly manner

### v0.4.0 - Proposed

Handle more index-related cases.

Functionality:
* List aliases
* Modify aliases
* Show index settings
* Modify index settings
* Show mappings
* Diff mappings
