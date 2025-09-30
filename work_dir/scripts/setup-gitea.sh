#!/bin/bash
set -e

echo "🔧 Setting up Gitea repository for Nephio packages..."

# Wait for Gitea to be ready
echo "⏳ Waiting for Gitea to start..."
until curl -f http://gitea:3000 > /dev/null 2>&1; do
    sleep 2
done

echo "✅ Gitea is ready"

# Create initial user (admin)
curl -X POST "http://gitea:3000/api/v1/admin/users" \
  -H "Content-Type: application/json" \
  -u "admin:admin" \
  -d '{
    "username": "nephio",
    "email": "nephio@example.com",
    "password": "nephio123",
    "must_change_password": false
  }' || echo "User may already exist"

# Create repository
curl -X POST "http://gitea:3000/api/v1/user/repos" \
  -H "Content-Type: application/json" \
  -u "nephio:nephio123" \
  -d '{
    "name": "packages",
    "description": "Nephio package repository",
    "private": false,
    "auto_init": true
  }' || echo "Repository may already exist"

echo "✅ Gitea setup complete"
echo "📦 Repository URL: http://gitea:3000/nephio/packages.git"