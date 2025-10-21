# Phase 8 Development Priority Guide

**Target**: v0.8.0 - Core Upgrade Automation
**Branch**: `feature/phase8-upgrade-automation`
**Total Duration**: 17 days (2.5-3 weeks)

---

## Executive Summary

### Critical Path (Sequential - Must Follow Order)
```
Week 1: Foundation
├── Day 1-3: Task 8.1 (HeightMonitor) ⭐⭐⭐ P0
├── Day 4-8: Task 8.2 (UpgradeOrchestrator) ⭐⭐⭐ P0
└── Day 9-10: Task 8.3 (Main Integration) ⭐⭐⭐ P0

Week 2: External Interfaces (Can be parallel after Day 8)
├── Day 11-12: Task 8.4 (CLI Commands) ⭐⭐ P1
└── Day 11-12: Task 8.5 (API Endpoints) ⭐⭐ P1 (parallel with 8.4)

Week 3: Validation
└── Day 13-15: Task 8.6 (Integration Testing) ⭐⭐⭐ P0

Buffer: Day 16-17 (Bug fixes, documentation, polish)
```

**Why This Order?**
- 8.1 → 8.2 → 8.3: Core functionality dependency chain
- 8.4 ∥ 8.5: Independent interfaces to orchestrator (can parallelize)
- 8.6: Validates everything together (must be last)

---

## Detailed Priority Breakdown

### Priority Level Definitions

**P0 - Critical (Blocking)**
- Must be completed in order
- Blocks subsequent tasks
- Core functionality
- Cannot ship without this

**P1 - High (Important)**
- Important for user experience
- Can be done in parallel
- Should complete before release
- Can be deferred if time-critical

**P2 - Medium (Nice to Have)**
- Improves usability
- Can be pushed to v0.8.1
- Optional enhancements

---

## Week 1: Foundation Layer

### Day 1-3: Task 8.1 - HeightMonitor Implementation
**Priority**: ⭐⭐⭐ P0 - CRITICAL BLOCKER
**Dependencies**: None (foundation component)
**Blocks**: Task 8.2, 8.3, 8.6

**Why First?**
- Foundation for entire upgrade automation
- No dependencies - can start immediately
- Required by UpgradeOrchestrator
- Smallest scope - quick win for momentum

**Daily Breakdown**:

#### Day 1: Design + TDD Red Phase
**Goal**: All failing tests written
```
Morning (4h):
├── 8.1.1: Design & Interface Definition
│   ├── Create directory structure
│   ├── Define HeightProvider interface
│   └── Design HeightMonitor struct
└── 8.1.2.1-8.1.2.6: Write ALL failing tests
    ├── Mock HeightProvider
    ├── Constructor tests
    ├── Lifecycle tests
    ├── Height monitoring tests
    ├── Subscriber pattern tests
    └── Concurrency tests

Afternoon (4h):
├── Run tests (confirm all fail)
├── Review test coverage plan
└── Document expected behaviors
```

#### Day 2: TDD Green Phase
**Goal**: All tests passing with minimal code
```
Morning (4h):
├── 8.1.2.7-8.1.2.9: Core structure
│   ├── Implement NewHeightMonitor
│   ├── Implement Start method
│   └── Implement Stop method
└── Run tests (some should pass)

Afternoon (4h):
├── 8.1.2.10-8.1.2.13: Core logic
│   ├── Implement monitorLoop
│   ├── Implement Subscribe
│   ├── Implement notifySubscribers
│   └── Implement GetCurrentHeight
└── Run ALL tests (should all pass ✅)
```

#### Day 3: TDD Refactor + Integration
**Goal**: 90%+ coverage, production ready
```
Morning (4h):
├── 8.1.2.14-8.1.2.17: Refactor phase
│   ├── SOLID principles review
│   ├── Code quality improvements
│   ├── Edge case tests
│   └── Coverage verification (90%+)

Afternoon (4h):
├── 8.1.3: Integration Testing
│   ├── Test with real RPC client
│   ├── Benchmark tests
├── 8.1.4: Documentation
│   ├── Package docs
│   ├── Godoc comments
└── 8.1.5: Commit and Review
    ├── Final checks (fmt, lint, vet)
    └── Git commit

Evening:
└── Push to remote, mark Task 8.1 DONE ✅
```

