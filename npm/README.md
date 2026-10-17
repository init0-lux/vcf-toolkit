# @init0-lux/vcf-toolkit

Run the [vcf-toolkit](https://github.com/init0-lux/vcf-toolkit) CLI via npx. On first run the package downloads the release binary for your platform (Linux, macOS or Windows on x64 or arm64) and caches it.

## Usage

- `npx @init0-lux/vcf-toolkit --help`
- `npx @init0-lux/vcf-toolkit convert contacts.csv > contacts.vcf`
- `npx @init0-lux/vcf-toolkit convert registrations.csv --country +91 --out contacts.vcf`
- `npx @init0-lux/vcf-toolkit dedupe contacts.csv --dry-run`

## Environment

- `VCF_TOOLKIT_VERSION`: run a different release than the package version, e.g. `1.0.1`
- `VCF_TOOLKIT_DIST_URL`: download binaries from a mirror instead of GitHub releases
