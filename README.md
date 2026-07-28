# Mycorrhizal CRM - a fork of Meerkat for me to play around in

<p align="center">
  <img src="https://github.com/user-attachments/assets/ef5a9313-43bc-492e-ae29-91c643e95bda" alt="Mycorrhizal CRM Logo" width="180" />
</p>

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Backend: Go](https://img.shields.io/badge/Backend-Go-00ADD8?logo=go)](https://golang.org)
[![Frontend: React](https://img.shields.io/badge/Frontend-React-61DAFB?logo=react)](https://reactjs.org)


## About the project
Mycorrhizal CRM (Contact Relationship Management) is a  self-hosted solution to keep track of your important contacts. As your digital rolodex it reminds you of birthdays, helps you to keep in mind dietary habits as well as names of spouses of contacts - and much more.

At this point, this repo is still in the exploration phase. If you're looking for a well-maintained CRM, check out [Meerkat](https://github.com/fbuchner/meerkat-crm)

## Mycorrhizal?
Mycorrhizal (*my-core-high-zal*) relationships are mutually beneficial relationships between a special kind of fungi, [Mycorrhizae](https://hort.extension.wisc.edu/articles/mycorrhizae/), and plants. Mycorrhiza refers specifically to the fungal side of the relationship where hyphae of the fungus form networks to gather nutrients that are shared with the plant(s).

## Features
- Contact management
    - add and search contacts
    - group contacts by circles (e.g. friends, family, work)
    - store relationships of contacts (e.g. spouses, children)
    - CardDAV server for two-way synchronization with your phone's contact list
    - CardDAV client for two-way synchronization with other servers
    - Import and export all of vCard 3.0, vCard 4.0, and JSContact
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

Mycorrhizal CRM ships as a single all-in-one image that bundles the frontend and
backend into one container, built locally from source (no published registry
image is required). The easiest way to run it is with Docker Compose:

1. **Download the Docker Compose file:**
    ```sh
    curl -O https://raw.githubusercontent.com/DrewBrunning/mycorrhizal-crm/main/docker-compose.yml
    curl -O https://raw.githubusercontent.com/DrewBrunning/mycorrhizal-crm/main/.env.example
    ```

2. **Configure environment:**
    ```sh
    # Copy the environment template
    cp .env.example .env

    # Edit with your settings
    nano .env
    ```

3. **Build and start the container:**
    ```sh
    docker compose up -d --build
    ```

4. **Access the application:**
    Open http://localhost:7300 in your browser.


## Contributing

### Bugs and feature requests
This application is under development. You're free to note bugs or share ideas as issues, but I can't promise I'll implement them at this point.

### Development
To set up this repository for development, follow these steps:

1. **Clone the repository:**
    ```sh
    git clone https://github.com/DrewBrunning/mycorrhizal-crm.git
    cd mycorrhizal-crm
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
