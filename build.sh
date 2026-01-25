#!/usr/bin/env bash
set -e

# This script builds the Elm app in production mode, minifies it,
# copies static assets to ./web for Go embedding, then builds the Go binary.

ELM_SRC="src/Main.elm"
ELM_OUT="elm.js"
ELM_MIN="elm.min.js"
ELM_INDEX="index.html"
WEB_DIR="../web"
ELM_INDEX_SRC="index.html"

# 1. Clean any prior build output and build Elm app with optimizations (inside elm-app dir)
echo "[build.sh] Building Elm app in optimized mode..."
(cd elm-app && rm -f "$ELM_OUT")
(cd elm-app && elm make "$ELM_SRC" --optimize --output="$ELM_OUT")


# 2. Minify using uglifyjs
(cd elm-app && \
  echo "[build.sh] Minifying Elm output..." && \
  uglifyjs "$ELM_OUT" \
    --compress 'pure_funcs=[F2,F3,F4,F5,F6,F7,F8,F9,A2,A3,A4,A5,A6,A7,A8,A9],pure_getters,keep_fargs=false,unsafe_comps,unsafe' \
    | uglifyjs --mangle --output "$ELM_MIN" && \
  mkdir -p "$WEB_DIR" && \
  echo "[build.sh] Copying web assets to $WEB_DIR/ ..." && \
   cp "$ELM_MIN" "$WEB_DIR/elm.min.js" && \
   cp "$ELM_INDEX_SRC" "$WEB_DIR/index.html" && \
   echo "[build.sh] Cleaning temp files..." && \
   rm -f "$ELM_OUT" "$ELM_MIN")

# 3. Verify assets for Go embed
ls -l web/

go clean -cache -modcache

# 4. Build Go binary (with embed) from project root
cd "$(dirname "$0")"
echo "[build.sh] Building Go binary and embedding web assets..."
go build -o home-gate ./main.go

echo "[build.sh] Build complete! Elm and Go web assets embedded."