**Success Criteria**:
- [ ] All tests pass
- [ ] Coverage ≥ 90%
- [ ] No race conditions
- [ ] Documentation complete
- [ ] Committed and pushed

---

### Day 4-8: Task 8.2 - UpgradeOrchestrator Implementation
**Priority**: ⭐⭐⭐ P0 - CRITICAL BLOCKER
**Dependencies**: Task 8.1 (HeightMonitor)
**Blocks**: Task 8.3, 8.4, 8.5, 8.6

**Why Second?**
- Core orchestration logic
- Most complex component (5 days)
- Integrates with HeightMonitor
- Required for CLI and API

**Daily Breakdown**:

#### Day 4: Design + Mock Setup + TDD Red Phase Start
```
Morning (4h):
├── 8.2.1: Design & Interface Definition
│   ├── Create directory structure
│   ├── Define component interfaces
│   │   ├── NodeManager interface
│   │   ├── ConfigManager interface
│   │   └── UpgradeWatcher interface
│   ├── Design UpgradeOrchestrator struct
│   └── Design UpgradeStatus struct

Afternoon (4h):
├── 8.2.2.1: Create mock components
│   ├── MockNodeManager (with call tracking)
│   ├── MockConfigManager
│   └── MockUpgradeWatcher
└── 8.2.2.2-8.2.2.5: Write core tests
    ├── Constructor test
    ├── Lifecycle test
    ├── Schedule upgrade test
    └── Height trigger test
```

#### Day 5: TDD Red Phase Complete + Green Phase Start
```
Morning (4h):
├── 8.2.2.6-8.2.2.10: Finish failing tests
│   ├── Upgrade execution test
│   ├── Rollback test ⭐ (critical!)
│   ├── Concurrent upgrade test
│   ├── Error handling test
│   └── Run all tests (confirm fail)

Afternoon (4h):
├── 8.2.2.11-8.2.2.13: Start implementation
│   ├── Implement constructor
│   ├── Implement Start method
│   └── Implement Stop method
└── Run tests (basic tests pass)
```

#### Day 6: TDD Green Phase - Core Logic
```
Morning (4h):
├── 8.2.2.14-8.2.2.17: Watchers
│   ├── Implement watchUpgradeConfigs
│   ├── Implement scheduleUpgrade
│   ├── Implement monitorHeights
│   └── Implement shouldTriggerUpgrade
└── Run tests (more tests pass)

Afternoon (4h):
├── 8.2.2.18: ⭐ CRITICAL - executeUpgrade
│   ├── Set upgrading flag
│   ├── Stop node
│   ├── Switch binary
│   ├── Start node
│   ├── Verify running
│   └── Handle errors → rollback
└── Run tests (most should pass)
```

#### Day 7: TDD Green Phase - Complete + Refactor
```
Morning (4h):
├── 8.2.2.19-8.2.2.20: Finish implementation
│   ├── ⭐ Implement rollback (critical!)
│   │   ├── Log rollback
│   │   ├── Restore genesis binary
│   │   ├── Restart node
│   │   └── Error handling
│   ├── Implement GetPendingUpgrade
│   └── Implement GetStatus
└── 8.2.2.21: Run ALL tests
    └── Confirm all pass ✅

Afternoon (4h):
├── 8.2.2.22-8.2.2.25: Refactor phase
│   ├── SOLID principles review
│   ├── Error handling improvements
│   ├── Edge case tests
│   └── Coverage verification (90%+)
```

#### Day 8: Integration Testing + Documentation
```
Morning (4h):
├── 8.2.3: Integration Testing
│   ├── Test with real NodeManager
│   ├── Simulate complete upgrade
│   └── Test rollback scenarios

Afternoon (4h):
├── 8.2.4: Documentation
│   ├── Package documentation
│   ├── Usage examples
└── 8.2.5: Commit and Review
    ├── Final checks (fmt, lint, vet, race)
    ├── Coverage verification (≥90%)
    └── Git commit and push

Evening:
└── Mark Task 8.2 DONE ✅
```

