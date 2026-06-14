#!/usr/bin/env bash
set -euo pipefail

# Web Application Example Setup Script
# Automates workspace registration and validation for full-stack example

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EXAMPLE_NAME="web-app"
PROJECT_DIR="${SCRIPT_DIR}/project"
BACKEND_DIR="${PROJECT_DIR}/backend"
FRONTEND_DIR="${PROJECT_DIR}/frontend"

echo "╔════════════════════════════════════════════════╗"
echo "║   ADP Web Application Example Setup            ║"
echo "╚════════════════════════════════════════════════╝"
echo ""

# Check prerequisites
echo "🔍 Checking prerequisites..."

# Check if adp command exists
if ! command -v adp &> /dev/null; then
    echo "❌ ERROR: 'adp' command not found"
    echo ""
    echo "Please install ADP first:"
    echo "  • See docs/install.md for installation instructions"
    echo ""
    exit 1
fi
echo "   ✅ adp command found: $(command -v adp)"

# Check Go installation
if ! command -v go &> /dev/null; then
    echo "❌ ERROR: Go not installed"
    echo ""
    echo "Please install Go 1.21+ first:"
    echo "  • Visit: https://go.dev/doc/install"
    echo ""
    exit 1
fi
echo "   ✅ Go installed: $(go version)"

# Verify Go version is 1.21+
GO_VERSION=$(go version | grep -oP 'go1\.\d+' || echo "go1.0")
GO_MAJOR=$(echo "$GO_VERSION" | grep -oP '\d+' | head -1)
GO_MINOR=$(echo "$GO_VERSION" | grep -oP '\d+' | tail -1)

if [[ "$GO_MAJOR" -lt 1 ]] || [[ "$GO_MAJOR" -eq 1 && "$GO_MINOR" -lt 21 ]]; then
    echo "   ❌ ERROR: Go 1.21+ required (you have $GO_VERSION)"
    echo ""
    echo "Please upgrade Go:"
    echo "  • Visit: https://go.dev/doc/install"
    echo ""
    exit 1
fi

# Check Node.js installation
if ! command -v node &> /dev/null; then
    echo "❌ ERROR: Node.js not installed"
    echo ""
    echo "Please install Node.js 16+ first:"
    echo "  • Visit: https://nodejs.org/"
    echo "  • Or use nvm: nvm install --lts"
    echo ""
    exit 1
fi
echo "   ✅ Node.js installed: $(node --version)"

# Check npm
if ! command -v npm &> /dev/null; then
    echo "❌ ERROR: npm not installed"
    exit 1
fi
echo "   ✅ npm installed: $(npm --version)"

echo ""

# Initialize ADP if needed
echo "📦 Initializing ADP..."
if [ ! -d "${ADP_HOME:-$HOME/.adp}" ]; then
    export ADP_HOME="${HOME}/.adp"
    adp init
    echo "   ✅ ADP initialized at: $ADP_HOME"
else
    echo "   ✅ ADP already initialized at: ${ADP_HOME:-$HOME/.adp}"
fi
echo ""

# Build backend
echo "🔨 Building backend..."
cd "$BACKEND_DIR"
if go build -o backend-server; then
    echo "   ✅ Backend built successfully"
else
    echo "   ❌ Backend build failed"
    exit 1
fi

# Run backend tests
echo "   Running backend tests..."
if go test ./... > /tmp/backend-test.log 2>&1; then
    echo "   ✅ Backend tests passed"
else
    echo "   ❌ Backend tests failed:"
    cat /tmp/backend-test.log
    rm -f /tmp/backend-test.log
    exit 1
fi
rm -f /tmp/backend-test.log

cd "$SCRIPT_DIR"
echo ""

# Install frontend dependencies
echo "📦 Installing frontend dependencies..."
cd "$FRONTEND_DIR"
if npm install --silent > /tmp/npm-install.log 2>&1; then
    echo "   ✅ Frontend dependencies installed"
else
    echo "   ⚠️  npm install had warnings (check /tmp/npm-install.log)"
fi
rm -f /tmp/npm-install.log

# Run frontend tests (optional, may take time)
echo "   Running frontend tests..."
if npm test -- --passWithNoTests > /tmp/frontend-test.log 2>&1; then
    echo "   ✅ Frontend tests passed"
else
    echo "   ⚠️  Frontend tests had issues (check /tmp/frontend-test.log)"
fi
rm -f /tmp/frontend-test.log

cd "$SCRIPT_DIR"
echo ""

# Register workspace
echo "🔧 Registering ADP workspace..."

# Check if workspace already exists
if adp workspace list 2>/dev/null | grep -q "^${EXAMPLE_NAME} "; then
    echo "   ⚠️  Workspace '${EXAMPLE_NAME}' already exists"
    read -p "   Remove and recreate? (y/N): " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        adp workspace remove "$EXAMPLE_NAME"
        adp workspace add "$EXAMPLE_NAME" "$SCRIPT_DIR"
        echo "   ✅ Workspace recreated: $EXAMPLE_NAME"
    else
        echo "   Using existing workspace."
    fi
else
    adp workspace add "$EXAMPLE_NAME" "$SCRIPT_DIR"
    echo "   ✅ Workspace registered: $EXAMPLE_NAME"
fi
echo ""

# Verify setup
echo "🔍 Verifying workspace configuration..."
if adp workspace doctor "$EXAMPLE_NAME" > /tmp/adp-web-app-doctor.txt 2>&1; then
    echo "   ✅ Workspace diagnostics passed"
else
    echo "   ⚠️  Some diagnostics failed:"
    cat /tmp/adp-web-app-doctor.txt
    echo ""
    echo "   This may be OK - review the output above."
fi
rm -f /tmp/adp-web-app-doctor.txt
echo ""

# Success summary
echo "╔════════════════════════════════════════════════╗"
echo "║            Setup Complete! 🎉                  ║"
echo "╚════════════════════════════════════════════════╝"
echo ""
echo "📚 Next Steps:"
echo ""
echo "1. Review the API contracts:"
echo "   cat ${SCRIPT_DIR}/memory/api-contracts.md"
echo ""
echo "2. Start the backend (Terminal 1):"
echo "   cd ${BACKEND_DIR}"
echo "   ./backend-server"
echo "   # Server runs on http://localhost:8080"
echo ""
echo "3. Start the frontend (Terminal 2):"
echo "   cd ${FRONTEND_DIR}"
echo "   npm start"
echo "   # App opens at http://localhost:3000"
echo ""
echo "4. Launch an agent:"
echo "   adp run codex --workspace ${EXAMPLE_NAME} --profile backend-dev"
echo "   adp run codex --workspace ${EXAMPLE_NAME} --profile frontend-dev"
echo ""
echo "🌐 This example demonstrates:"
echo "   • Full-stack development (Go + React)"
echo "   • API contract-first design"
echo "   • Task dependencies (frontend waits for backend)"
echo "   • Agent specialization (backend vs frontend)"
echo ""
echo "⏱️  Setup time: ~3-4 minutes"
echo "📖 Documentation: ${SCRIPT_DIR}/README.md"
echo ""
