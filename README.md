<p align="center">
  <img src="https://github.com/user-attachments/assets/ef5a9313-43bc-492e-ae29-91c643e95bda" alt="Mycorrhizal CRM Logo" width="180" />
</p>

<p align="center">
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Backend-Go-00ADD8?logo=go" alt="Backend: Go"></a>
  <a href="https://reactjs.org"><img src="https://img.shields.io/badge/Frontend-React-61DAFB?logo=react" alt="Frontend: React"></a>
</p>

# Mycorrhizal CRM

**Mycorrhizal** (*my-core-high-zal*) relationships are mutually beneficial networks between fungi ([Mycorrhizae](https://hort.extension.wisc.edu/articles/mycorrhizae/)) and plants. I chose the name because the goal of this project is to help nurture connections with people I care about by keeping track of the small details I might otherwise forget and prompting me to check in periodically.

Mycorrhizal CRM is a self-hosted, privacy-focused contact relationship management solution. It is a fork of [Meerkat CRM](https://github.com/fbuchner/meerkat-crm) by Frederic Buchner.

> **⚠️ Structural Fork & Compatibility Notice:** 
> Mycorrhizal CRM is a **structural fork** of [Meerkat CRM](https://github.com/fbuchner/meerkat-crm). Because this project introduces custom database schemas, modified tables, and expanded data types (such as bidirectional relationships and custom field mappings), **it is not directly database-compatible with upstream Meerkat.** 
> 
> * Direct database migrations from upstream Meerkat are **NOT** supported at this time.
> * Syncing Options: You can sync contacts between Meerkat and Mycorrhizal using CardDAV (though data not supported by standard CardDAV specs will not sync) or by exporting data from one and importing it into the other (though data not defined in the vCard 3.0 RFC is not guaranteed to persist across the export and import).

> **⚠️ Exploration Phase Warning:** This repository is currently in an active exploration and development phase. Breaking changes are expected. If you are looking for a stable, production-ready self-hosted CRM right now, check out upstream [Meerkat](https://github.com/fbuchner/meerkat-crm).

---

## Features & Enhancements On Top Of Upstream Meerkat (In Development)

Mycorrhizal builds heavily upon the solid foundation of Meerkat CRM, adding modern protocol support, deeper structural relationships, and lifestyle tracking utilities:

### Modern Data Formats & Syncing
- **Expanded Protocol Support:** In addition to vCard 3.0 and CardDAV/CalDAV, Mycorrhizal adds full support for **vCard 4.0** and **JSContact**.
- **Flexible Export:** Granular selective field export so you can choose exactly which fields get exported for available formats.

### Relationships, Households & Pets
- **Bidirectional Relationship Graphs:** Relationships are no longer strictly unidirectional. Creating a connection automatically maps it both ways and facilitates relationship-based searching.
- **Household Tracking:** Automatically suggests relationships for contacts sharing the same address. Search by a household to pull lists for event invites, mail, or holiday cards.
- **Pets as Contacts:** Add pets directly to your CRM and search for owners using their pet's name with the relationship support.

### Data Management & Organization
- **Contact Merging:** Seamlessly merge duplicate contact records.
- **Custom Fields & Mappings:** Support for custom fields, including custom mappings to vCard fields to enable extended properties
- **Repurposed General Notes:** Upstream Meerkat's journaling notes have been refactored into general notes that can be cleanly associated with any contact at a later point.
- **File & Document Attachments:** Attach files and documents directly to contact records.
- **One-Time Cross-User Sharing:** Share specific contacts with other users on the same instance, including granular selection of which fields are shared. *(Note: This is a one-time point-in-time copy/share to the target user rather than an ongoing real-time sync).*

### Tracking & Integrations
- **Expanded Life Event Reminders:** Automated reminders for major life events like anniversaries, complementing existing birthday tracking.
- **Gift Tracking:** Modeled after [Monica](https://github.com/monicahq/monica), allowing you to track gift ideas, past gifts given, and received items.
- **Immich Integration:** Link contacts directly to identified persons/faces in an [Immich](https://github.com/immich-app/immich) instance to easily view photos of individuals right from their profile.

---

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