**Success Criteria**:
- [ ] All tests pass
- [ ] Rollback mechanism tested thoroughly
- [ ] Coverage ≥ 90%
- [ ] Integration tests pass
- [ ] Documentation complete

---

### Day 9-10: Task 8.3 - Main Integration
**Priority**: ⭐⭐⭐ P0 - CRITICAL BLOCKER
**Dependencies**: Task 8.1, 8.2
**Blocks**: Task 8.6 (E2E tests)

**Why Third?**
- Wires everything together
- Creates working end-to-end flow
- Enables manual testing
- Required before external interfaces

**Daily Breakdown**:

#### Day 9: Integration Implementation
```
Morning (4h):
├── 8.3.1: Update Main Entry Point
│   ├── Import new packages
│   ├── Initialize HeightProvider
│   │   ├── Reuse governance RPC if available
│   │   └── Create new client if needed
│   ├── Create HeightMonitor
│   ├── Create UpgradeOrchestrator
│   ├── Start components (correct order)
│   └── Handle shutdown gracefully

Afternoon (4h):
├── 8.3.2: Configuration Updates
│   ├── Add HeightPollInterval to config
│   ├── Add UpgradeEnabled to config
│   ├── Update DefaultConfig
│   └── Add environment variable parsing
└── Build and initial test
    └── make build
```

#### Day 10: Testing + Documentation
```
Morning (4h):
├── 8.3.3: Testing
│   ├── Manual testing
│   │   ├── Build: make build
│   │   ├── Initialize: wemixvisor init
│   │   ├── Start with upgrade enabled
│   │   ├── Check height monitoring logs
│   │   ├── Create test upgrade-info.json
│   │   └── Verify upgrade detection
│   └── Integration test
│       └── Test full startup flow

Afternoon (4h):
├── Fix any issues found
├── Documentation updates
│   ├── Update README with new config
│   ├── Update env var documentation
└── 8.3.4: Commit
    ├── Final checks
    └── Git commit and push

Evening:
└── Mark Task 8.3 DONE ✅
```

**Success Criteria**:
- [ ] Builds successfully
- [ ] Components start in correct order
- [ ] Height monitoring works
- [ ] Upgrade detection works
- [ ] Manual test successful

**🎯 CHECKPOINT**: End of Week 1
- Core functionality complete
- Can demonstrate working upgrade automation
- Ready for external interfaces

---

## Week 2: External Interfaces (Parallel Development)

### Day 11-12: Task 8.4 + 8.5 (PARALLEL)
**Strategy**: Work on CLI and API simultaneously

**Why Parallel?**
- Both depend on Task 8.2 (orchestrator) - already done ✅
- Independent of each other
- Can be split between developers or time-boxed
- Saves 2 days vs sequential

---

### Day 11-12 (Track A): Task 8.4 - CLI Commands
**Priority**: ⭐⭐ P1 - HIGH
**Dependencies**: Task 8.2
**Parallel With**: Task 8.5

**Time Allocation**: 2 days, 50% focus each day

#### Day 11 (Morning + Evening)
```
Morning (4h):
├── 8.4.1: Create Package
├── 8.4.2: Schedule Command
│   ├── Define command structure
│   ├── Implement schedule logic
│   │   ├── Parse arguments
│   │   ├── Validate height
│   │   ├── Support flags (binaries, checksum)
│   │   ├── Create UpgradeInfo
│   │   └── Write to file
│   └── Write tests

Evening (2h):
├── 8.4.3: List Command
│   ├── Implement list logic
│   └── Write tests
└── 8.4.4: Cancel Command
    ├── Implement cancel logic
    └── Write tests
```

#### Day 12 (Morning + Evening)
```
Morning (4h):
├── 8.4.5: Status Command
│   ├── Implement status logic
│   └── Write tests
├── 8.4.6: Integration
│   └── Update root command
└── 8.4.7: Testing
    ├── Manual CLI testing
    │   ├── wemixvisor upgrade schedule
    │   ├── wemixvisor upgrade list
    │   ├── wemixvisor upgrade status
    │   └── wemixvisor upgrade cancel
    └── Integration tests

Evening (2h):
├── Fix any issues
└── 8.4.8: Commit
    ├── Run checks
    └── Git commit and push
```

---

