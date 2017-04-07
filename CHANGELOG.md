Change Log
=========

## [Unreleased]


## [0.1.6] - 2017-04-07
### Added
 - restapi route for adding and getting tilelayers
### Changed
 - Read tilelayers from database


## [0.1.5] - 2017-04-03
### Added
 - Global variables
### Changed
 - merged duplicated functions for easier testing and debugging
 - logging reports server name and version

## [0.1.4] - 2017-04-03
### Added
 - Commenting for documentation
 - metadata route for each layer
### Changed
 - gorilla mux used for postgresql and sqlite3 servers
 - Postgres descriptions to table and fields
 - Updated Makefile
 - tms xml responses
### Fixed
 - metadata table support for multiple tile layers

## [0.1.3] - 2017-12-01
### Added
 - config file
 - tile url proxy
 - Vagrantfile

## [0.1.2] - 2016-11-01
### Added
 - RESTApi
 - Logging library
### Changed
 - Postgres backend option

## [0.1.1] - 2016-11-01
### Added
 - Fork from https://github.com/fawick/go-mapnik
