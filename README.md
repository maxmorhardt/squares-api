# Squares API

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Gin](https://img.shields.io/badge/gin-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Kubernetes](https://img.shields.io/badge/kubernetes-%23326ce5.svg?style=for-the-badge&logo=kubernetes&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-316192?style=for-the-badge&logo=postgresql&logoColor=white)
![NATS](https://img.shields.io/badge/NATS-27AAE1?style=for-the-badge&logo=nats.io&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)

## Overview
A real-time football squares pool API built with Go and Gin. Supports contest lifecycle management, square claiming, automated winner calculation, and WebSocket-based real-time updates.

## Features
- **Contest Management** - Create and manage football squares contests with 10x10 grids
- **Contest Lifecycle** - State machine: ACTIVE → Q1 → Q2 → Q3 → Q4 → FINISHED (or DELETED at any time)
- **Square Claiming** - Users can claim squares during ACTIVE state only
- **Automatic Label Randomization** - X/Y axis labels (0-9) are randomly shuffled when transitioning to Q1
- **Quarter Results** - Record scores and automatically calculate winners based on last digit matching
- **Winner Tracking** - Stores winner username, first name, and last name for each quarter
- **Real-time Updates** - WebSocket connections for live contest, square, and quarter result updates
- **OIDC Authentication** - JWT token validation with username, first name, and last name claims
- **NATS Messaging** - Scales horizontally with cross-instance WebSocket broadcasting
- **PostgreSQL** - Data persistence with GORM ORM and automatic migrations
- **Swagger Documentation** - Auto-generated API documentation

## Dependencies
This application requires the following services to be deployed:
- **OIDC Provider** (e.g., Keycloak) for authentication
- **PostgreSQL** database for data persistence
- **NATS** for pub/sub messaging and real-time event broadcasting
- **SMTP Server** for email notifications

## Development
1. Set up environment variables
2. Start required services (PostgreSQL, NATS, OIDC provider)
3. Run `go run cmd/main.go`