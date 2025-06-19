---
applyTo: '**/*.go'
---
Coding standards, domain knowledge, and preferences that AI should follow.

## Directory and file conventions

- Avoid package sprawl. Find an appropriate subdirectory for new packages.
- Libraries with no appropriate home belong in new package subdirectories of pkg/util.
- Avoid general utility packages. Packages called “util” are suspect. Instead, derive a name that describes your desired function. For example, the utility functions dealing with waiting for operations are in the wait package and include functionality like Poll. The full name is wait.Poll.
- All filenames should be lowercase.
- Go source files and directories use underscores, not dashes.
- Package directories should generally avoid using separators as much as possible. When package names are multiple words, they usually should be in nested subdirectories.
- Document directories and filenames should use dashes rather than underscores.
- Examples should also illustrate best practices for configuration and using the system.
- Follow these conventions for third-party code:
- Go code for normal third-party dependencies is managed using go modules and is described in the kubernetes vendoring guide.
- Other third-party code belongs in third_party.

## Testing conventions

- All new packages and most new significant functionality must come with unit tests.
- Table-driven tests are preferred for testing multiple scenarios/inputs. For an example, see TestNamespaceAuthorization.
- Significant features should come with integration (test/integration) and/or end-to-end (test/e2e) tests.
- Including new kubectl commands and major features of existing commands.
- Unit tests must pass on macOS and Windows platforms - if you use Linux specific features, your test case must either be skipped on windows or compiled out (skipped is better when running Linux specific commands, compiled out is required when your code does not compile on Windows).
- Avoid relying on Docker Hub. Use the Google Cloud Artifact Registry instead.
- Do not expect an asynchronous thing to happen immediately—do not wait for one second and expect a pod to be running. Wait and retry instead.
