#!/bin/bash
# Build Elm app for development (non-optimized for faster builds)
cd "$(dirname "$0")"
elm make src/Main.elm --output=../web/elm.min.js
echo "✓ Built Elm app to web/elm.min.js"
echo "→ Access at http://localhost:8080"
