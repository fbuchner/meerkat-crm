# Perema - a simple CRM for the personal life

## About the project
The **Pe**rsonal **Re**lationship **Ma**nager is a simple self-hosted solution to keep track of all your contacts. As your digital Rolodex it reminds of of birthdays, helps you to remember dietary habits as well as names of spouses of contacts - and much more.

## Features


## Installation


## Usage


## Contributing

### Development
To set up this repository for development, follow these steps:

1. **Clone the repository:**
    ```sh
    git clone https://github.com/fbuchner/perema.git
    cd perema
    ```

1. **Install dependencies:**
    Ensure you have [Go](https://golang.org/doc/install) installed. Then, install the required Go packages:
    ```sh
    go mod tidy
    ```

1. **Load the environment variables:**
    The project uses an sqlite database for storage, it will be automatically created if it does not exist.
    ```sh
    source environment.env
    ```

1. **Run the backend:**
    ```sh
    go run main.go
    ```

1. **Run the frontend (in a second terminal):**
    ```sh
    cd frontend
    yarn serve
    ```



