#!/bin/bash
set -e

echo "=== Hermix Dev Environment Setup ==="

# 1. Clone NodeBB fork
if [ ! -d "dev/nodebb" ]; then
  echo "[1/4] Cloning NodeBB v4.12.0..."
  mkdir -p dev
  git clone --depth 1 --branch v4.12.0 https://github.com/AYin-Z/NodeBB.git dev/nodebb
else
  echo "[1/4] NodeBB already cloned, updating..."
  cd dev/nodebb && git fetch origin && cd ../..
fi

# 2. Install NodeBB dependencies
echo "[2/4] Installing NodeBB dependencies..."
cd dev/nodebb
npm install --omit=dev
cd ../..

# 3. Link theme & plugin
echo "[3/4] Linking Hermix theme and plugin..."
cd theme && npm link && cd ..
cd plugin && npm link && cd ..
cd dev/nodebb
npm link nodebb-theme-hermix
npm link nodebb-plugin-hermix
cd ../..

# 4. Setup NodeBB (if not configured)
if [ ! -f "dev/nodebb/config.json" ]; then
  echo "[4/4] NodeBB setup (interactive or use config defaults)..."
  echo "  Run: cd dev/nodebb && ./nodebb setup"
  echo "  Or copy a config.json to dev/nodebb/config.json"
fi

echo ""
echo "=== Hermix Dev Environment Ready ==="
echo "  Start:      cd dev/nodebb && ./nodebb dev"
echo "  Build:      cd dev/nodebb && ./nodebb build"
echo "  Theme dev:  edit theme/ and restart NodeBB"
echo "  Plugin dev: edit plugin/ and restart NodeBB"
