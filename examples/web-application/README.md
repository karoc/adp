# Web Application Example

简体中文：[README.zh-CN.md](README.zh-CN.md)

> Learn full-stack development patterns with backend and frontend agent specialization for Go REST API and React applications.

## What You'll Learn

- **API Contract-First Development**: Define contracts before implementation
- **Agent Specialization**: Backend vs Frontend division of responsibilities
- **Task Dependencies**: Frontend tasks wait for backend API availability
- **Full-Stack Integration**: Connect Go backend with React frontend

## Prerequisites

- **ADP installed**: Run `adp version` to verify
- **Go 1.21+**: Required for backend API
- **Node.js 16+**: Required for React frontend
- **4 minutes**: Time budget from setup to running

## Quick Start

```bash
# Clone the repository (if not already)
cd examples/web-application

# One-command setup
./setup.sh

# Start backend (Terminal 1)
cd project/backend
./backend-server

# Start frontend (Terminal 2)
cd project/frontend
npm start

# Open browser
# Frontend: http://localhost:3000
# Backend API: http://localhost:8080
```

**That's it!** No configuration edits required.

## Project Structure

```
web-application/
├── README.md                    # This file
├── setup.sh                     # One-command setup script
├── workspace.yaml               # Workspace configuration
├── AGENTS.md                    # Agent collaboration patterns
├── tasks.yaml                   # Example task definitions
├── phases.yaml                  # Development phase structure
│
├── profiles/                    # Agent profiles
│   ├── backend-dev.yaml         # Backend engineer
│   └── frontend-dev.yaml        # Frontend engineer
│
├── prompts/                     # Agent instructions
│   ├── backend-engineer.md      # Backend guidelines
│   └── frontend-engineer.md     # Frontend guidelines
│
├── memory/                      # Shared context
│   └── api-contracts.md         # API endpoint contracts
│
├── mcp/                         # MCP server config
│   └── config.yaml
│
└── project/                     # Full-stack application
    ├── backend/                 # Go REST API
    │   ├── main.go              # HTTP server
    │   ├── go.mod
    │   └── api/
    │       ├── handlers.go      # API endpoints
    │       └── handlers_test.go # Tests
    │
    └── frontend/                # React app
        ├── package.json
        ├── public/
        │   └── index.html
        └── src/
            ├── App.js           # Main component
            ├── api.js           # API client
            ├── App.test.js      # Component tests
            └── api.test.js      # API tests
```

## Agent Orchestration

This example demonstrates **API contract-first development**:

### backend-dev (Backend Engineer)
- **Focus**: REST API, data models, authentication
- **Skills**: Go, HTTP servers, API design, validation
- **Assigned Tasks**: T1-T3 (health, users, login endpoints), T7 (pagination)

### frontend-dev (Frontend Engineer)
- **Focus**: React components, UI/UX, API integration
- **Skills**: React, JavaScript, responsive design
- **Assigned Tasks**: T4-T6 (API client, login UI, users list), T8 (search)

### Collaboration Pattern: Contract-First

1. **Both agents agree on API contract** - Defined in `memory/api-contracts.md`
2. **Backend implements endpoints** - Following contract spec
3. **Frontend integrates API** - Consuming contract-compliant endpoints
4. **Integration tests validate** - Ensures both parts work together

This enables **parallel development** without blocking.

## Try It Out

### 1. Explore the API

```bash
cd project/backend

# Start server
./backend-server

# In another terminal, test endpoints
curl http://localhost:8080/api/health
curl http://localhost:8080/api/users
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"test"}'
```

### 2. Explore the Frontend

```bash
cd project/frontend

# Start dev server
npm start

# Opens http://localhost:3000 in browser
```

**Try the UI**:
- View server health status
- Click "Fetch Users" to load user list
- Login with username "alice" or "bob" (any password)

### 3. Review API Contract

```bash
# See endpoint definitions
cat memory/api-contracts.md
```

