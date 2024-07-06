# PumoIDE

PumoIDE is a powerful API development and testing tool, designed to streamline your workflow and enhance productivity.

## Features

- **Collection Management**: Organize your API requests into collections for easy access and management.
- **Environment Support**: Create and manage multiple environments to easily switch between different setups.
- **Request Chaining**: Define dependencies between requests to create complex workflows.
- **Response Validation**: Validate API responses using JSON schema and custom assertions.
- **Authentication Support**: Handles various authentication methods including Basic, Bearer Token, API Key, OAuth2, AWS SigV4, and Digest.
- **Rate Limiting**: Built-in rate limiting to prevent overloading of APIs.
- **Dynamic variable substitution**: Use environment variables and data from previous requests in your API calls.
- **Function expansion**: Support for a wide range of function expansions within your requests.

### Environment variables: Variable substitution and Function expansion

- Variables: `{{VARIABLE_NAME}}`
- Functions: `{{$functionName(arg1,arg2)}}`

### Supported Functions

1. `randomInt`: Generates a random integer. Usage: `{{$randomInt(min,max)}}`
2. `randomFloat`: Generates a random float. Usage: `{{$randomFloat(min,max)}}`
3. `randomString`: Generates a random string. Usage: `{{$randomString(length,charset)}}`
4. `timestamp`: Generates a timestamp. Usage: `{{$timestamp(format)}}`
5. `uuid`: Generates a UUID. Usage: `{{$uuid}}`
6. `base64`: Encodes/decodes base64. Usage: `{{$base64(input)}}` or `{{$base64(input,decode)}}`
7. `md5`: Generates an MD5 hash. Usage: `{{$md5(input)}}`
8. `sha1`: Generates a SHA1 hash. Usage: `{{$sha1(input)}}`
9. `sha256`: Generates a SHA256 hash. Usage: `{{$sha256(input)}}`
10. `lower`: Converts to lowercase. Usage: `{{$lower(input)}}`
11. `upper`: Converts to uppercase. Usage: `{{$upper(input)}}`
12. `capitalize`: Capitalizes the first letter. Usage: `{{$capitalize(input)}}`
13. `env`: Retrieves an environment variable. Usage: `{{$env(VAR_NAME)}}` or `{{$env(VAR_NAME,default)}}`

## Getting Started

1. Clone the repository
2. Navigate to the `backend` directory
3. Run `go mod tidy` to ensure all dependencies are installed
4. Start the server with `go run cmd/pumoide/main.go`

The server will start and write its port to an environment variable.

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