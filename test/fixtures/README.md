# Test Fixtures

This directory contains project fixtures used by integration tests.

## Directory Structure

```
fixtures/
├── minimal/              # Minimal valid project
├── multi-language/       # Multi-target project with dependencies
├── with-docker/          # Docker configuration testing
└── invalid/              # Invalid configuration scenarios
    ├── missing-name/     # Missing required project.name
    ├── circular-deps/    # Circular target dependency
    ├── invalid-toolchain/# Unknown toolchain reference
    └── malformed-json/   # Syntax error in JSON
```

## Fixture Descriptions

### `minimal/`

Minimal valid structyl project with only required fields.

- **Purpose**: Test project loading with no targets or optional configuration
- **Used by**: Basic project discovery tests, config validation tests

### `multi-language/`

Multi-target project demonstrating:
- Multiple language targets (py, rs)
- Target dependencies (rs depends on py)
- Reference test data in `tests/basic/`

- **Purpose**: Test target registry, dependency resolution, cross-target operations
- **Used by**: Integration tests, target resolution tests

### `with-docker/`

Project with Docker configuration enabled.

- **Purpose**: Test Docker mode detection and compose file parsing
- **Used by**: Docker integration tests

### `invalid/`

Collection of invalid configurations for error handling tests.

| Subdirectory | Error Type | Expected Behavior |
|--------------|------------|-------------------|
| `missing-name` | Validation error | Exit code 2, missing project.name |
| `circular-deps` | Dependency cycle | Exit code 2, topological sort fails |
| `invalid-toolchain` | Unknown toolchain | Exit code 2 or warning depending on context |
| `malformed-json` | Parse error | Exit code 2, JSON syntax error |

## Adding New Fixtures

1. Create a new directory under `fixtures/`
2. Add `.structyl/config.json` with the required configuration
3. Add any additional files needed for the test scenario
4. Update this README with the fixture description
5. Add tests that use the fixture in `test/integration/`

## Conventions

- Fixtures should be self-contained and not depend on external state
- Invalid fixtures go under `invalid/` with a descriptive subdirectory name
- Test data files (JSON) go under `tests/` within the fixture if needed
- Keep fixtures minimal - only include what's needed for the test
