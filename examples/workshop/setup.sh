#!/usr/bin/env bash
set -euo pipefail

# ADP Workshop Setup Script
# Automates environment preparation for the 30-minute hands-on workshop

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SAMPLE_PROJECT="${SCRIPT_DIR}/sample-project"
WORKSHOP_AGENT="${SCRIPT_DIR}/workshop-agent"

echo "╔════════════════════════════════════════════════╗"
echo "║       ADP Workshop Environment Setup           ║"
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
    echo "Please install Go first:"
    echo "  • Visit: https://go.dev/doc/install"
    echo "  • Or use your package manager (brew install go, apt install golang-go, etc.)"
    echo ""
    exit 1
fi

echo "   ✅ Go installed: $(go version)"
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

# Copy sample project to user workspace
echo "📂 Setting up sample project..."
WORKSHOP_PROJECT="${HOME}/adp-workshop-project"

if [ -d "$WORKSHOP_PROJECT" ]; then
    echo "   ⚠️  Directory already exists: $WORKSHOP_PROJECT"
    read -p "   Overwrite? (y/N): " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "   Skipping project setup. Using existing directory."
    else
        rm -rf "$WORKSHOP_PROJECT"
        cp -r "$SAMPLE_PROJECT" "$WORKSHOP_PROJECT"
        echo "   ✅ Sample project copied to: $WORKSHOP_PROJECT"
    fi
else
    cp -r "$SAMPLE_PROJECT" "$WORKSHOP_PROJECT"
    echo "   ✅ Sample project copied to: $WORKSHOP_PROJECT"
fi
echo ""

# Register workspace
echo "🔧 Registering ADP workspace..."
WORKSPACE_NAME="workshop"

# Check if workspace already exists
if adp workspace list 2>/dev/null | grep -q "^${WORKSPACE_NAME} "; then
    echo "   ⚠️  Workspace '${WORKSPACE_NAME}' already exists"
    read -p "   Remove and recreate? (y/N): " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        adp workspace remove "$WORKSPACE_NAME"
        adp workspace add "$WORKSPACE_NAME" "$WORKSHOP_PROJECT"
        echo "   ✅ Workspace recreated: $WORKSPACE_NAME"
    else
        echo "   Using existing workspace."
    fi
else
    adp workspace add "$WORKSPACE_NAME" "$WORKSHOP_PROJECT"
    echo "   ✅ Workspace registered: $WORKSPACE_NAME"
fi
echo ""

# Install workshop-agent to PATH
echo "🤖 Installing workshop agent..."
INSTALL_DIR="${HOME}/.local/bin"
mkdir -p "$INSTALL_DIR"

if [ -f "${INSTALL_DIR}/workshop-agent" ]; then
    echo "   ⚠️  workshop-agent already installed"
    read -p "   Overwrite? (y/N): " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cp "$WORKSHOP_AGENT" "${INSTALL_DIR}/workshop-agent"
        chmod +x "${INSTALL_DIR}/workshop-agent"
        echo "   ✅ workshop-agent updated"
    else
        echo "   Using existing workshop-agent."
    fi
else
    cp "$WORKSHOP_AGENT" "${INSTALL_DIR}/workshop-agent"
    chmod +x "${INSTALL_DIR}/workshop-agent"
    echo "   ✅ workshop-agent installed to: ${INSTALL_DIR}/workshop-agent"
fi

# Check if $INSTALL_DIR is in PATH
if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    echo ""
    echo "   ⚠️  ${INSTALL_DIR} is not in your PATH"
    echo ""
    echo "   Add this line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo "   export PATH=\"\${HOME}/.local/bin:\${PATH}\""
    echo ""
    read -p "   Continue anyway? (Y/n): " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Nn]$ ]]; then
        echo "Setup cancelled."
        exit 1
    fi
    # Temporarily add to PATH for this session
    export PATH="${INSTALL_DIR}:${PATH}"
fi
echo ""

# Verify setup
echo "🔍 Verifying setup..."
adp workspace doctor "$WORKSPACE_NAME" > /tmp/adp-workshop-doctor.txt 2>&1 || true

if grep -q "error" /tmp/adp-workshop-doctor.txt; then
    echo "   ⚠️  Some diagnostics failed (this may be OK for workshop):"
    cat /tmp/adp-workshop-doctor.txt
else
    echo "   ✅ Workspace diagnostics passed"
fi

rm -f /tmp/adp-workshop-doctor.txt
echo ""

# Success summary
echo "╔════════════════════════════════════════════════╗"
echo "║            Setup Complete! 🎉                  ║"
echo "╚════════════════════════════════════════════════╝"
echo ""
echo "📚 Next Steps:"
echo ""
echo "1. Open the workshop guide:"
echo "   • English: docs/workshop.md"
echo "   • 中文: docs/workshop.zh-CN.md"
echo ""
echo "2. Start with Module 1 (5 minutes):"
echo "   adp workspace show workshop"
echo "   adp doctor workshop"
echo ""
echo "3. Your workshop environment:"
echo "   • Workspace name: workshop"
echo "   • Project directory: ${WORKSHOP_PROJECT}"
echo "   • Fake agent: $(command -v workshop-agent || echo "${INSTALL_DIR}/workshop-agent")"
echo ""
echo "⏱️  Total workshop time: ~30 minutes"
echo ""
echo "💡 Tip: The sample project has an intentional bug."
echo "   You'll discover and fix it during Module 2!"
echo ""
