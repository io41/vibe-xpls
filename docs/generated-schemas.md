# Generated Schemas

The built-in Crossplane schema bundle is generated from committed upstream Crossplane release artifacts.

Current pins:

- Crossplane `v1.20.7`, commit `5fae6c1ab967e57b1dc792b5c52c97bceda12953`
- Crossplane `v2.2.1`, commit `713541df7fc5cf0946b6573837831086465a2dbe`

Regenerate after changing `internal/analyzer/schemadata/config.json` or generator code:

```bash
go run ./cmd/vibe-xpls-schema-gen --config internal/analyzer/schemadata/config.json --out internal/analyzer/schemadata
go test ./internal/schemagen ./internal/analyzer
```

The generator must produce byte-identical output from committed inputs. Runtime never downloads schemas.
