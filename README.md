# Squares API

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Gin](https://img.shields.io/badge/gin-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Kubernetes](https://img.shields.io/badge/kubernetes-%23326ce5.svg?style=for-the-badge&logo=kubernetes&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-316192?style=for-the-badge&logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/redis-%23DD0031.svg?style=for-the-badge&logo=redis&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)

## Overview
A real-time football squares pool API built with Go and Gin. Supports contest lifecycle management, square claiming, automated winner calculation, and WebSocket-based real-time updates. Features OIDC authentication with JWT tokens.

## Features
- **Contest Management** - Create and manage football squares contests with 10x10 grids
- **Contest Lifecycle** - State machine: ACTIVE → Q1 → Q2 → Q3 → Q4 → FINISHED (or DELETED at any time)
- **Square Claiming** - Users can claim squares during ACTIVE state only
- **Automatic Label Randomization** - X/Y axis labels (0-9) are randomly shuffled when transitioning to Q1
- **Quarter Results** - Record scores and automatically calculate winners based on last digit matching
- **Winner Tracking** - Stores winner username, first name, and last name for each quarter
- **Real-time Updates** - WebSocket connections for live contest, square, and quarter result updates
- **OIDC Authentication** - JWT token validation with username, first name, and last name claims
- **Redis Pub/Sub** - Scales horizontally with cross-instance WebSocket broadcasting
- **PostgreSQL** - Data persistence with GORM ORM and automatic migrations
- **Swagger Documentation** - Auto-generated API documentation
- **Kubernetes Ready** - Includes Helm charts and Docker support

## Contest Flow
1. **ACTIVE** - Contest is open for square claiming. Users select squares and provide a value.
2. **Q1-Q4** - After transition to Q1, squares become immutable and labels are randomized. Record scores for each quarter to automatically determine winners.
3. **FINISHED** - Contest is complete with all quarter results recorded.
4. **DELETED** - Contest can be deleted at any time.

## Winner Calculation
Winners are determined by matching the last digit of each team's score to the randomized X and Y axis labels:
- Home team score last digit → X axis label position (column)
- Away team score last digit → Y axis label position (row)
- The square at (row, col) is the winner for that quarter

## Architecture
The application uses WebSockets for bidirectional real-time communication. Contest updates, square claims, and quarter results are published to Redis Pub/Sub, which broadcasts to all connected WebSocket clients across multiple application instances.

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