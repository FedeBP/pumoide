# PumoIDE

PumoIDE is a powerful API development and testing tool, designed to streamline your workflow and enhance productivity.

## Features

- **Collection Management**: Organize your API requests into collections for easy access and management.
- **Environment Support**: Create and manage multiple environments to easily switch between different setups.
- **Variable Substitution**: Use environment variables and data from previous requests in your API calls.
- **Request Chaining**: Define dependencies between requests to create complex workflows.
- **Response Validation**: Validate API responses using JSON schema and custom assertions.
- **Authentication Support**: Handles various authentication methods including Basic, Bearer Token, API Key, OAuth2, AWS SigV4, and Digest.
- **Rate Limiting**: Built-in rate limiting to prevent overloading of APIs.

## Getting Started

1. Clone the repository
2. Navigate to the `backend` directory
3. Run `go mod tidy` to ensure all dependencies are installed
4. Start the server with `go run cmd/pumoide/main.go`

The server will start and write its port to a `port.txt` file in the project root.

## Project Structure

- `cmd/pumoide`: Contains the main entry point of the application.
- `internal`: Houses the core application code.
   - `api`: API handlers for collections, environments, methods, and requests.
   - `app`: Application setup and configuration.
   - `models`: Data models for collections and environments.
   - `middleware`: Middleware functions like rate limiting.
   - `utils`: Utility functions for file operations and variable substitution.
   - `validators`: Validation logic for requests and responses.
- `pkg`: Reusable packages that could potentially be used by external applications.
   - `constants`: Constant values used throughout the application.
   - `errors`: Custom error handling.
   - `logger`: Logging functionality.
- `test`: Contains all test files.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

[MIT License](LICENSE)