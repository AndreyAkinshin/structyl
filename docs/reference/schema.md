# JSON Schema

Structyl provides a JSON Schema for IDE autocomplete and validation.

## Using the Schema

Add the schema reference to your `.structyl/config.json`:

```json
{
  "$schema": "https://structyl.akinshin.dev/structyl.schema.json",
  "project": {
    "name": "my-library"
  }
}
```

Or use a local path:

```json
{
  "$schema": "./docs/public/structyl.schema.json",
  "project": {
    "name": "my-library"
  }
}
```

## Download

Download the schema: [structyl.schema.json](/structyl.schema.json)

## IDE Support

### VS Code

VS Code automatically uses the `$schema` reference for:

- Autocomplete suggestions
- Error highlighting
- Hover documentation

### JetBrains IDEs

IntelliJ, WebStorm, and other JetBrains IDEs support JSON Schema via the `$schema` field.

### Vim/Neovim

With coc.nvim or nvim-lspconfig, JSON schemas are automatically applied.

## Schema vs Runtime Validation

The JSON Schema is for **IDE assistance**. Structyl's runtime validation is more lenient:

| Aspect         | JSON Schema (IDE) | Runtime (Structyl)   |
| -------------- | ----------------- | -------------------- |
| Unknown fields | May reject        | Ignored with warning |
| Purpose        | Editor assistance | Execution            |
| Strictness     | Full validation   | Required fields only |

This design allows newer configurations to work with older IDE schemas while maintaining forward compatibility.

## Configuration Reference

The full configuration structure:

```json
{
  "$schema": "https://structyl.akinshin.dev/structyl.schema.json",

  "project": {
    "name": "string (required)",
    "description": "string",
    "homepage": "string (URL)",
    "repository": "string (URL)",
    "license": "string (SPDX identifier)"
  },

  "version": {
    "source": "string (default: VERSION)",
    "files": [
      {
        "path": "string (required)",
        "pattern": "string (required, regex)",
        "replace": "string (required)",
        "replace_all": "boolean (default: false)"
      }
    ]
  },

  "targets": {
    "<slug>": {
      "type": "language | auxiliary",
      "title": "string (required)",
      "toolchain": "string",
      "directory": "string",
      "cwd": "string",
      "commands": {
        "<command>": "string | array | object | null"
      },
      "vars": { "<key>": "string" },
      "env": { "<key>": "string" },
      "depends_on": ["string"]
    }
  },

  "toolchains": {
    "<name>": {
      "extends": "string",
      "commands": { "<command>": "string" }
    }
  },

  "tests": {
    "directory": "string (default: tests)",
    "pattern": "string (default: **/*.json)",
    "comparison": {
      "float_tolerance": "number (default: 1e-9)",
      "tolerance_mode": "absolute | relative | ulp",
      "array_order": "strict | unordered",
      "nan_equals_nan": "boolean (default: true)"
    }
  },

  "docker": {
    "compose_file": "string",
    "env_var": "string (default: STRUCTYL_DOCKER)",
    "services": {
      "<target>": {
        "base_image": "string",
        "dockerfile": "string",
        "platform": "string"
      }
    }
  }
}
```

## Required Fields

Only `project.name` is required. All other fields have sensible defaults.

Minimal valid configuration:

```json
{
  "project": {
    "name": "my-library"
  }
}
```
