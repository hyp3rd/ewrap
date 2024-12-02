# Base Golang App Repository

![Logo](./assets/logo.svg)

A skeleton repository for starting new Go applications with recommended structure and tooling.

## Getting Started

1. Clone this repository:

   git clone <https://github.com/hyp3rd/go-base-app-skel.git> your-new-project

2. **Remove the existing .git folder** and initialize a new repository and your go module:

   ```bash
   cd your-new-project
   rm -rf .git
   git init
   go mod init github.com/your-org/your-new-project
   ```

3. Install VS Code Extensions Recommended (optional):

   ```json
   {
     "recommendations": [
       "github.vscode-github-actions",
       "golang.go",
       "ms-vscode.makefile-tools",
       "esbenp.prettier-vscode",
       "pbkit.vscode-pbkit",
       "trunk.io",
       "streetsidesoftware.code-spell-checker",
       "ms-azuretools.vscode-docker",
       "eamodio.gitlens"
     ]
   }
   ```

### Development Setup

1. Install [**Golang**](https://go.dev/dl).
2. Install [**Docker**](https://docs.docker.com/get-docker/).
3. Install [**GitVersion**](https://github.com/GitTools/GitVersion).
4. Install [**Make**](https://www.gnu.org/software/make/), follow the procedure for your OS.
5. **Set up the toolchain:**

   ```bash
   make prepare-toolchain
   ```

6. Initialize `pre-commit` (strongly recommended to create a virtual env, using for instance [PyEnv](https://github.com/pyenv/pyenv)) and its hooks:

```bash
   pip install pre-commit
   pre-commit install
   pre-commit install-hooks
```

## Project Structure

```txt
├── cmd/ # Main applications
│   └── app/ # Your application
│       └── main.go # Application entry point
├── internal/ # Private code
│   ├── pkg/ # Internal packages
│   └── app/ # Application specific code
├── pkg/ # Public libraries
├── api/ # API contracts (proto files, OpenAPI specs)
├── configs/ # Configuration files
├── scripts/ # Scripts for development
├── test/ # Additional test files
└── docs/ # Documentation
```

## Best Practices

- Follow the [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- Run `golangci-lint` before committing code
- Ensure the pre-commit hooks pass
- Write tests for new functionality
- Keep packages small and focused
- Use meaningful package names
- Document exported functions and types

## Available Make Commands

- `make test`: Run tests.
- `make update-deps`: Update all dependencies in the project.
- `make prepare-toolchain`: Install all tools required to build the project.
- `make lint`: Run the staticcheck and golangci-lint static analysis tools on all packages in the project.
- `make run`: Build and run the application in Docker.

## License

[MIT License](LICENSE)

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

Refer to [CONTRIBUTING](CONTRIBUTING.md) for more information.

## Author

I'm a surfer, and a software architect with 15 years of experience designing highly available distributed production systems and developing cloud-native apps in public and private clouds. Feel free to connect with me on LinkedIn.

[![LinkedIn](https://img.shields.io/badge/LinkedIn-0077B5?style=for-the-badge&logo=linkedin&logoColor=white)](https://www.linkedin.com/in/francesco-cosentino/)
