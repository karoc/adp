# Agent Orchestration: Web Application Development

This example demonstrates agent collaboration patterns for full-stack web development with separate backend and frontend specialists.

## Agents

### backend-dev (Backend Engineer)
- **Focus**: REST API, data models, authentication, server logic
- **Skills**: Go, HTTP servers, API design, data validation
- **Responsibilities**:
  - Design and implement REST endpoints
  - Define data models and validation
  - Handle authentication and authorization
  - Write backend tests
  - Document API contracts

### frontend-dev (Frontend Engineer)
- **Focus**: React components, UI/UX, API integration, user interaction
- **Skills**: React, JavaScript, CSS, responsive design
- **Responsibilities**:
  - Implement UI components
  - Integrate with backend API
  - Handle user interactions and state
  - Write frontend tests
  - Ensure responsive design

## Collaboration Patterns

### Pattern 1: API Contract-First Development

**Step 1 - Both agents**: Agree on API contract
- Define endpoints, request/response formats
- Document in `memory/api-contracts.md`

**Step 2 - Backend-dev**: Implement API
- Create endpoints matching contract
- Add validation and error handling
- Write tests

**Step 3 - Frontend-dev**: Implement UI
- Create API client following contract
- Build components consuming API
- Handle loading and error states

**Key**: Contract defined first enables parallel development

### Pattern 2: Feature Development with Dependencies

When adding a new feature requiring both backend and frontend:

**Task T1 (backend-dev)**: Implement backend endpoint
```go
// POST /api/auth/login
func (s *Server) LoginHandler(w http.ResponseWriter, r *http.Request) {
    // Validate credentials
    // Generate token
    // Return response
}
```

**Task T2 (frontend-dev, depends_on: T1)**: Build login UI
```javascript
// Login form component
const handleLogin = async () => {
    const data = await apiClient.login(username, password);
    setToken(data.token);
};
```

Frontend task waits for backend API to be ready.

### Pattern 3: Iterative Refinement

**Iteration 1 - Backend-dev**: Basic endpoint
```go
// Returns simple user list
func (s *Server) UsersHandler(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, s.users)
}
```

**Iteration 1 - Frontend-dev**: Basic display
```javascript
// Simple list rendering
{users.map(user => <li key={user.id}>{user.username}</li>)}
```

**Iteration 2 - Both refine**:
- Backend adds pagination, filtering
- Frontend adds search, sorting UI

### Pattern 4: Error Handling Coordination

**Backend-dev establishes error format**:
```go
// Consistent error response
writeError(w, http.StatusBadRequest, "Username required")
// Returns: {"error": "Username required"}
```

**Frontend-dev consumes consistently**:
```javascript
try {
    await apiClient.login(username, password);
} catch (err) {
    setError(err.message); // Display to user
}
```

**Documentation**: Error format documented in `memory/api-contracts.md`

## Communication

Agents communicate through:
- **API Contracts** (`memory/api-contracts.md`) - Endpoint definitions
- **Task Dependencies** (`tasks.yaml`) - Explicit ordering
- **Code Comments** - Interface expectations
- **Integration Tests** - End-to-end validation

## Task Assignment Guidelines

Assign to **backend-dev**:
- API endpoint implementation
- Data models and validation
- Authentication/authorization
- Database operations
- Server configuration

Assign to **frontend-dev**:
- UI components and layouts
- API client integration
- User interaction handling
- Responsive design
- Frontend state management

Assign to **both** (requires coordination):
- New features spanning both layers
- API contract changes
- Error handling strategies
- Performance optimization
- Integration testing

## Example Workflow

### Adding User Registration Feature

**Step 1**: Define contract in `memory/api-contracts.md`
```markdown
## POST /api/auth/register
Request: { username, email, password }
Response: { user_id, username, email }
Errors: 400 (validation), 409 (user exists)
```

**Step 2**: Create tasks in `tasks.yaml`
```yaml
- id: T10
  title: "Implement user registration API"
  assignee: backend-dev
  
- id: T11
  title: "Create registration form UI"
  assignee: frontend-dev
  depends_on: [T10]
```

**Step 3**: Backend-dev implements
```go
func (s *Server) RegisterHandler(w http.ResponseWriter, r *http.Request) {
    // Validate input
    // Check if user exists
    // Create user
    // Return response
}
```

**Step 4**: Frontend-dev implements
```javascript
const handleRegister = async () => {
    const data = await apiClient.register(username, email, password);
    // Show success, redirect to login
};
```

**Step 5**: Integration test validates both parts work together

## Best Practices

1. **Document API changes** - Always update `memory/api-contracts.md`
2. **Use task dependencies** - Frontend tasks depend on backend API availability
3. **Consistent error handling** - Agree on error response format
4. **Test integration** - Run both backend and frontend together
5. **Version API changes** - Consider backward compatibility
