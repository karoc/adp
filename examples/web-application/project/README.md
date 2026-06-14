# Web Application Example Project

This is a full-stack web application with a Go backend REST API and React frontend.

## Architecture

```
project/
├── backend/         # Go REST API server
│   ├── main.go      # HTTP server and routes
│   └── api/
│       ├── handlers.go       # Request handlers
│       └── handlers_test.go  # API tests
│
└── frontend/        # React web application
    ├── package.json
    ├── public/
    │   └── index.html
    └── src/
        ├── App.js           # Main app component
        ├── api.js           # API client
        ├── App.test.js      # Component tests
        └── api.test.js      # API client tests
```

## Running the Application

### Backend (Terminal 1)

```bash
cd backend
go build
./backend

# Server starts on http://localhost:8080
```

### Frontend (Terminal 2)

```bash
cd frontend
npm install
npm start

# App opens at http://localhost:3000
```

## API Endpoints

- **GET /api/health** - Server health status
- **GET /api/users** - List all users
- **POST /api/auth/login** - User authentication

## Testing

```bash
# Backend tests
cd backend && go test ./...

# Frontend tests
cd frontend && npm test
```

## Demo Credentials

- Username: `alice` or `bob`
- Password: any value (demo mode)

## Features

- REST API with Go
- React frontend with hooks
- CORS enabled for development
- JSON API responses
- Error handling
- Unit tests for both backend and frontend
