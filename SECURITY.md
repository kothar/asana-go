# Security Policy

## Supported Versions

We actively support the following versions with security updates:

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| < Latest| :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly:

1. **Do not** open a public GitHub issue
2. Email the maintainers directly (if contact info is available)
3. Use GitHub's Security Advisory feature for private disclosure
4. Include as much detail as possible about the vulnerability

## Security Best Practices

### For Users of this Library

1. **Always use the latest version** of this library
2. **Keep Go updated** to the latest stable version
3. **Validate all API responses** before processing
4. **Use HTTPS** for all API communications
5. **Store API tokens securely** (environment variables, secret management systems)
6. **Implement proper timeout configurations** for HTTP clients
7. **Use rate limiting** to prevent API abuse

### Example Secure Configuration

```go
package main

import (
    "context"
    "net/http"
    "time"
    "bitbucket.org/mikehouston/asana-go"
)

func main() {
    // Create HTTP client with security-focused configuration
    httpClient := &http.Client{
        Timeout: 30 * time.Second,
        Transport: &http.Transport{
            MaxIdleConns:       10,
            IdleConnTimeout:    30 * time.Second,
            DisableCompression: false,
            TLSHandshakeTimeout: 10 * time.Second,
        },
    }
    
    client := asana.NewClient(httpClient)
    // Use the client...
}
```

## Dependency Security

This project uses Go modules for dependency management. We regularly update dependencies to address security vulnerabilities.

### Current Security Measures

- Automated vulnerability scanning with `govulncheck`
- Dependency vulnerability scanning with Trivy
- GitHub Security Advisories enabled
- Regular dependency updates

## Security Scanning

The project includes automated security scanning in CI/CD:

- **govulncheck**: Scans for known vulnerabilities in Go dependencies
- **Trivy**: Multi-purpose security scanner for vulnerabilities and misconfigurations
- **gosec**: Static analysis tool for Go security issues

## Changelog of Security Updates

### Latest Updates
- **2025-01**: Updated Go to 1.24.4 to address standard library vulnerabilities (GO-2025-3751, GO-2025-3750, GO-2025-3749)
- **2025-01**: Updated golang.org/x/oauth2 to v0.30.0 to address memory consumption vulnerability (GO-2025-3488)
- **2025-01**: Enhanced CI/CD pipeline with comprehensive security scanning

## Contact

For security-related questions or concerns, please:
1. Check existing GitHub Security Advisories
2. Review this security policy
3. Contact the maintainers through appropriate channels

## Acknowledgments

We appreciate security researchers who responsibly disclose vulnerabilities. All verified security issues will be acknowledged in our security changelog.