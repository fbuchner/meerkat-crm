# Meerkat CRM - a simple CRM for the personal life

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
    git clone https://github.com/fbuchner/perema.git
    cd perema
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
   The project uses an SQLite database for storage, which will be automatically created if it doesn't exist.

1. **Run the frontend (in a second terminal):**
    ```sh
    cd frontend
    yarn install
    yarn serve
    ```

You can also use the Debug button in [Visual Studio Code](https://code.visualstudio.com/) as configured in the launch.json file.
