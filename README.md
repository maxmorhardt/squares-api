# Squares API

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Gin](https://img.shields.io/badge/gin-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Kubernetes](https://img.shields.io/badge/kubernetes-%23326ce5.svg?style=for-the-badge&logo=kubernetes&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-316192?style=for-the-badge&logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/redis-%23DD0031.svg?style=for-the-badge&logo=redis&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)

## Overview
A real-time collaborative grid application backend built with Go and Gin. Features OIDC authentication, role-based authorization, and Server-Sent Events for real-time updates across multiple clients.

## Features
- **RESTful API** for grid management and cell updates
- **OIDC Authentication** with JWT token validation
- **Role-based Authorization** with configurable permissions
- **Real-time Updates** via Server-Sent Events (SSE)
- **Redis Pub/Sub** for horizontal scaling and cross-instance broadcasting
- **PostgreSQL** database with GORM ORM
- **Swagger Documentation** for API endpoints
- **Kubernetes Ready** with Helm charts and Docker support

## Architecture
The application uses Server-Sent Events (SSE) to broadcast real-time updates to connected clients. When an HTTP request triggers a grid update, the change is published to Redis Pub/Sub, which then broadcasts the update to all connected SSE clients across multiple application instances.

## Dependencies
This application requires the following services to be deployed:
- **OIDC Provider** (e.g., Keycloak) for authentication
- **PostgreSQL** database for data persistence
- **Redis** for pub/sub messaging and session management

## Development
1. Set up environment variables (see `config/` directory)
2. Start required services (PostgreSQL, Redis `kubectl port-forward svc/redis-master 6379:6379 -n maxstash-global`, OIDC provider)
3. Run `go run cmd/main.go`

## Deployment
The application includes Helm charts in the `helm/` directory for Kubernetes deployment. Configure your environment-specific values and deploy with:
```bash
helm install squares-api ./helm/squares-api
```