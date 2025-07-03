# RainBridge Project Overview

RainBridge is a command-line utility designed to streamline the process of importing bookmarks from Raindrop.io into Karakeep. Its primary goal is to provide a seamless and efficient migration path for users, ensuring that valuable links, associated metadata, and organizational structures (collections/lists) are transferred accurately.

## Key Project Aspects & Decisions:

### Core Functionality:
- Imports bookmarks, including their URLs, titles, descriptions, and tags.
- Preserves the organizational structure by migrating Raindrop.io collections to Karakeep lists.
- Handles API rate limiting with exponential backoff to ensure reliable batch operations.

### Technical Stack:
- **Language:** Go (chosen for its excellent cross-platform distribution capabilities, compiling to a single static binary, which simplifies packaging for various operating systems like macOS, Linux, and Windows).
- **API Clients:** Custom-built Go clients for both Raindrop.io and Karakeep APIs.

### Authentication & Configuration:
- API tokens for both Raindrop.io and Karakeep are securely managed via environment variables.
- Support for `.env` files is included for local development convenience.

### Data Fetching Strategy:
- Utilizes direct API calls to Raindrop.io to fetch all user data (bookmarks and collections) through paginated requests.
- The Raindrop.io "backup to zip file" feature is explicitly *not* used, as direct API access provides a more real-time and seamless user experience.

### Distribution:
- The tool will be released as open-source software.
- Planned distribution channels include popular package managers: Homebrew (macOS/Linux), Arch User Repository (AUR), Fedora (COPR/official), and Debian/Ubuntu (PPA/official).

### Testing Strategy:
- **Unit Tests:** Employ mock HTTP servers (`httptest`) to test API clients and core import logic in isolation, ensuring no real network calls are made.
- **Integration Tests:** Dedicated tests for each API client (Raindrop.io and Karakeep) that interact with their respective live services, while mocking the other service to isolate the test scope.

## Relevant API Documentation:

### Karakeep API:
- [Karakeep API Documentation](https://docs.karakeep.app/API/karakeep-api)
- Specific endpoints used/relevant:
    - Get all bookmarks: `https://docs.karakeep.app/API/get-all-bookmarks`
    - Create a new bookmark: `https://docs.karakeep.app/API/create-a-new-bookmark`
    - Attach tags to a bookmark: `https://docs.karakeep.app/API/attach-tags-to-a-bookmark`
    - Get all lists: `https://docs.karakeep.app/API/get-all-lists`
    - Create a new list: `https://docs.karakeep.app/API/create-a-new-list`
    - Add a bookmark to a list: `https://docs.karakeep.app/API/add-a-bookmark-to-a-list`

### Raindrop.io API:
- [Raindrop.io Developer Documentation](https://developer.raindrop.io/)
- Specific endpoints used/relevant:
    - Get raindrops (bookmarks) by collection: `/rest/v1/raindrops/{collectionId}` (with pagination)
    - Get collections: `/rest/v1/collections`

## Project Repository:
- This project's GitHub repository: `https://github.com/ashebanow/rainbridge`

## Justfile Cheatsheet

This section provides a quick reference for `justfile` syntax and common patterns used in this project.

### Variables:
- Define variables using `VAR_NAME := "value"`.
- Access variables within recipes using `{{VAR_NAME}}`.

### Recipes (Tasks):
- Define a recipe with its name followed by a colon, e.g., `my-task:`.
- Commands within a recipe are indented.
- Use `@` before a command to prevent `just` from echoing the command itself.

### Passing Arguments to Recipes:
- Define arguments in the recipe signature, e.g., `my-task ARG:`.
- Access arguments within the recipe using `{{ARG}}`.

### Multi-line Shell Scripts (using Shebang):
- For complex shell logic or when `just`'s parsing is problematic, start the recipe with a shebang (e.g., `#!/bin/bash`).
- This tells `just` to pass the entire recipe body to the specified shell.
- Inside a shebang-driven recipe, arguments passed to the `just` command (e.g., `just my-task ARG=value`) are available as shell environment variables (e.g., `$ARG`).

**Example:**
```justfile
# Variables
PROJECT_NAME := "my-app"

# A simple recipe
hello:
    @echo "Hello from {{PROJECT_NAME}}"

# A recipe with an argument and shebang
greet NAME:
    #!/bin/bash
    echo "Hello, ${NAME}! This is ${PROJECT_NAME}."

# How to call:
# just hello
# just greet NAME=Alice
```
