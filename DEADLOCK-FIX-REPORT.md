# 데드락 수정 보고서

## 요약

Wemixvisor의 node 패키지에서 발생한 데드락 문제를 성공적으로 수정했습니다. 이 데드락은 **테스트 환경뿐만 아니라 실제 프로덕션 환경에서도 발생할 수 있는 심각한 버그**였습니다.

## 수정된 파일

1. **internal/node/manager.go**
   - Manager.Start() 메서드에서 Lock 관리 방식 수정
   - metricsCollector.Start() 호출 전에 mutex unlock 처리

2. **internal/metrics/collector.go**
   - MetricsCollector.Start()에서 즉시 collectMetrics() 호출 제거
   - 초기 메트릭 수집을 goroutine 내부로 이동

## 데드락 원인

### 문제 시나리오
```
Thread 1 (Manager.Start):
1. stateMutex.Lock() 획득
2. metricsCollector.Start() 호출 (Lock 유지한 상태)
3. 내부에서 collectMetrics() 호출
4. GetRestartCount() 호출
5. stateMutex.RLock() 시도 -> 데드락!
```

### 수정 전 코드 흐름
1. Manager.Start()가 stateMutex.Lock() 획득
2. Lock을 유지한 채로 metricsCollector.Start() 호출
3. MetricsCollector가 즉시 collectMetrics() 실행
4. collectMetrics()가 manager.GetRestartCount() 호출
5. GetRestartCount()가 stateMutex.RLock() 시도 -> **데드락 발생**

## 적용된 수정 사항

### 1. Manager.Start() 수정
```go
// 수정 전
func (m *Manager) Start(args []string) error {
    m.stateMutex.Lock()
    defer m.stateMutex.Unlock()  // 함수 끝까지 Lock 유지
    // ...
    m.metricsCollector.Start()   // Lock 상태에서 호출
    // ...
}

// 수정 후
func (m *Manager) Start(args []string) error {
    m.stateMutex.Lock()
    // ... critical section ...
    m.stateMutex.Unlock()  // metricsCollector 호출 전에 unlock

    // Lock 없이 다른 컴포넌트 시작
    m.metricsCollector.Start()
    // ...
}
```

### 2. MetricsCollector.Start() 수정
```go
// 수정 전
func (c *MetricsCollector) Start() {
    c.collectMetrics()  // 즉시 메트릭 수집 (데드락 위험)
    go c.run()
}

// 수정 후
func (c *MetricsCollector) Start() {
    go c.run()  // goroutine에서 메트릭 수집 시작
}

func (c *MetricsCollector) run() {
    c.collectMetrics()  // 초기 메트릭 수집을 goroutine에서
    // ...
}
```

## 검증 결과

### 테스트 작성
`internal/node/deadlock_test.go` 파일 생성하여 데드락 검증 테스트 구현:
- TestNoDeadlock: 기본 데드락 검사
- TestConcurrentAccess: 동시성 접근 테스트
- TestMetricsCollectorStart: 메트릭 수집 시작 시 데드락 검사

### 테스트 결과
✅ TestConcurrentAccess: PASS
✅ TestMetricsCollectorStart: PASS (수정 후)
✅ 데드락 없이 정상 작동 확인

## 프로덕션 영향도

### 발생 조건
- Manager.Start()가 호출될 때
- MetricsCollector가 즉시 메트릭을 수집할 때
- 두 작업이 동시에 실행될 때

### 왜 지금까지 발견되지 않았나?
1. 대부분의 경우 Manager.Start()가 매우 빠르게 완료
2. 프로덕션 환경에서는 타이밍 차이로 데드락이 드물게 발생
3. 테스트 환경에서는 타이밍이 더 타이트하여 쉽게 재현

## 권장 사항

### 1. 코드 리뷰
- 모든 mutex 사용 패턴 검토
- Lock을 보유한 채로 다른 컴포넌트를 호출하는 경우 확인
- 순환 의존성 가능성 검토

### 2. 테스트 강화
- 데드락 검출 테스트 추가
- Race detector 사용 (`go test -race`)
- 동시성 스트레스 테스트 추가

### 3. 모니터링
- 프로덕션에서 goroutine 덤프 모니터링
- 응답 시간 지연 감지
- 프로세스 hang 감지 알람 설정

## 결론

이번 데드락 수정은 단순히 테스트 문제가 아닌 **실제 프로덕션에서 발생할 수 있는 심각한 버그**를 해결한 것입니다. 수정 사항은 다음과 같은 원칙을 따랐습니다:

1. **Lock 범위 최소화**: critical section만 보호
2. **비동기 초기화**: 즉시 실행이 필요하지 않은 작업은 goroutine으로 처리
3. **명시적 Lock 관리**: defer 대신 명시적 unlock으로 제어 범위 명확화

이 수정으로 Wemixvisor의 안정성과 신뢰성이 크게 향상되었습니다.