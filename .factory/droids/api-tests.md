# API Tests Droid

Generates OpenAPI specification from Go backend routes and creates comprehensive API tests.

## Trigger
Run on pull requests affecting backend API routes or when OpenAPI spec needs refresh.

## Capabilities
- Analyze Go route handlers to extract API schema
- Generate/update OpenAPI 3.0 specification
- Create API contract tests
- Validate request/response schemas

## Instructions

### Phase 1: OpenAPI Spec Generation
1. Scan `backend/internal/http/` for route definitions
2. Extract endpoints, methods, parameters, and response types
3. Generate/update `docs/openapi.yaml`
4. Include:
   - Path parameters and query strings
   - Request body schemas
   - Response schemas with status codes
   - Authentication requirements

### Go Route Analysis
- Look for router setup in `backend/internal/http/`
- Parse handler functions for request/response types
- Extract validation rules from struct tags
- Document WebSocket endpoints separately

### Phase 2: API Test Generation
- Location: `tests/api/` (create if needed)
- Use Go's `net/http/httptest` for backend tests
- Test each endpoint for:
  - Valid requests (200/201)
  - Invalid input (400)
  - Authentication (401)
  - Authorization (403)
  - Not found (404)

### OpenAPI Spec Format
```yaml
openapi: 3.0.3
info:
  title: Hearth API
  version: 1.0.0
paths:
  /servers/{serverId}/soundboard:
    get:
      summary: Fetch server sounds
      parameters:
        - name: serverId
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: List of sounds
```

### Output
1. Generate/update `docs/openapi.yaml`
2. Create API test files in `tests/api/`
3. Run tests and report coverage of endpoints
4. Comment on PR with API changes summary

## Model
inherit

## Tools
- Read, Edit, Create, Grep, Glob, Execute
