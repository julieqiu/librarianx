# Librarian Documentation

This directory contains documentation for the Librarian project.

## Core Documentation

- **[prd.md](prd.md)** - Project objective, background, and key design principles
- **[userguide.md](userguide.md)** - CLI overview, commands, and common workflows
- **[config.md](config.md)** - Complete `librarian.yaml` schema reference
- **[librarianops.md](librarianops.md)** - Operations tool for managing multiple repositories
- **[alternatives.md](alternatives.md)** - Design decisions and alternatives considered

## Language-Specific Documentation

- **[go.md](go.md)** - Go generation, configuration, and workflows
- **[python.md](python.md)** - Python generation, configuration, and workflows
- **[rust.md](rust.md)** - Rust generation, configuration, and workflows

## Sidekick Documentation (Rust)

- **[sidekick.md](sidekick.md)** - Sidekick overview
- **[sidekick-how-to-guide.md](sidekick-how-to-guide.md)** - Detailed Sidekick configuration guide
- **[sidekick-merge-strategy.md](sidekick-merge-strategy.md)** - Merge strategies for Sidekick updates

## Other Documentation

- **[CONTRIBUTING.md](CONTRIBUTING.md)** - Contribution guidelines
- **[prompt.md](prompt.md)** - Design assistant prompt for working on Librarian
- **[todo.md](todo.md)** - Current project TODO list

## Quick Links

### Getting Started
1. Read [prd.md](prd.md) to understand the project
2. Read [userguide.md](userguide.md) to learn the CLI
3. Read language-specific docs ([go.md](go.md), [python.md](python.md), or [rust.md](rust.md))

### Reference
- [config.md](config.md) - Complete configuration schema
- [alternatives.md](alternatives.md) - Design decisions and rationale

### Contributing
- [CONTRIBUTING.md](CONTRIBUTING.md) - How to contribute to Librarian

## Documentation Guidelines

All documents should follow these guidelines:

### Formatting

- Use [GitHub Flavored Markdown](https://github.github.com/gfm/)
- Wrap lines around 80 columns for easier PR reviews
- Use code blocks with language specifiers (```yaml, ```bash, etc.)

### Content Guidelines

- **Be clear and concise** - Use simple language
- **Provide examples** - Show concrete examples for concepts
- **Link between docs** - Cross-reference related documentation
- **Keep it current** - Update docs when features change

### Organization

- **prd.md** - High-level project overview and principles
- **userguide.md** - User-facing CLI documentation for single repository
- **librarianops.md** - Operations tool for managing multiple repositories
- **config.md** - Technical reference for configuration
- **Language docs** - Language-specific details and workflows
- **alternatives.md** - Design decisions and history

## Updating Documentation

When making changes:

1. **Update the right document** - Follow the organization above
2. **Update cross-references** - Fix links if you rename or move content
3. **Add examples** - Include code examples for new features
4. **Test examples** - Ensure all examples work
5. **Update this README** - If adding new documents

## Confidentiality

As this repository is public, do not provide any details of Google internal systems, or refer to such systems unless they have already been described in public documents.

The following systems have been mentioned publicly and can be referred to:
- go links
- Piper and google3
- Critique
- Kokoro
- Changelists (CLs)
- GAPIC

See the [Google Open Source Glossary](https://opensource.google/documentation/reference/glossary) for more terms.

Use go links and b/ links to refer to internal documents and bugs.
