#!/bin/bash

# Shared variables
BINARY_NAME="meshsync-snapshot"
TAR_NAME="meshsync-snapshot.tar.gz"
BINARY_PATH="bin/${BINARY_NAME}"
TAR_PATH="bin/${TAR_NAME}"
SHA256_PATH="bin/${TAR_NAME}.sha256"
TEMPLATE_FILE="meshsync-snapshot-local-template.yaml"

# Check dependencies function
check_dependencies() {
    echo "Checking dependencies..."
    local missing_deps=()
    
    # Check for required commands
    if ! command -v go &> /dev/null; then
        missing_deps+=("go")
    fi
    
    if ! command -v kubectl &> /dev/null; then
        missing_deps+=("kubectl")
    fi
    
    if ! command -v tar &> /dev/null; then
        missing_deps+=("tar")
    fi
    
    if ! command -v sha256sum &> /dev/null; then
        missing_deps+=("sha256sum")
    fi
    
    # Check for kubectl krew plugin
    if ! kubectl krew version &> /dev/null; then
        missing_deps+=("kubectl-krew")
    fi
    
    # Report results
    if [ ${#missing_deps[@]} -eq 0 ]; then
        echo "✓ All dependencies are available"
        return 0
    else
        echo "✗ Missing dependencies:"
        for dep in "${missing_deps[@]}"; do
            echo "  - $dep"
        done
        echo ""
        echo "Please install the missing dependencies before proceeding."
        echo "For kubectl krew installation: https://krew.sigs.k8s.io/docs/user-guide/setup/install/"
        return 1
    fi
}

# Build function
build() {
    echo "build"
    go build -o ${BINARY_PATH} cmd/meshsync/*.go

    echo "tar"
    tar -czvf ${TAR_PATH} ${BINARY_PATH}

    echo "sha256"
    sha256sum ${TAR_PATH} > ${SHA256_PATH}
    cat ${SHA256_PATH}
}

# Install function
install() {
    kubectl krew install --manifest=meshsync-snapshot.yaml
}

# Install local function
install_local() {
    # Check if tar file exists
    if [ ! -f "${TAR_PATH}" ]; then
        echo "⚠️  Warning: ${TAR_PATH} not found. Building first..."
        build
    elif [ ! -f "${SHA256_PATH}" ]; then
        echo "⚠️  Warning: ${SHA256_PATH} not found. Building first..."
        build
    else
        # Check if current hash matches stored hash
        current_hash=$(sha256sum "${TAR_PATH}" | cut -d' ' -f1)
        stored_hash=$(cut -d' ' -f1 "${SHA256_PATH}")
        
        if [ "${current_hash}" != "${stored_hash}" ]; then
            echo "⚠️  Warning: Hash mismatch detected!"
            echo "   Stored hash: ${stored_hash}"
            echo "   Real hash:  ${current_hash}"
            echo "   Rebuilding..."
            build
        fi
    fi
    
    # Get the current hash for template replacement
    current_hash=$(sha256sum "${TAR_PATH}" | cut -d' ' -f1)
    
    # Check if template file exists
    if [ ! -f "${TEMPLATE_FILE}" ]; then
        echo "❌ Error: Template file ${TEMPLATE_FILE} not found"
        exit 1
    fi
    
    # Create temporary manifest file (cross-platform)
    local_manifest=$(mktemp -t meshsync-manifest.XXXXXX.yaml)
    
    # Ensure cleanup on exit
    trap "rm -f '${local_manifest}'" EXIT
    
    # Create manifest from template with real hash
    echo "📝 Generating manifest from template..."
    sed "s/<sha256 hash  placeholder, populated from make, DO NOT UPDATE>/${current_hash}/g" "${TEMPLATE_FILE}" > "${local_manifest}"
    
    echo "🚀 Installing with generated manifest..."
    kubectl krew install --manifest="${local_manifest}" --archive="${TAR_PATH}"
}

# Uninstall function
uninstall() {
    kubectl krew uninstall meshsync-snapshot 2>/dev/null
}

# Help function
help() {
    echo "Usage: $0 {check_dependencies|build|install|install_local|uninstall}"
    echo ""
    echo "Commands:"
    echo "  check_dependencies Check if all required dependencies are installed"
    echo "  build             Build binary and create tar archive"
    echo "  install           Install using krew with remote manifest"
    echo "  install_local     Install using krew with local manifest and archive"
    echo "  uninstall         Uninstall the plugin"
}

# Main script logic
case "$1" in
    check_dependencies)
        check_dependencies
        ;;
    build)
        build
        ;;
    install)
        install
        ;;
    install_local)
        install_local
        ;;
    uninstall)
        uninstall
        ;;
    help|--help|-h)
        help
        ;;
    *)
        echo "Error: Unknown command '$1'"
        help
        exit 1
        ;;
esac