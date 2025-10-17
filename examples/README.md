# Wemixvisor Examples

이 디렉토리에는 wemixvisor의 다양한 사용 시나리오에 대한 예제와 설정 파일이 포함되어 있습니다.

## 디렉토리 구조

```
examples/
├── config/              # 설정 파일 예제
│   ├── basic-config.toml       # 기본 설정
│   ├── advanced-config.toml    # 고급 기능 설정
│   └── production-config.toml  # 프로덕션 환경 설정
├── scripts/             # 유틸리티 스크립트
│   └── start-with-monitoring.sh
└── README.md           # 이 파일
```

## 설정 파일 예제

### 1. Basic Config (`config/basic-config.toml`)

최소한의 설정으로 빠르게 시작하기 위한 기본 설정입니다.

**사용 방법:**
```bash
# 설정 파일 복사
cp examples/config/basic-config.toml ~/.wemixd/config/wemixvisor.toml

# 필요에 따라 수정
vi ~/.wemixd/config/wemixvisor.toml

# 실행
wemixvisor start --home ~/.wemixd
```

**주요 기능:**
- 기본 노드 관리
- 간단한 모니터링
- 최소한의 로깅

### 2. Advanced Config (`config/advanced-config.toml`)

모든 Phase 7 기능을 활용하는 완전한 설정입니다.

**사용 방법:**
```bash
cp examples/config/advanced-config.toml /opt/wemixd/config/wemixvisor.toml

# 알림 설정 (환경 변수)
export WEMIXVISOR_ALERT_EMAIL_PASSWORD="your-password"
export WEMIXVISOR_ALERT_SLACK_WEBHOOK="https://hooks.slack.com/..."
export WEMIXVISOR_ALERT_DISCORD_WEBHOOK="https://discord.com/api/webhooks/..."

wemixvisor start --home /opt/wemixd
```

**주요 기능:**
- 전체 메트릭 수집
- Prometheus 익스포트
- 알림 시스템 (Email, Slack, Discord, Webhook)
- 성능 프로파일링
- 자동 백업
- 성능 최적화

### 3. Production Config (`config/production-config.toml`)

프로덕션 환경에 최적화된 안전하고 신뢰할 수 있는 설정입니다.

**사용 방법:**
```bash
# Root 권한으로 설치
sudo cp examples/config/production-config.toml /etc/wemixd/wemixvisor.toml

# 환경 변수 설정
export WEMIXVISOR_ALERT_EMAIL_PASSWORD="secure-password"
export WEMIXVISOR_ALERT_SLACK_WEBHOOK="https://hooks.slack.com/..."
export WEMIXVISOR_API_KEY="secure-api-key"

# Systemd 서비스로 실행
sudo systemctl start wemixvisor
```

**프로덕션 특징:**
- 수동 바이너리 검증 (자동 다운로드 비활성화)
- 포괄적인 알림 규칙
- 크리티컬 알림
- 장기 로그 보관
- JSON 로그 형식 (로그 수집 시스템 통합)
- 향상된 백업 정책
- 보안 설정

## CLI 사용 예제

### API 서버 시작

```bash
# 기본 API 서버
wemixvisor api --port 8080

# 메트릭과 거버넌스 모니터링 활성화
wemixvisor api --port 8080 --enable-metrics --enable-governance

# 커스텀 메트릭 간격
wemixvisor api --port 8080 --metrics-interval 5
```

### 메트릭 관리

```bash
# 현재 메트릭 표시
wemixvisor metrics show

# JSON 형식으로 표시
wemixvisor metrics show --json

# 실시간 모니터링 (5초마다 갱신)
wemixvisor metrics show --watch --interval 5

# 메트릭 수집 시작 (60초 동안)
wemixvisor metrics collect --interval 10 --duration 60

# Prometheus 익스포터 시작
wemixvisor metrics export --port 9090
```

### 성능 프로파일링

```bash
# CPU 프로파일 (30초)
wemixvisor profile cpu --duration 30

# 메모리 힙 프로파일
wemixvisor profile heap

# 고루틴 프로파일
wemixvisor profile goroutine

# 모든 프로파일 캡처
wemixvisor profile all

# 저장된 프로파일 목록
wemixvisor profile list

# 오래된 프로파일 정리 (7일 이상)
wemixvisor profile clean --max-age 168h
```

## 스크립트 사용

### 모니터링과 함께 시작

```bash
# 실행 권한 부여
chmod +x examples/scripts/start-with-monitoring.sh

# 기본 설정으로 실행
./examples/scripts/start-with-monitoring.sh

# 커스텀 포트 설정
WEMIXVISOR_API_PORT=9000 WEMIXVISOR_METRICS_PORT=9100 \\
  ./examples/scripts/start-with-monitoring.sh
```

이 스크립트는:
- API 서버 시작 (포트 8080)
- Prometheus 익스포터 시작 (포트 9090)
- 접속 URL 표시
- 종료 시 자동 정리

## 모니터링 통합

### Prometheus 설정

`prometheus.yml`에 추가:

```yaml
scrape_configs:
  - job_name: 'wemixvisor'
    static_configs:
      - targets: ['localhost:9090']
    scrape_interval: 15s
```

### Grafana 대시보드

Grafana 대시보드 템플릿은 `examples/grafana/` 디렉토리를 참조하세요.

### WebSocket 연결

```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/ws');

ws.onopen = () => {
  // 메트릭 구독
  ws.send(JSON.stringify({
    action: 'subscribe',
    topics: ['metrics', 'alerts', 'logs']
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Received:', data.type, data.topic, data.data);
};
```

## API 엔드포인트

### 헬스체크
```bash
curl http://localhost:8080/health
```

### 상태 조회
```bash
curl http://localhost:8080/api/v1/status
```

### 메트릭 조회
```bash
curl http://localhost:8080/api/v1/metrics
```

### 업그레이드 목록
```bash
curl http://localhost:8080/api/v1/upgrades
```

### 거버넌스 제안 목록
```bash
curl http://localhost:8080/api/v1/governance/proposals
```

## 문제 해결

### API 서버가 시작되지 않는 경우

```bash
# 포트가 이미 사용 중인지 확인
lsof -i :8080

# 로그 확인
wemixvisor api --debug
```

### 메트릭이 수집되지 않는 경우

```bash
# 수집기 상태 확인
curl http://localhost:8080/api/v1/metrics

# 수동으로 메트릭 수집 시작
wemixvisor metrics collect --interval 5 --duration 10
```

### 프로파일 분석

```bash
# CPU 프로파일 분석
go tool pprof /opt/wemixd/profiles/cpu_*.prof

# 웹 UI로 보기
go tool pprof -http=:8081 /opt/wemixd/profiles/cpu_*.prof
```

## 추가 리소스

- [메트릭 문서](../docs/metrics.md)
- [알림 설정 가이드](../docs/alerting.md)
- [Grafana 대시보드 가이드](../docs/grafana.md)
- [API 문서](../docs/api.md)

## 지원

문제가 발생하거나 질문이 있는 경우:
- GitHub Issues: https://github.com/wemix/wemixvisor/issues
- 문서: https://github.com/wemix/wemixvisor/docs
