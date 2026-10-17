# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [1.0.1]

### Fixed

- `npx @init0-lux/vcf-toolkit` failed with "command not found": the npm package now ships its launcher script and downloads from the correct release URL. CI checks the packed files before publishing.
- Phone numbers written without a country code were given a wrong one (`(212) 555-0101` became `+2125550101`). They are now written as typed unless `--country` supplies the calling code.
- Outlook and Google Contacts exports lost last names, phones and emails. Header detection now matches by words, combines first/middle/last name columns, skips label and type columns, and splits Google `:::` multi-value cells. `convert`, `dedupe`, `normalize` and the TUI share one parser.
- A CSV with no recognizable columns now fails with an error instead of producing empty output.
- `Doe, John` produced a swapped structured name; `Family, Given` and `Given Family, Jr.` are now understood.
- Email addresses were rewritten in VCF output (plus-aliases removed, lowercased). They are now written as given.
- Dedupe merged different people who shared a phone or inbox. The minimum name similarity is now 0.80.
- The TUI's normalize-and-dedupe workflows dropped every contact that had no duplicate.
- Reading CSV from an input that fails mid-stream (e.g. a broken pipe) looped forever; it now stops with an error.

### Added

- `--country` on `convert` and `normalize`, and `VCF_TOOLKIT_COUNTRY` for the TUI.
- `--map "Column=field"` on `convert`, `dedupe` and `normalize`.
- `--strip-aliases` on `normalize`.
- `--version`.
- Go API: `ContactToVCFWithOptions`, `NormalizeContact`, `normalize.NormalizeCountryCode`, `dedupe.Representatives`, given/middle/family name fields and constants, and `ErrNoRecognizedColumns`.

### Changed

- The npm package depends on `tar` 7.
- The built binary is no longer tracked in git.

## [1.0.0] - 2026-05-26

- Initial release: CLI and Go SDK for normalizing, deduplicating and converting CSV contacts to vCard 4.0, with a terminal UI and an npm wrapper.
