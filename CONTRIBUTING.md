# Contributing to Kyma

Thank you for your interest in contributing to Kyma! This document provides guidelines and instructions for contributing to this project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Commit Guidelines](#commit-guidelines)
- [Pull Request Process](#pull-request-process)
- [Testing](#testing)
- [Documentation](#documentation)

## Code of Conduct

By participating in this project, you agree to maintain a respectful environment for everyone. Please be kind and courteous to others. We're just writing some lines of code in the end of the day.

## Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/kyma.git
   cd kyma
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/museslabs/kyma.git
   ```
4. Create a new branch for your feature/fix:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Workflow

1. Keep your fork up to date:

   ```bash
   git fetch upstream
   git checkout main
   git rebase upstream/main
   ```

2. Make your changes in a new branch
3. Write tests for your changes
4. Run the test suite
5. Update documentation if needed
6. Commit your changes following the conventional commits format
7. Push to your fork
8. Create a Pull Request

**Note**: We use rebase instead of merge to maintain a clean, linear history. When updating your branch with upstream changes, always use `git rebase` rather than `git merge`. This helps keep the commit history clean and makes it easier to review changes.

## Commit Guidelines

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification. Each commit message should be structured as follows:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Types:

- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `perf`: A code change that improves performance
- `test`: Adding missing tests or correcting existing tests
- `chore`: Changes to the build process or auxiliary tools

Examples:

```
feat(timer): add pause/resume functionality
fix(transition): correct slide transition animation
docs(readme): update installation instructions
```

## Pull Request Process

1. If possible, we'd appreciate if your PR adheres to the following template:

```
Title: provide with concise and informative title.

### TLDR
A quick, less than 80 chars description of the changes.

## Change Summary
- Provide list of key changes with good structure.
- Mention the class name, function name, and file name.
- Explain the code changes.

For example:
## Change Summary
#### Added Features:
    1. **New Functions in `file_name`**:
        - `function_name`: code description.
#### Code Changes:
    1. **In `file_name`**:
#### Documentation Updates:
    1. **In `file_name`**:

### Demo
- N/A

### Context
- N/A
```

2. Update the README.md with details of changes if applicable
3. The PR will be merged once you have the sign-off of at least one other developer
4. Ensure all CI checks pass before marking your PR as ready for review
5. Keep your PR up to date with the main branch by rebasing:
   ```bash
   git fetch upstream
   git rebase upstream/main
   git push -f origin your-branch
   ```

## Testing

Before submitting a PR:

1. Run the test suite:

   ```bash
   go test ./...
   ```

2. Run linters:

   ```bash
   go vet ./...
   golangci-lint run
   ```

3. Ensure all tests pass and there are no linting errors

## Documentation

- Update documentation for any new features or changes
- Update README.md if necessary
- Add comments to complex code sections

## Additional Guidelines

- Follow Go best practices and idioms
- Write meaningful commit messages
- Keep PRs focused and manageable in size
- Be respectful in discussions

Thank you for contributing to Kyma! 🎉