This document defines the contract between backend and frontend.

### 4. Launch an Agent

```bash
# Start backend engineer
adp run codex --workspace web-app --profile backend-dev

# Or start frontend engineer
adp run codex --workspace web-app --profile frontend-dev
```

### 5. Assign a Task

Once an agent is running:

```
User: "Work on task T7 - add pagination to users endpoint"

Agent: [reads tasks.yaml, implements pagination, updates API contract]
```

## Task Flow Example

From `tasks.yaml`:

```yaml
- id: T3
  title: "Implement login endpoint"
  assignee: backend-dev
  priority: high

- id: T4
  title: "Create API client module"
  assignee: frontend-dev
  depends_on: [T1, T2, T3]  # Waits for backend
  
- id: T5
  title: "Build login form component"
  assignee: frontend-dev
  depends_on: [T4]  # Waits for API client
```

This creates a dependency chain: Frontend waits for backend API readiness.

## Development Phases

From `phases.yaml`:

- **Phase 1 (API Foundation)**: Backend endpoints with tests
- **Phase 2 (Frontend Integration)**: React UI consuming API
- **Phase 3 (Feature Enhancement)**: Pagination, search
- **Phase 4 (Integration & Deployment)**: End-to-end testing

Each phase has clear milestones and depends on the previous phase.

## API Endpoints

Defined in `memory/api-contracts.md`:

### GET /api/health
Server health check

### GET /api/users
List all users (returns array of {id, username, email})

### POST /api/auth/login
Authenticate user (returns {token, expires_at})

## Testing

```bash
# Backend tests
cd project/backend
go test ./...
# Output: 6/6 tests passed

# Frontend tests
cd project/frontend
npm test
```

## Demo Credentials

For the login endpoint:
- **Username**: `alice` or `bob`
- **Password**: any value (demo mode, not validated)

## Next Steps

- **Modify API Contract**: Edit `memory/api-contracts.md` and implement
- **Add New Endpoint**: Update backend, document contract, integrate in frontend
- **Customize Agents**: Adjust profiles and prompts for your workflow
- **Try Other Examples**:
  - `examples/game-development` - Game engine with physics
  - `examples/data-pipeline` - ETL pipeline with quality checks

## Validation

Run the workspace doctor to verify configuration:

```bash
adp workspace doctor web-app
```

All checks should pass ✓

## Time Budget Verification

- **Setup**: < 4 minutes (`./setup.sh` + npm install)
- **Backend Start**: < 5 seconds
- **Frontend Start**: < 30 seconds (first time)
- **Total**: Meets the "5-minute rule" ✓

## Architecture Highlights

### Backend (Go)
- Standard library HTTP server (no frameworks)
- JSON request/response handling
- CORS middleware for development
- Consistent error format: `{"error": "message"}`

### Frontend (React)
- Functional components with hooks
- Centralized API client
- Loading and error state handling
- Responsive CSS design

### Communication
- REST API over HTTP
- JSON payload format
- CORS enabled for cross-origin requests

## Learn More

- [ADP Documentation](../../docs/)
- [Workspace Configuration Guide](../../docs/workspace.md)
- [Agent Orchestration Patterns](../../docs/agent-patterns.md)
- [Task Management](../../docs/tasks.md)

## Troubleshooting

**Setup fails?**
- Verify Go 1.21+: `go version`
- Verify Node.js 16+: `node --version`
- Check ports 8080 and 3000 are available

**Backend won't start?**
- Check if port 8080 is in use: `lsof -i :8080`
- Review backend logs for errors

**Frontend won't connect?**
- Verify backend is running on port 8080
- Check browser console for CORS errors
- Verify API_BASE_URL in frontend/.env

**Agent doesn't see tasks?**
- Verify workspace registered: `adp workspace list`
- Check tasks.yaml exists and is valid YAML
- Review agent profile: `cat profiles/backend-dev.yaml`
