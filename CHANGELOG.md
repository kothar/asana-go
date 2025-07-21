# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Security
- **CRITICAL**: Updated Go to version 1.24.4 to address multiple standard library vulnerabilities:
  - GO-2025-3751: Fixed sensitive headers not cleared on cross-origin redirect in net/http
  - GO-2025-3750: Fixed inconsistent handling of O_CREATE|O_EXCL on Unix and Windows in syscall
  - GO-2025-3749: Fixed usage of ExtKeyUsageAny disabling policy validation in crypto/x509
- **MEDIUM**: Updated golang.org/x/oauth2 from v0.24.0 to v0.30.0 to fix:
  - GO-2025-3488: Fixed unexpected memory consumption during token parsing vulnerability

### Added
- Added comprehensive security scanning to CI/CD pipeline:
  - `govulncheck` for Go vulnerability scanning
  - Trivy for dependency and container vulnerability scanning
  - gosec for static application security testing
- Added SECURITY.md with security policy and best practices
- Added Dependabot configuration for automated dependency updates
- Added security-focused GitHub Actions workflow

### Changed
- Enhanced GitHub Actions workflow with multiple security jobs
- Updated GitHub Actions to use latest versions (checkout@v4, setup-go@v5)
- Added support for both `develop` and `main` branches in CI/CD

### Infrastructure
- Added automated security scanning on every push and pull request
- Configured Dependabot for weekly dependency updates
- Enhanced CI/CD pipeline with parallel security checks

## Security Notice

This release addresses critical security vulnerabilities in the Go standard library and OAuth2 dependency. Users are strongly encouraged to update immediately.

### Upgrade Instructions

1. Update your Go installation to 1.24.4 or later
2. Run `go get -u bitbucket.org/mikehouston/asana-go` to get the latest version
3. Run `go mod tidy` to ensure dependencies are updated
4. Test your application thoroughly after updating

### Breaking Changes

None. All changes are backward compatible.