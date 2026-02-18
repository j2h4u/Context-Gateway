#!/bin/sh
# Context Gateway quickstart installer
# Usage: curl -sL https://raw.githubusercontent.com/j2h4u/Context-Gateway/main/docs/quickstart/install.sh | sh
set -e

REPO="https://raw.githubusercontent.com/j2h4u/Context-Gateway/main/docs/quickstart"
DIR="/opt/docker/context-gateway"

mkdir -p "$DIR/logs"
for f in docker-compose.yml config.yaml telemetry-report.py claude-toggle-gateway.py AGENTS.md; do
    curl -sL "$REPO/$f" -o "$DIR/$f"
done
chmod +x "$DIR/telemetry-report.py" "$DIR/claude-toggle-gateway.py"

echo "Installed to $DIR"
echo ""
echo "Next steps:"
echo "  cd $DIR && docker compose up -d"
echo "  $DIR/claude-toggle-gateway.py"
echo ""
echo "Verify: curl http://localhost:18080/health"
