# Life Dashboard Server

This is a backend service built in Go to interact with the Google ecosystem, allowing users to access and manipulate their data across different Google services. Built with industry best practices in mind, it utilizes an onion architecture with Data Transfer Objects (DTOs) to maintain a strict separation of concerns.

## Features

* **Google Auth Integration:** Uses OAuth2 and the Google login flow for secure authentication, storing user credentials in a MySQL database. All OAuth tokens are AES-256 encrypted before persistence.
* **Task Management:** Two-way synchronization with Google Tasks. Supports creating, updating, moving, and deleting tasks and subtasks directly through the API.
* **Calendar Events:** Read-only access to Google Calendar events based on designated start and end time windows.
* **Time "Zones":** Custom logic for managing daily time blocks (Zones) with specific schedules, days active, and color coding.
* **Secure Sessions:** Issues secure, HTTP-only JWT session cookies with automated, hashed refresh token handling.

## Tech Stack

* **Language:** Go 1.24
* **Routing:** Chi Router
* **Database:** MySQL via GORM
* **Dependency Injection:** Uber Dig
* **Configuration:** Koanf for YAML files and godotenv for `.env` management

## Architecture

The application is heavily structured around the Onion/Clean Architecture pattern to ensure modularity and ease of testing:
* **`cmd/http`**: Application entry point and initialization.
* **`internal/domain`**: Core database models and Data Transfer Objects (DTOs).
* **`internal/repository`**: Data persistence layer interacting with the SQL database.
* **`internal/service`**: Core business logic and external Google API communication.
* **`internal/handlers`**: HTTP handlers, Chi routing, and middleware (including OIDC validation for Cloud Run).

## Setup & Configuration

The server relies on a mix of environment variables and a YAML configuration file. By default, it looks for `config.dev.yaml` locally. 

Create a `.env` file in the root directory for your secrets (see `.env.example`  for an example).

## Running the Application Locally:


```Bash
go run ./cmd/http
Using Docker:
A multi-stage Dockerfile is included for generating a minimal, statically linked Alpine container.
```

```Bash
docker build -t life-dashboard-server .
docker run -p 8080:8080 --env-file .env life-dashboard-server
```
