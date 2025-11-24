# Meerkat CRM - a simple CRM for the personal life

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org)
[![React Version](https://img.shields.io/badge/React-19.2-61DAFB?logo=react)](https://reactjs.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.9-3178C6?logo=typescript)](https://www.typescriptlang.org)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

**Tech Stack:**  
![Go](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white)
![Gin](https://img.shields.io/badge/Gin-00ADD8?logo=go&logoColor=white)
![React](https://img.shields.io/badge/React-61DAFB?logo=react&logoColor=black)
![TypeScript](https://img.shields.io/badge/TypeScript-3178C6?logo=typescript&logoColor=white)
![Material--UI](https://img.shields.io/badge/Material--UI-007FFF?logo=mui&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-003B57?logo=sqlite&logoColor=white)

## About the project
Meerkat CRM is a simple self-hosted solution to keep track of all your contacts. As your digital Rolodex it reminds of of birthdays, helps you to remember dietary habits as well as names of spouses of contacts - and much more.

## Features
- Contact management
    - add and search contacts
    - details of contacts
    - group contacts by circles (e.g. friends, family, work)
- Notes and activities
    - social network style timeline for contacts
    - notes assigned to individual contacts
    - general notes (for e.g. journaling)
    - activities with one or multiple contacts
- Reminders
    - Keep in touch through reminders and get e-mail notifications
    - Birthday notifications
- Usability
    - I18N (multiple languages)

## Installation

## Contributing

### Development
To set up this repository for development, follow these steps:

1. **Clone the repository:**
    ```sh
    git clone https://github.com/fbuchner/meerkat-crm.git
    cd meerkat
    ```

1. **Run the backend:**
Ensure you have [Go](https://golang.org/doc/install) installed. Then, set up your environment configuration:
    ```sh
    cd backend
    # Copy the example environment file and configure it with your settings
    cp .env.example my_environment.env
    # Edit my_environment.env with your actual configuration values
    
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
   
   See [MIGRATIONS.md](backend/MIGRATIONS.md) for detailed migration documentation.

1. **Run the frontend (in a second terminal):**
    ```sh
    cd frontend
    yarn install
    yarn serve
    ```

You can also use the Debug button in [Visual Studio Code](https://code.visualstudio.com/) as configured in the launch.json file.
