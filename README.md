# Meerkat CRM - a simple CRM for the personal life

<p align="center">
  <img src="assets/meerkat-crm-logo.svg" alt="Meerkat CRM Logo" width="180" />
</p>

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Backend: Go](https://img.shields.io/badge/Backend-Go-00ADD8?logo=go)](https://golang.org)
[![Frontend: React](https://img.shields.io/badge/Frontend-React-61DAFB?logo=react)](https://reactjs.org)


## About the project
Meerkat CRM (Contact Relationship Management) is a  self-hosted solution to keep track of your important contacts. As your digital rolodex it reminds you of birthdays, helps you to keep in mind dietary habits as well as names of spouses of contacts - and much more.

> [!TIP]
>**[Click here to try the Demo!](https://meerkat-crm-demo.fly.dev/)**
>
>User: test
>
>Password: test_12345

<p align="center">
  <img src="assets/screengrab.gif" alt="Meerkat CRM Demo" />
</p>

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

### Bugs and feature requests
This application is under development. You can [open an issue](https://github.com/fbuchner/meerkat-crm/issues/new/choose) to report a bug or request a new feature.

You can also participate and open up a pull request.

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
    cp .env.example .env
    
    # Install dependencies and run
    go mod tidy
    source .env
    go run main.go
   ```
   The project uses an SQLite database for storage. Database migrations run automatically on startup.


1. **Run the frontend (in a second terminal):**
   ```sh
   cd frontend

   yarn install
   yarn start
   ```

You can find a more comprehensive overview for developers in the [developer README](README-developer.md).

## Alternative software
Notable other personal CRM systems are
* [MonicaHQ](https://www.monicahq.com/) (Open Source, development seems to have stalled; the new version chandler is available at beta.monicahq.com)
* [Dex](https://getdex.com/) (paid offering with social media integration)
* [Clay](https://clay.earth/) (paid offering with focus on automation)

Other software that can be used to build or configure something similar includes
* [Twenty](https://twenty.com/) (Open Source "classic" CRM system)
