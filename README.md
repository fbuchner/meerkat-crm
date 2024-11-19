# Perema - a simple CRM for the personal life

## About the project
The **Pe**rsonal **Re**lationship **Ma**nager is a simple self-hosted solution to keep track of all your contacts. As your digital Rolodex it reminds of of birthdays, helps you to remember dietary habits as well as names of spouses of contacts - and much more.

## Features
**Implemented**
- Contact management
    - add and search contacts
    - special fields (e.g. birthdays w/o known year)
    - group contacts by circles (e.g. friends, family, work)
- Notes and activities
    - social network style timeline for contacts
    - notes assigned to individual contacts
    - general notes (for e.g. journaling)
    - activities with one or multiple contacts

**Yet to come**
- Set custom reminders for contacts and get e-mail notifications
- Keep in touch for contacts at regular intervals
- LinkedIn sync
- Google contacts sync
- I18N (English, German)

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
Ensure you have [Go](https://golang.org/doc/install) installed. Then, install the required Go packages, load the environment variables and run the backend (consider creating a copy of the environment.env named my_environment.env to store your personal configuration. It will be ignored by git). The project uses an sqlite database for storage, it will be automatically created if it does not exist.:
    ```sh
    cd backend
    go mod tidy
    source environment.env
    go run main.go
    ```

1. **Run the frontend (in a second terminal):**
    ```sh
    cd frontend
    yarn install
    yarn serve
    ```

You can also use the Debug button in [Visual Studio Code](https://code.visualstudio.com/) as configured in the launch.json file.


