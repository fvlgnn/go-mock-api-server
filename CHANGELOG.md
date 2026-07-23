# Changelog

All notable changes to this project are documented in this file. The format is
based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the
project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-07-23

### Added

- Runtime configuration through environment variables and read-only volumes.
- Strict endpoint validation with duplicate route detection.
- Exact method and path routing with correct 404, 405 and `Allow` responses.
- Configurable response status codes and headers.
- Built-in health endpoint and container healthcheck.
- Graceful shutdown, HTTP timeouts and structured logging.
- Unit tests, race detection and container smoke tests.
- Version information embedded in the binary and OCI image metadata.
- Multi-architecture GHCR images for `linux/amd64` and `linux/arm64`.
- Automated GitHub releases driven by version-specific changelog sections.

### Changed

- Updated the project to Go 1.26.
- Reworked the final container as a minimal, non-root `scratch` image.
- Corrected and expanded the documentation and bundled endpoint examples.

### Fixed

- Duplicate paths no longer cause an unhandled `http.ServeMux` panic.
- Configured paths no longer match unintended child paths.
- Invalid or incomplete JSON definitions now fail before the server starts.

[Unreleased]: https://github.com/fvlgnn/go-mock-api-server/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/fvlgnn/go-mock-api-server/releases/tag/v1.0.0
