# Wemixvisor Project Guidelines

## Project Overview
Wemixvisor is a process manager for WBFT-based blockchain node upgrades, inspired by Cosmos SDK's Cosmovisor.

## Development Guidelines

### Language and Framework
- **Language**: Go (Golang)
- **Target**: WBFT consensus blockchain nodes
- **Architecture**: Modular design with clear separation of concerns

### Version Management
- Each phase completion gets a version tag (v0.1.0, v0.2.0, etc.)
- Maintain CHANGES.md with detailed history
- Document each phase in docs/ directory

### Git Commit Rules
- **Format**: `type(scope): description`
- **Types**:
  - `feat`: New feature
  - `fix`: Bug fix
  - `docs`: Documentation changes
  - `refactor`: Code refactoring
  - `test`: Test additions or changes
  - `chore`: Build process or auxiliary tool changes
- **Language**: English only
- **No co-author tags**: Remove any auto-generated co-author information
- **Commit at each phase completion**

### Code Quality Standards
- Follow Go best practices and idioms
- Maintain test coverage above 80%
- Document all public APIs
- Use structured logging
- Handle errors explicitly

### Project Phases

#### Phase 1: MVP (v0.1.0)
- Basic process management
- File-based upgrade detection
- Symbolic link management

#### Phase 2: Core Features (v0.2.0)
- Backup functionality
- Pre-upgrade hooks
- Graceful shutdown

#### Phase 3: Advanced Features (v0.3.0)
- WBFT consensus integration
- Batch upgrades
- Automatic binary download

### Directory Structure
```
wemixvisor/
├── cmd/              # Command-line interface
├── internal/         # Private application code
│   ├── config/      # Configuration management
│   ├── process/     # Process management
│   ├── upgrade/     # Upgrade handling
│   └── wbft/        # WBFT integration
├── pkg/             # Public libraries
├── docs/            # Documentation
├── test/            # Integration tests
└── examples/        # Example configurations
```

### Testing Requirements
- Unit tests for all packages
- Integration tests for critical workflows
- Test with actual WBFT node binaries

### Security Considerations
- Validate all binary checksums
- Secure handling of private keys
- Proper file permissions
- No automatic commits without validation