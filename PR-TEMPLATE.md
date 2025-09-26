# Pull Request: Phase 1 - Basic Process Management MVP

## 🎯 Overview
Implementation of Phase 1 MVP for Wemixvisor - a process manager for automated binary upgrades of WBFT-based blockchain nodes.

## 📋 Description
This PR introduces the foundational features for Wemixvisor, providing basic process management capabilities, file-based upgrade detection, and symbolic link management for seamless binary version switching.

## ✨ Changes

### Core Features
- **Process Management**
  - Start/stop blockchain node processes
  - Signal handling (SIGTERM, SIGINT, SIGQUIT)
  - Process health monitoring
  - Automatic restart capabilities

- **Upgrade Detection**
  - File-based monitoring via `upgrade-info.json`
  - Configurable polling intervals (default: 300ms)
  - Modification time tracking
  - Upgrade validation

- **Binary Management**
  - Symbolic link-based version switching
  - Genesis binary initialization
  - Upgrade directory structure
  - Atomic binary switching

### CLI Commands
- `init` - Initialize directory structure and genesis binary
- `run` - Start managed process with upgrade monitoring
- `version` - Display version information

### Configuration
- Environment variable support (DAEMON_*)
- Command-line flag overrides
- Configuration validation
- Default sensible values

## 🏗️ Technical Details

### Architecture
```
Process Manager
├── Process Launcher
├── File Watcher
├── Signal Handler
└── Child Process (wemixd)
```

### Directory Structure
```
$DAEMON_HOME/
├── wemixvisor/
│   ├── current/    → symbolic link
│   ├── genesis/    → initial binary
│   └── upgrades/   → upgrade binaries
├── data/
│   └── upgrade-info.json
└── backups/        → (Phase 2)
```

## 📊 Testing
- [x] Manual testing of process lifecycle
- [x] Upgrade detection validation
- [x] Signal handling verification
- [x] Symbolic link operations
- [ ] Unit tests (planned for Phase 2)
- [ ] Integration tests (planned for Phase 2)

## 📝 Documentation
- Added comprehensive Phase 1 documentation in `docs/phase1-mvp.md`
- Updated README with installation and usage instructions
- Created CLAUDE.md with development guidelines
- Maintained CHANGES.md for version history

## 🔄 Related Issues
- Implements Phase 1 of the Wemixvisor roadmap
- Inspired by Cosmos SDK's Cosmovisor

## ✅ Checklist
- [x] Code follows project style guidelines
- [x] Changes are documented
- [x] Manual testing completed
- [x] No breaking changes
- [x] Ready for review

## 🚀 Next Steps (Phase 2)
- Data backup functionality
- Pre-upgrade hooks
- Graceful shutdown enhancements
- Custom pre-upgrade scripts
- Comprehensive testing suite

## 📌 Notes
This is the MVP implementation focusing on core functionality. Advanced features like WBFT consensus integration and automatic binary downloads are planned for Phase 3.

---

**Branch**: `feature/process-management-mvp` → `dev`
**Version**: v0.1.0