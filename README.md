# Meerkat CRM - a simple CRM for the personal life

<p align="center">
  <img src="assets/meerkat-crm-logo.svg" alt="Meerkat CRM Logo" width="180" />
</p>

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org)
[![React Version](https://img.shields.io/badge/React-19.2-61DAFB?logo=react)](https://reactjs.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.9-3178C6?logo=typescript)](https://www.typescriptlang.org)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

## About the project
Meerkat CRM (Contact Relationship Management) is a simple self-hosted solution to keep track of all your contacts. As your digital rolodex it reminds you of birthdays, helps you to remember dietary habits as well as names of spouses of contacts - and much more.

## Features
- Contact management
    - add and search contacts
    - group contacts by circles (e.g. friends, family, work)
    - store relationships of contacts (e.g. spouses, children)
- Notes and activities
    - social network style timeline for contacts
    - notes assigned to individual contacts
    - activities with one or multiple contacts
    - general notes (for e.g. journaling)
- Reminders
    - Keep in touch through reminders and get e-mail notifications
    - See upcoming birthdays
- Usability
    - Multiple languages (currently EN and DE)
    - Light and dark mode

## Installation

### Docker (Recommended)

The easiest way to run Meerkat CRM is with Docker Compose:

1. **Clone the repository:**
    ```sh
    git clone https://github.com/fbuchner/meerkat-crm.git
    cd meerkat-crm
    ```

2. **Configure environment:**
    ```sh
    # Copy the Docker environment template
    cp .env.docker.example .env.docker
    
    # Edit with your settings
    nano .env.docker
    ```

3. **Start the containers:**
    ```sh
    docker compose up -d
    ```

4. **Access the application:**
    Open http://localhost:3000 in your browser.


## Contributing

### Development
To set up this repository for development, follow these steps:

1. **Clone the repository:**
    ```sh
    git clone https://github.com/fbuchner/meerkat-crm.git
    cd meerkat-crm
    ```

1. **Run the backend:**

Ensure you have [Go](https://golang.org/doc/install) installed. Then, set up your environment configuration:
   ```sh
    cd backend
    # Copy the example environment file and configure it with your settings
    cp .env.example my_environment.env
    
    # Install dependencies and run
    go mod tidy
    source my_environment.env
    go run main.go
   ```
   The project uses an SQLite database for storage. Database migrations run automatically on startup.
   
   **Database Migrations:**
   ```sh
   # View migration commands
   make help
   
   # Check current migration status
   make migrate-status
   
   # Create a new migration
   make migrate-create NAME=your_migration_name
   ```

   **Run Tests:**
   ```sh
   go test ./...
   ```  


1. **Run the frontend (in a second terminal):**
   ```sh
   cd frontend
   # Copy the example environment file and configure it with your settings
   cp .env.example my_environment.env

   yarn install
   yarn start
   ```

## Alternative software
Notable other personal CRM systems are
* [MonicaHQ](https://www.monicahq.com/) (Open Source, development has stalled; the new version chandler is available at beta.monicahq.com)
* [Dex](https://getdex.com/) (paid offering with social media integration)
* [Clay](https://clay.earth/) (paid offering with focus on automation)

Other software that can be used to build or configure something similar includes
* [Twenty](https://twenty.com/) (Open Source "classic" CRM system)
