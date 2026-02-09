# TaskD Changelog

## [Unreleased]

### Fixed
- Process detachment issue on Windows: Tasks and daemon now properly detach from parent terminal
  - Added `DETACHED_PROCESS` flag to prevent processes from being killed when terminal closes
  - Both daemon and user tasks now survive terminal closure
  - Processes run in their own process group independent of the parent session

### Added
- Version support following Go conventions (v0.1.0)
  - Version constant in main.go (defaults to "develop")
  - Build-time version injection via ldflags from VERSION file
  - `--version` and `-v` flags to display version
  - Cobra version command integration

## [0.1.0] - 2026-01-27

### Added
- Initial project structure and architecture
- Basic CLI framework using Cobra
- Task configuration management with TOML format
- Task manager with CRUD operations
- Simple task lifecycle management
- Cross-platform support foundation
- Configuration file management with Viper
- Basic logging infrastructure
- Build scripts and Makefile
- Comprehensive documentation

### Features
- Add tasks with executable path and arguments
- List all tasks with status information
- Start and stop task management
- Task status monitoring
- Working directory configuration
- Environment variable management (inherit/override)
- Standard input/output redirection support
- Configuration persistence in TOML format

### Technical
- Go 1.21+ support
- Modular architecture with clear separation of concerns
- Thread-safe task management
- Graceful error handling
- Cross-platform file path handling

### Documentation
- Complete project structure documentation
- Development plan with 4-phase roadmap
- Usage examples and best practices
- Configuration file examples
- Getting started guide

### Build System
- Makefile with common development tasks
- Windows build script
- Cross-compilation support
- Dependency management with Go modules

### Known Limitations
- No automatic restart policies yet
- No log rotation functionality
- No daemon mode
- Limited process monitoring
- No web interface

### Next Steps
- Implement restart policies
- Add log rotation with Lumberjack
- Enhance process monitoring
- Add daemon mode support
- Implement web dashboard (optional)

---

## Development Notes

### Proxy Configuration
- Successfully configured HTTP proxy (localhost:57890) for dependency downloads
- All external dependencies downloaded successfully
- Build system working correctly

### Code Quality
- All Chinese comments and messages converted to English
- Consistent error handling patterns
- Thread-safe operations with proper mutex usage
- Clean separation between CLI, business logic, and configuration layers

### Testing
- Basic functionality verified
- Task creation and listing working
- Configuration file generation confirmed
- Both simple and full versions building successfully