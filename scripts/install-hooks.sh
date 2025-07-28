#!/bin/bash

# Script to install git hooks for the provider-gitea repository
# Run this after cloning to enable large file protection

echo "🔧 Installing git hooks for provider-gitea..."

# Create hooks directory if it doesn't exist
mkdir -p .git/hooks

# Install pre-commit hook
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash

# Pre-commit hook to prevent large files and binary artifacts from being committed
# This prevents the repository from growing with unnecessary large files

echo "🔍 Checking for large files and binary artifacts..."

# Configuration
MAX_FILE_SIZE=10485760  # 10MB in bytes
BLOCKED_EXTENSIONS=("*.xpkg" "*.tar.gz" "*.zip" "*.exe" "*.dll" "*.so" "*.dylib")
BLOCKED_PATHS=("provider" "bin/provider" "_output/" "_build/" ".cache/")

# Check file sizes
large_files=$(git diff --cached --name-only --diff-filter=A | xargs -I {} find {} -type f -size +${MAX_FILE_SIZE}c 2>/dev/null)

if [ -n "$large_files" ]; then
    echo "❌ ERROR: The following files exceed the 10MB limit:"
    echo "$large_files" | while read file; do
        size=$(du -h "$file" | cut -f1)
        echo "  - $file ($size)"
    done
    echo ""
    echo "💡 Large files should not be committed to git. Consider:"
    echo "  - Adding them to .gitignore"
    echo "  - Using Git LFS for necessary large files"
    echo "  - Removing them if they are build artifacts"
    echo ""
    exit 1
fi

# Check for blocked file extensions
for ext in "${BLOCKED_EXTENSIONS[@]}"; do
    blocked_files=$(git diff --cached --name-only --diff-filter=A | grep -E "$ext$" 2>/dev/null || true)
    if [ -n "$blocked_files" ]; then
        echo "❌ ERROR: Blocked file extension detected: $ext"
        echo "$blocked_files" | while read file; do
            echo "  - $file"
        done
        echo ""
        echo "💡 These file types should not be committed. Add them to .gitignore."
        echo ""
        exit 1
    fi
done

# Check for blocked paths
for path in "${BLOCKED_PATHS[@]}"; do
    blocked_files=$(git diff --cached --name-only --diff-filter=A | grep -E "^$path" 2>/dev/null || true)
    if [ -n "$blocked_files" ]; then
        echo "❌ ERROR: Blocked path detected: $path"
        echo "$blocked_files" | while read file; do
            echo "  - $file"
        done
        echo ""
        echo "💡 These paths contain build artifacts. Add them to .gitignore."
        echo ""
        exit 1
    fi
done

# Check for binary files (heuristic check)
binary_files=$(git diff --cached --numstat | awk '$1 == "-" && $2 == "-" {print $3}' | head -10)
if [ -n "$binary_files" ]; then
    echo "⚠️  WARNING: The following binary files are being committed:"
    echo "$binary_files" | while read file; do
        echo "  - $file"
    done
    echo ""
    echo "💡 Consider if these binary files should really be in version control."
    echo "   Press Ctrl+C to abort, or Enter to continue..."
    read -r
fi

echo "✅ Pre-commit checks passed!"
exit 0
EOF

# Make the hook executable
chmod +x .git/hooks/pre-commit

echo "✅ Git hooks installed successfully!"
echo ""
echo "📋 Installed hooks:"
echo "  - pre-commit: Prevents large files and binary artifacts"
echo ""
echo "💡 The pre-commit hook will now check for:"
echo "  - Files larger than 10MB"
echo "  - Blocked extensions: *.xpkg, *.tar.gz, *.zip, binaries"
echo "  - Blocked paths: provider, bin/, _output/, _build/, .cache/"
echo "  - Binary files (with warning)"
echo ""
echo "🎉 Repository is now protected against large file commits!"