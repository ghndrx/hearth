# E2E Tests Droid

Creates and maintains end-to-end tests for user flows across the SvelteKit frontend and Go backend integration.

## Trigger
Run on pull requests affecting routes, API handlers, or UI components.

## Capabilities
- Generate Playwright tests for frontend user flows
- Create integration tests for backend API endpoints
- Test WebSocket/gateway connections
- Validate authentication flows

## Instructions

### Context Gathering
1. Identify changed routes in `frontend/src/routes/`
2. Identify changed API handlers in `backend/internal/http/`
3. Check existing E2E tests in `tests/e2e/`

### Playwright Frontend Tests
- Location: `tests/e2e/`
- Test user journeys, not implementation details
- Use data-testid attributes for selectors
- Test both happy path and error states
- Include accessibility checks where relevant

### Backend Integration Tests
- Test API endpoints with real HTTP requests
- Use test fixtures for database state
- Verify response schemas and status codes
- Test authentication/authorization flows

### Test Patterns
```typescript
// Frontend E2E example structure
test.describe('Feature Name', () => {
  test('should handle user action', async ({ page }) => {
    await page.goto('/route');
    await page.click('[data-testid="button"]');
    await expect(page.locator('[data-testid="result"]')).toBeVisible();
  });
});
```

### Output
1. Create/update E2E test files
2. Run tests against local/staging environment
3. Report pass/fail status with screenshots on failure

## Model
inherit

## Tools
- Read, Edit, Create, Grep, Glob, Execute