### Day 11-12 (Track B): Task 8.5 - API Endpoints
**Priority**: ⭐⭐ P1 - HIGH
**Dependencies**: Task 8.2
**Parallel With**: Task 8.4

**Time Allocation**: 2 days, 50% focus each day

#### Day 11 (Afternoon)
```
Afternoon (4h):
├── 8.5.1: Create Handlers
├── 8.5.2: POST /api/v1/upgrades
│   ├── Define request/response structures
│   ├── Implement handler
│   │   ├── Bind JSON
│   │   ├── Validate input
│   │   ├── Call orchestrator
│   │   └── Return response
│   └── Write tests
└── 8.5.3: GET /api/v1/upgrades
    ├── Implement handler
    └── Write tests
```

#### Day 12 (Afternoon)
```
Afternoon (4h):
├── 8.5.4: DELETE /api/v1/upgrades/:name
│   ├── Implement handler
│   └── Write tests
├── 8.5.5: GET /api/v1/upgrades/status
│   ├── Implement handler
│   └── Write tests
├── 8.5.6: Register Routes
├── 8.5.7: Update Server Struct
└── 8.5.8: Testing
    ├── Manual API testing
    │   ├── curl POST schedule
    │   ├── curl GET list
    │   ├── curl GET status
    │   └── curl DELETE cancel
    └── Integration tests
```

#### End of Day 12
```
Evening (1h):
├── 8.5.9: Commit
│   ├── Run checks
│   └── Git commit and push
└── Mark Tasks 8.4 and 8.5 DONE ✅
```

**🎯 CHECKPOINT**: End of Day 12
- CLI commands working
- API endpoints working
- All external interfaces complete
- Ready for comprehensive testing

---

## Week 3: Validation & Polish

### Day 13-15: Task 8.6 - Integration Testing
**Priority**: ⭐⭐⭐ P0 - CRITICAL
**Dependencies**: ALL previous tasks (8.1-8.5)
**Blocks**: Release

**Why Last?**
- Tests entire system integration
- Catches integration bugs
- Validates all workflows
- Final quality gate

**Daily Breakdown**:

#### Day 13: E2E Test Infrastructure + Core Workflows
```
Morning (4h):
├── 8.6.1: Setup E2E Environment
│   ├── Create test directory
│   ├── Create TestEnvironment helper
│   │   ├── setupTestEnvironment
│   │   ├── Cleanup
│   │   ├── Start
│   │   ├── ScheduleUpgrade
│   │   ├── AdvanceBlockHeight
│   │   └── GetNodeStatus
│   └── Create mock node and RPC server

Afternoon (4h):
├── 8.6.2: Complete Workflow Tests
│   ├── Test normal upgrade flow ⭐
│   │   ├── Start wemixvisor
│   │   ├── Schedule upgrade
│   │   ├── Advance to upgrade height
│   │   ├── Verify upgrade executed
│   │   └── Verify new binary running
│   ├── Test rollback on failure ⭐
│   │   ├── Setup invalid binary
│   │   ├── Trigger upgrade
│   │   └── Verify rollback to genesis
│   └── Test multiple upgrades
│       ├── First upgrade
│       └── Second upgrade
```

#### Day 14: Interface Testing + Error Scenarios
```
Morning (4h):
├── 8.6.3: CLI Integration
│   ├── Test schedule and execute
│   └── Test cancel
├── 8.6.4: API Integration
│   ├── Test schedule and monitor
│   └── Test WebSocket notifications

Afternoon (4h):
├── 8.6.5: Error Scenarios
│   ├── Network disconnection
│   ├── Concurrent upgrade attempts
│   └── Node crash during upgrade
```

#### Day 15: Performance + Load Testing + Documentation
```
Morning (4h):
├── 8.6.6: Performance Testing
│   ├── Height monitoring overhead (<5%)
│   └── Upgrade execution time (<30s)
├── 8.6.7: Load Testing
│   ├── Rapid height changes (1/sec)
│   └── Long-running stability (1 hour)

Afternoon (4h):
├── 8.6.8: Documentation
│   ├── E2E testing guide
│   ├── Update README
│   │   ├── New CLI commands section
│   │   ├── New API endpoints section
│   │   ├── Configuration options
│   │   └── Upgrade workflow examples
│   └── Update examples
└── 8.6.9: Commit
    ├── Run all E2E tests
    └── Git commit and push

Evening:
└── Mark Task 8.6 DONE ✅
```

