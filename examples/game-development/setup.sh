#!/usr/bin/env bash
set -euo pipefail

# Game Development Example Setup Script
# Automates workspace registration and validation

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EXAMPLE_NAME="game-dev"
PROJECT_DIR="${SCRIPT_DIR}/project"

echo "╔════════════════════════════════════════════════╗"
echo "║   ADP Game Development Example Setup           ║"
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
    echo "  • Or run: go build -o \$HOME/bin/adp ./cmd/adp"
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
    echo "  • Or use your package manager (brew install go, apt install golang-go, etc.)"
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
    echo "  • Or use your package manager"
    echo ""
    exit 1
fi

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

# Build the example project
echo "🔨 Building game engine project..."
cd "$PROJECT_DIR"
if go build -o game-engine; then
    echo "   ✅ Project built successfully"
else
    echo "   ❌ Build failed"
    exit 1
fi

# Run tests
echo "   Running tests..."
if go test ./... > /tmp/game-dev-test.log 2>&1; then
    echo "   ✅ All tests passed"
else
    echo "   ❌ Tests failed:"
    cat /tmp/game-dev-test.log
    rm -f /tmp/game-dev-test.log
    exit 1
fi
rm -f /tmp/game-dev-test.log

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
if adp workspace doctor "$EXAMPLE_NAME" > /tmp/adp-game-dev-doctor.txt 2>&1; then
    echo "   ✅ Workspace diagnostics passed"
else
    echo "   ⚠️  Some diagnostics failed:"
    cat /tmp/adp-game-dev-doctor.txt
    echo ""
    echo "   This may be OK - review the output above."
fi
rm -f /tmp/adp-game-dev-doctor.txt
echo ""

# Success summary
echo "╔════════════════════════════════════════════════╗"
echo "║            Setup Complete! 🎉                  ║"
echo "╚════════════════════════════════════════════════╝"
echo ""
echo "📚 Next Steps:"
echo ""
echo "1. Explore the workspace configuration:"
echo "   adp workspace show ${EXAMPLE_NAME}"
echo ""
echo "2. Review agent orchestration pattern:"
echo "   cat ${SCRIPT_DIR}/AGENTS.md"
echo ""
echo "3. Check available tasks:"
echo "   cat ${SCRIPT_DIR}/tasks.yaml"
echo ""
echo "4. Start an agent:"
echo "   adp run codex --workspace ${EXAMPLE_NAME}"
echo ""
echo "5. Try the game engine:"
echo "   cd ${PROJECT_DIR}"
echo "   ./game-engine --test"
echo "   ./game-engine         # Run 5-second demo"
echo ""
echo "🎮 This example demonstrates:"
echo "   • Agent specialization (gameplay vs graphics)"
echo "   • Task-based workflow with dependencies"
echo "   • Phase-based project organization"
echo "   • Domain-specific agent collaboration"
echo ""
echo "⏱️  Setup time: ~2 minutes"
echo "📖 Documentation: ${SCRIPT_DIR}/README.md"
echo ""
