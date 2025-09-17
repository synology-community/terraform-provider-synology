---
applyTo: '**/*.go'
---
# Go Development Conventions Summary

## Directory and File Structure
- **Package organization**: Avoid package sprawl; group related functionality appropriately
- **Utility packages**: Avoid generic "util" packages - use descriptive names (e.g., `wait.Poll` instead of `util.Poll`)
- **Naming conventions**: 
  - All filenames lowercase
  - Use underscores in Go source files/directories (not dashes)
  - Package directories avoid separators when possible
  - Use nested subdirectories for multi-word package names

## Code Organization
- **New libraries**: Place in `pkg/util` subdirectories if no better home exists
- **Third-party code**: Manage Go dependencies with modules; other code goes in `third_party/`

## Testing Requirements
- **Unit tests**: Required for all new packages and significant functionality
- **Test style**: Prefer table-driven tests for multiple scenarios
- **Integration tests**: Required for significant features and kubectl commands
- **Cross-platform**: Tests must pass on macOS and Windows
- **Async testing**: Use wait/retry patterns instead of fixed delays
- **Dependencies**: Use Google Cloud Artifact Registry instead of Docker Hub

## Key Principles
- Prioritize descriptive naming over generic utilities
- Ensure cross-platform compatibility
- Write comprehensive tests at multiple levels
- Follow established patterns for package organization
