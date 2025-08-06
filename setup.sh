#!/bin/bash

echo "Building ParaXm..."
if ! go build -o paraxm cmd/main.go; then
    echo "Build failed!"
    exit 1
fi

echo "Build successful!"

if [[ -d "$HOME/.local/bin" ]]; then
    INSTALL_DIR="$HOME/.local/bin"
elif [[ -d "$HOME/bin" ]]; then
    INSTALL_DIR="$HOME/bin"
elif [[ -d "/usr/local/bin" ]]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="$HOME/.local/bin"
fi

if [[ ! -d "$INSTALL_DIR" ]]; then
    echo "Creating directory $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR" || {
        echo "Failed to create installation directory"
        exit 1
    }
fi

echo "Installing ParaXm to $INSTALL_DIR"
if cp paraxm "$INSTALL_DIR/" 2>/dev/null; then
    echo "Installation successful!"
    echo "You can now run ParaXm with: paraxm -h"
elif sudo cp paraxm "$INSTALL_DIR/" 2>/dev/null; then
    echo "Installation successful!"
    echo "You can now run ParaXm with: paraxm -h"
else
    echo "Installation failed!."
    exit 1
fi