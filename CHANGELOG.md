# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.5.5] - 2025-01-27
### Update
- Bump `golang.org/x/net` to `v0.34.0`

## [0.5.4] - 2024-09-02
### Update
- Bump `docker/docker` to `v27.2.0`

## [0.5.3] - 2024-08-12
### Update
- Bump `docker/docker` to `v26.1.5`

## [0.5.2] - 2024-07-31
### Fixed
- Use `client.FromEnv` when creating docker client

## [0.5.1] - 2024-07-31
### Update
- Bump `docker/docker` to `v26.1.4`

## [0.5.0] - 2024-07-30
### Update
- BREAKING: Update `docker/docker` version. `RepositoryConfig.Auth` is now a `registry.AuthConfig` instead of `types.AuthConfig`.

## [0.4.0] - 2023-07-26
### Update
- Update `docker/docker` version

## [0.3.0] - 2020-11-20
### Added
- `Runner.StopWithTimeout` and `Config.StopTimeout` to stop containers within the provided time, 0 by default to reduce the tear down time.

### Changed
- Refactored image pulling logging.

## [0.2.1] - 2020-11-18
### Fixed
- `ImagePuller` authentication.

## [0.2.0] - 2020-11-18
### Changed
- BREAKING: `aceptadora.New()` now accepts an `ImagePuller` instead of creating it, and `aceptadora.Config` no longer contains the `aceptadora.ImagePullerConfig`.
  This allows reusing same ImagePuller for multiple aceptadora instances (one per test probably) and taking advantage of the image cache that `ImagePuller` has to avoid pulling same image multiple times.

## [0.1.0] - 2020-10-14
### Added
- Initial public version

