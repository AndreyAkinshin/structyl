# Documentation Generation

> **Terminology:** This specification uses [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) keywords (MUST, SHOULD, MAY, etc.) to indicate requirement levels.

This document describes Structyl's documentation generation capabilities.

## Scope (v1.0)

Structyl v1.0 focuses on **README generation only**. More complex documentation (PDF manuals, websites) is out of scope.

The README generator:
- Creates per-language README files from templates
- Injects version, install instructions, and demo code
- Ensures consistency across all language implementations

## README Generation

### Command

```bash
structyl docs generate
```

This generates README files for all language targets.

### Template File

Create a template at the path specified in configuration:

```json
{
  "documentation": {
    "readme_template": "templates/README.md.tmpl"
  }
}
```

### Template Format

Templates use placeholder syntax `$PLACEHOLDER$`:

```markdown
# MyProject - $LANG_TITLE$ Implementation

$DESCRIPTION$

## Installation

$INSTALL$

## Quick Start

```$LANG_CODE$
$DEMO$
```

## Version

Current version: $VERSION$

## License

$LICENSE$
```

### Supported Placeholders

| Placeholder | Description | Source |
|-------------|-------------|--------|
| `$VERSION$` | Current version | `VERSION` file |
| `$LANG_TITLE$` | Language display name | Target config `title` |
| `$LANG_SLUG$` | Language short code | Target key (e.g., `cs`, `py`) |
| `$LANG_CODE$` | Markdown code fence language | Target config or default |
| `$DESCRIPTION$` | Project description | `project.description` |
| `$LICENSE$` | License identifier | `project.license` |
| `$INSTALL$` | Installation instructions | Per-language file |
| `$DEMO$` | Demo code | Per-language file |

### Placeholder Resolution Errors

| Condition | Behavior | Exit Code |
|-----------|----------|-----------|
| Unknown placeholder (e.g., `$FOO$`) | Left verbatim in output; warning emitted | 0 |
| Known placeholder, source file missing | Error: `placeholder $INSTALL$ for target cs: source not found at templates/install/cs.md` | 2 |
| Known placeholder, source file empty | Empty string substituted; no error | 0 |
| Known placeholder, source value not configured | Empty string substituted; warning emitted | 0 |

Unknown placeholders are preserved in the output to support custom post-processing or template evolution.

### Per-Language Content

Installation instructions and demo code are sourced from separate files:

```
templates/
├── README.md.tmpl           # Main template
├── install/
│   ├── cs.md                # C# installation
│   ├── py.md                # Python installation
│   └── ...
└── demo/
    ├── cs.md                # C# demo (or extracted from code)
    ├── py.md
    └── ...
```

Alternatively, demo code can be extracted directly from source files:

```json
{
  "targets": {
    "cs": {
      "demo_path": "cs/Demo/Program.cs"
    }
  }
}
```

#### Demo Extraction Rules

When `demo_path` is specified, the source file is processed as follows:

1. **Check for markers:** Scan for `// structyl:demo:begin` and `// structyl:demo:end` markers
2. **If markers found:** Extract only the content between markers
3. **If no markers:** Use the entire file content

Marker syntax (language-specific comment style):
- C-style: `// structyl:demo:begin` ... `// structyl:demo:end`
- Python/R: `# structyl:demo:begin` ... `# structyl:demo:end`
- HTML/XML: `<!-- structyl:demo:begin -->` ... `<!-- structyl:demo:end -->`

Example with markers:
```csharp
using System;
// structyl:demo:begin
var result = Pragmastat.Center.Calculate(data);
Console.WriteLine($"Result: {result}");
// structyl:demo:end
```

Only the two lines between markers are extracted.

### Output Location

READMEs are generated at:

```
<target>/README.md
```

For example:
- `cs/README.md`
- `py/README.md`
- `rs/README.md`

## Configuration

```json
{
  "documentation": {
    "readme_template": "templates/README.md.tmpl",
    "placeholders": ["VERSION", "LANG_TITLE", "LANG_SLUG", "INSTALL", "DEMO"]
  }
}
```

| Field | Description | Default |
|-------|-------------|---------|
| `readme_template` | Path to README template | None (optional feature) |
| `placeholders` | List of placeholder names used | All standard placeholders |

## Example Workflow

1. Create template at `templates/README.md.tmpl`
2. Create per-language install files in `templates/install/`
3. Configure demo paths or create `templates/demo/` files
4. Run `structyl docs generate`
5. Commit generated README files

## Generated vs Source Files

Generated READMEs should be committed to the repository. This ensures:
- Package registries display documentation
- GitHub shows README on repository page
- Offline access to documentation

Add a header comment to indicate the file is generated:

```markdown
<!-- This file is auto-generated. Do not edit directly. -->
<!-- Source: templates/README.md.tmpl -->
<!-- Regenerate with: structyl docs generate -->

# MyProject - C# Implementation
...
```

## Future Considerations (Out of Scope for v1.0)

The following features are not included in v1.0:

- PDF manual generation
- Website/HTML generation
- API documentation extraction
- Changelog generation
- Multi-language documentation (i18n)

These may be considered for future versions based on demand.

## Integration with Version Updates

When running `structyl version set`, documentation is automatically regenerated to include the new version number.

```bash
structyl version set 2.0.0
# Also runs: structyl docs generate
```
