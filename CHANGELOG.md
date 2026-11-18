# Changelog

## v0.1.1

### Added

- Added a bundled `go` module that installs the official Go toolchain. Projects select a version in `go.work` or `go.mod`, without a VBX-specific version setting.

### Changed

- VBX discovers bundled modules from its embedded `modules/` directory. Adding a bundled module no longer requires a code registration change.

### Fixed

- New VMs use `cache: loose` for writable 9p workspace mounts so VS Code and other file watchers detect guest-side edits. Existing VMs keep their current mount settings. Loose caching can delay when host-side edits become visible in the guest.
