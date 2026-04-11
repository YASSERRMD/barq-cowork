#!/bin/bash
# Barq Cowork — macOS First-Run Setup Helper
# Double-click this file to prepare Barq Cowork after installing.
# This is only needed once. You can delete this file afterwards.

APP="/Applications/Barq Cowork.app"

echo "========================================"
echo "  Barq Cowork — macOS Setup Helper"
echo "========================================"
echo ""

if [ ! -d "$APP" ]; then
  echo "ERROR: 'Barq Cowork.app' not found in /Applications."
  echo ""
  echo "Please drag 'Barq Cowork.app' from the DMG to /Applications first,"
  echo "then run this helper again."
  echo ""
  read -p "Press Enter to close..."
  exit 1
fi

echo "Removing macOS quarantine flag..."
xattr -cr "$APP" 2>/dev/null

echo "Fixing binary permissions..."
chmod +x "$APP/Contents/MacOS/barq-cowork" 2>/dev/null
chmod +x "$APP/Contents/MacOS/barq-coworkd" 2>/dev/null

echo ""
echo "Done! You can now launch Barq Cowork normally from Launchpad or /Applications."
echo ""
echo "You can delete this helper file — it only needs to run once."
echo ""
read -p "Press Enter to close..."