**Success Criteria**:
- [ ] All E2E tests pass
- [ ] Performance targets met
- [ ] Load tests pass
- [ ] Documentation complete

**🎯 CHECKPOINT**: End of Day 15
- All tests pass
- All documentation updated
- Ready for pre-release review

---

## Day 16-17: Buffer & Pre-Release

### Day 16: Pre-Release Checklist
```
Morning (4h):
├── Run all checks
│   ├── make test (all unit + integration)
│   ├── go test -race ./... (no races)
│   ├── make coverage (≥90%)
│   ├── make lint (no warnings)
│   ├── make fmt (formatted)
│   └── make vet (no issues)
├── SOLID Principles Review
│   ├── Single Responsibility ✓
│   ├── Open/Closed ✓
│   ├── Liskov Substitution ✓
│   ├── Interface Segregation ✓
│   └── Dependency Inversion ✓

Afternoon (4h):
├── Documentation review
│   ├── All packages documented
│   ├── README updated
│   ├── API documentation
│   ├── CLI help text
│   └── Examples provided
├── Performance verification
│   ├── Height monitoring overhead <5% ✓
│   ├── Memory usage <100MB ✓
│   ├── Upgrade execution <30s ✓
│   └── No memory/goroutine leaks ✓
```

### Day 17: Manual Testing + Release Prep
```
Morning (4h):
├── Manual testing on testnet
│   ├── Complete upgrade flow
│   ├── Rollback scenario
│   ├── CLI commands
│   ├── API endpoints
│   └── Error handling

Afternoon (4h):
├── Final polish
│   ├── Fix any remaining issues
│   ├── Update version to v0.8.0
│   ├── Update CHANGES.md
│   └── Final README review
├── Release preparation
│   ├── Create release branch
│   └── Final commit

Evening:
└── Ready for merge to dev and main 🎉
```

---

## Priority Decision Matrix

### When to Skip/Defer Tasks

**Cannot Skip (Blockers)**:
- Task 8.1 ❌ - Foundation, blocks everything
- Task 8.2 ❌ - Core logic, blocks everything
- Task 8.3 ❌ - Integration, blocks testing
- Task 8.6 ❌ - Validation, blocks release

**Can Defer to v0.8.1** (if time-critical):
- Task 8.4 ✅ - CLI is nice to have, orchestrator works via file
- Task 8.5 ✅ - API is nice to have, CLI is sufficient
- Both 8.4 and 8.5 together ✅ - Can use file-based config

**Minimum Viable Release**:
If extremely time-critical, can ship with:
- ✅ Task 8.1 (HeightMonitor)
- ✅ Task 8.2 (UpgradeOrchestrator)
- ✅ Task 8.3 (Main Integration)
- ✅ Basic tests from 8.6
- ❌ Skip 8.4, 8.5 (defer to v0.8.1)

**Full Recommended Release** (strongly recommended):
- All tasks 8.1-8.6 ✅
- Professional, complete solution
- Good user experience

---

## Risk Management by Priority

### High-Risk Tasks (Need Extra Attention)

**Task 8.2 (Day 4-8)** ⚠️ HIGHEST RISK
- Most complex component
- Critical rollback logic
- Many edge cases
- **Mitigation**:
  - Spend extra time on test design
  - Test rollback thoroughly
  - Daily code review
  - Don't rush

**Task 8.6 (Day 13-15)** ⚠️ MEDIUM RISK
- Requires all components working
- E2E tests can be flaky
- **Mitigation**:
  - Good test environment setup
  - Proper cleanup between tests
  - Adequate timeouts
  - Retry logic for flaky tests

### Low-Risk Tasks (Quick Wins)

**Task 8.1 (Day 1-3)** ✅ LOW RISK
- Well-defined scope
- Simple responsibility
- Good for momentum

**Task 8.4, 8.5 (Day 11-12)** ✅ LOW RISK
- Straightforward implementations
- Good separation from core logic

---

## Success Metrics by Priority

