# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

