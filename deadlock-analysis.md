# 데드락 분석 보고서

## 문제 상황

테스트에서 발생한 데드락은 다음과 같은 상황에서 발생합니다:

### 데드락 시나리오

1. **Manager.Start()** 실행
   - `m.stateMutex.Lock()` 획득 (라인 137)
   - 프로세스 시작
   - `m.metricsCollector.Start()` 호출 (라인 175) - **Lock을 유지한 채로 호출**
   - `m.stateMutex.Unlock()` (라인 181 이후)

2. **MetricsCollector.Start()** 실행
   - 즉시 `collectMetrics()` 호출
   - `c.nodeInfo.GetRestartCount()` 호출 (라인 116)

3. **Manager.GetRestartCount()** 실행
   - `m.stateMutex.RLock()` 시도 (라인 581) - **데드락 발생!**
   - Manager.Start()가 이미 Write Lock을 보유하고 있음

## 데드락 발생 조건

```
Thread 1 (Manager.Start):
1. stateMutex.Lock() 획득
2. metricsCollector.Start() 호출
3. Lock 해제 대기

Thread 2 (MetricsCollector):
1. collectMetrics() 실행
2. GetRestartCount() 호출
3. stateMutex.RLock() 대기 <- 데드락!
```

## 실제 런타임에서도 발생 가능한가?

**예, 실제 런타임에서도 발생할 수 있습니다!**

### 발생 조건:
1. Manager.Start()가 호출될 때
2. MetricsCollector가 즉시 메트릭 수집을 시작할 때
3. 두 작업이 동시에 실행될 때

### 왜 지금까지 발견되지 않았나?
- 대부분의 경우 Manager.Start()가 빠르게 완료되어 Lock이 해제됨
- 테스트 환경에서는 타이밍이 더 타이트하게 실행되어 데드락이 쉽게 발생

## 해결 방법

### 방법 1: Lock 범위 축소 (권장)
```go
func (m *Manager) Start(args []string) error {
    m.stateMutex.Lock()
    // ... 프로세스 시작
    m.state = StateRunning
    m.stateMutex.Unlock()  // Lock을 먼저 해제

    // Lock 외부에서 다른 컴포넌트 시작
    go m.monitor()
    m.healthChecker.Start()
    m.metricsCollector.Start()  // Lock 없이 호출

    return nil
}
```

### 방법 2: 지연된 메트릭 수집
```go
func (c *MetricsCollector) Start() {
    go func() {
        // 첫 수집을 지연시킴
        time.Sleep(100 * time.Millisecond)
        c.collectMetrics()

        ticker := time.NewTicker(c.interval)
        // ...
    }()
}
```

### 방법 3: 별도의 메트릭용 Mutex 사용
```go
type Manager struct {
    stateMutex    sync.RWMutex
    metricsMutex  sync.RWMutex  // 별도의 mutex
    restartCount  int
    // ...
}

func (m *Manager) GetRestartCount() int {
    m.metricsMutex.RLock()
    defer m.metricsMutex.RUnlock()
    return m.restartCount
}
```

## 검증 방법

### 1. 데드락 검출 테스트
```go
func TestManager_NoDealock(t *testing.T) {
    // 데드락 검출을 위한 timeout 설정
    done := make(chan bool)

    go func() {
        manager := NewManager(cfg, logger)
        err := manager.Start([]string{"--test"})
        assert.NoError(t, err)
        done <- true
    }()

    select {
    case <-done:
        // 성공
    case <-time.After(1 * time.Second):
        t.Fatal("Deadlock detected: Start() did not complete")
    }
}
```

### 2. Race Detector 사용
```bash
go test -race ./internal/node
```

### 3. 동시성 테스트
```go
func TestConcurrentStartStop(t *testing.T) {
    manager := NewManager(cfg, logger)

    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            manager.Start([]string{})
            time.Sleep(10 * time.Millisecond)
            manager.Stop()
        }()
    }
    wg.Wait()
}
```

## 결론

이 데드락은 **실제 프로덕션 환경에서도 발생할 수 있는 심각한 버그**입니다. Manager.Start()에서 Lock을 보유한 상태로 MetricsCollector.Start()를 호출하는 것이 근본 원인이며, 이는 반드시 수정되어야 합니다.

권장 해결책은 Lock 범위를 최소화하여 다른 컴포넌트 시작 전에 Lock을 해제하는 것입니다.