### P0 Metrics (Must Achieve)
- [ ] All P0 tasks complete (8.1, 8.2, 8.3, 8.6)
- [ ] Test coverage ≥ 90% for new code
- [ ] Zero critical bugs
- [ ] All E2E tests pass
- [ ] Manual testing successful

### P1 Metrics (Should Achieve)
- [ ] All P1 tasks complete (8.4, 8.5)
- [ ] Performance targets met
- [ ] Documentation complete
- [ ] SOLID principles verified

### P2 Metrics (Nice to Have)
- [ ] Additional examples
- [ ] Video tutorial
- [ ] Web UI mockup

---

## Recommended Daily Routine

### Every Morning (30 min)
1. Review TODO.md for today's tasks
2. Check yesterday's commits
3. Plan today's 3 priority items
4. Read relevant docs

### During Development (7h)
1. Follow TDD: Red → Green → Refactor
2. Commit after each subtask completion
3. Run tests frequently
4. Take breaks every 90 minutes

### Every Evening (30 min)
1. Update TODO.md checkboxes
2. Commit all work
3. Push to remote
4. Plan tomorrow
5. Note blockers/questions

### Every Week End (1h)
1. Review week's progress
2. Update timeline if needed
3. Identify risks
4. Celebrate wins 🎉

---

## Emergency Procedures

### If Behind Schedule

**Day 5 Check** (should finish 8.1):
- If not done: Extend to Day 6, compress 8.2 by 1 day
- Skip some edge case tests (add later)
- Focus on core functionality

**Day 10 Check** (should finish 8.2):
- If not done: Critical - assess issues
- Can compress 8.3 to 1 day if simple
- May need to defer 8.4 or 8.5

**Day 12 Check** (should finish 8.3):
- If not done: Choose CLI OR API, not both
- Can add the other in v0.8.1
- Must keep 8.6 (testing)

### If Blocked

**Technical Blocker**:
1. Document the issue
2. Try alternative approach
3. Ask for help (team/community)
4. Timebox to 2 hours max

**Design Uncertainty**:
1. Check architectural-review document
2. Look at existing patterns in codebase
3. Make decision and document rationale
4. Can refactor later if needed

---

## Final Recommendations

### Absolute Must-Do (P0)
1. **Never skip tests** - 90% coverage non-negotiable
2. **Never skip 8.2 rollback testing** - Critical for stability
3. **Always verify SOLID principles** - Long-term maintainability
4. **Always commit daily** - Never lose work

### Strongly Recommended (P1)
1. **Do 8.4 AND 8.5** - Professional UX
2. **Take breaks** - Avoid burnout
3. **Code review each task** - Catch issues early
4. **Update docs as you go** - Easier than batch

### Nice to Have (P2)
1. Write additional examples
2. Create troubleshooting guide
3. Record demo video
4. Start thinking about v0.9.0

---

## Quick Reference Card

```
┌─────────────────────────────────────────────────────────┐
│                    PRIORITY SUMMARY                      │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  Week 1: FOUNDATION (Sequential)                        │
│  ├── Day 1-3:  Task 8.1  HeightMonitor        [P0] ⭐⭐⭐│
│  ├── Day 4-8:  Task 8.2  Orchestrator         [P0] ⭐⭐⭐│
│  └── Day 9-10: Task 8.3  Integration          [P0] ⭐⭐⭐│
│                                                          │
│  Week 2: INTERFACES (Can Parallel)                      │
│  ├── Day 11-12: Task 8.4  CLI Commands        [P1] ⭐⭐ │
│  └── Day 11-12: Task 8.5  API Endpoints       [P1] ⭐⭐ │
│                                                          │
│  Week 3: VALIDATION (Sequential)                        │
│  ├── Day 13-15: Task 8.6  E2E Testing         [P0] ⭐⭐⭐│
│  └── Day 16-17: Buffer & Release Prep         [P0] ⭐⭐⭐│
│                                                          │
│  Minimum Viable: 8.1 + 8.2 + 8.3 + basic 8.6           │
│  Recommended: ALL tasks 8.1-8.6                         │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

**Remember**:
- Quality > Speed
- Tests are not optional
- SOLID principles matter
- Document as you go
- Commit frequently
- Ask for help when stuck

**Let's build production-ready software! 🚀**
