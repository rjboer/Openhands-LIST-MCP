#!/bin/bash
# OpenHands setup script

set -e
set -x


mkdir -p /workspace/.openhands
cp /workspace/cto-evaluation-tool/.openhands/setup.sh /workspace/.openhands/
touch /workspace/.openhands/pre-commit.sh
chmod +x /workspace/.openhands/pre-commit.sh
# Install GitHub CLI
sudo apt update && sudo apt install -y gh curl tar

# Install Go
 GO_VERSION="1.22.3"
 GO_TARBALL="go${GO_VERSION}.linux-amd64.tar.gz"
 GO_URL="https://go.dev/dl/${GO_TARBALL}"

 curl -OL "${GO_URL}"
 sudo rm -rf /usr/local/go
 sudo tar -C /usr/local -xzf "${GO_TARBALL}"
 rm "${GO_TARBALL}"

# # Set up Go environment rooted in /workspace
 export GOROOT="/usr/local/go"
 export GOPATH="/workspace/go"
 export PATH="$GOROOT/bin:$GOPATH/bin:$PATH"

# Ensure Go workspace directories exist
 mkdir -p "$GOPATH"/{bin,pkg,src}
add .openhands_instructions
echo "Evaluate this Go project for compliance with the requirements in cto_tool_requirements.md." >  /workspace/.openhands_instructions
cd /workspace/Openhands-LIST-MCP
touch /workspace/.openhands/setup.sh

# Print confirmation
 go version
 go env

