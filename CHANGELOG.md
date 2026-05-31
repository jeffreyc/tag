# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [2.0.0] - 2026-05-31

### Changed

- Updated release builds to drop unsupported `darwin/386`, add `darwin/arm64` and `linux/arm64`, and package macOS binaries as `.tar.gz` archives.
- Changed the default search backend from `ag` to ripgrep (`rg`).
- Updated shell setup examples to alias `rg` through `tag` by default.
- Switched project identity to the `github.com/jeffreyc/tag` fork.
- Updated install and release links for the `jeffreyc/tag` fork.
- Added Go module metadata (`go.mod` and `go.sum`) so the project builds from a clean checkout with modern Go tooling.
- Modernized the release build target to build from the module root with explicit output paths.

### Fixed

- Fixed `--notag` handling when it is passed as the first argument.
- Fixed a ripgrep panic when `--color` was provided without a following value.
- Recognized both `--color never` and `--color=never` when preserving user color settings.
- Forced ripgrep to emit filename headings so single-file searches generate correct aliases.
- Preserved the current filename while parsing context output from `rg -C`, `-A`, and `-B`, so matches in context searches are tagged correctly.
- Increased scanner capacity and now report scanner errors instead of silently producing incomplete aliases for long output lines.
- Cleared stale generated aliases before writing new aliases, so old `eN` aliases do not survive shorter or empty result sets.
- Fixed Windows release packaging to include `tag.exe` instead of a stale non-Windows `tag` binary.

### Security

- Quoted generated shell aliases to prevent filenames from breaking out of the alias file when it is sourced.
- Escaped filenames for the default command template and documented safe double-quoted `{{.Filename}}` usage for custom templates.

[unreleased]: https://github.com/jeffreyc/tag/compare/v2.0.0...HEAD
[2.0.0]: https://github.com/jeffreyc/tag/releases/tag/v2.0.0
