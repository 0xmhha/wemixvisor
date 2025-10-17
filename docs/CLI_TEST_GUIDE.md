# CLI 기능 테스트 가이드

## 준비 사항

### 1. 빌드
```bash
cd /Users/wm-it-22-00661/workspace/cosmovisor/wemixvisor
go build -o bin/wemixvisor ./cmd/wemixvisor
```

### 2. 테스트 환경 설정
```bash
# 테스트 환경 설정 스크립트 실행
./test/setup_test_env.sh ~/.wemixd_test

# 환경 변수 설정
export DAEMON_HOME=~/.wemixd_test
export DAEMON_NAME=wemixd
```

## 기본 명령어 테스트

### 1. 도움말 및 버전
```bash
# 도움말
./bin/wemixvisor --help

# 버전 확인
./bin/wemixvisor version
```

### 2. 노드 시작 (Start)
```bash
# 기본 시작
./bin/wemixvisor start

# Geth 호환 인자와 함께 시작
./bin/wemixvisor start --datadir /tmp/data --syncmode full --port 30303

# wemixvisor 플래그와 geth 인자 혼합
./bin/wemixvisor start --home ~/.wemixd_test --network testnet --datadir /tmp/data --port 8545

# 백그라운드에서 실행
./bin/wemixvisor start --daemon
```

### 3. 상태 확인 (Status)
```bash
# 일반 상태 확인
./bin/wemixvisor status

# JSON 형식으로 출력
./bin/wemixvisor status --json
```

### 4. 노드 중지 (Stop)
```bash
./bin/wemixvisor stop
```

### 5. 노드 재시작 (Restart)
```bash
# 기본 재시작
./bin/wemixvisor restart

# 새로운 인자로 재시작
./bin/wemixvisor restart --datadir /new/data --port 9000
```

### 6. 로그 확인 (Logs)
```bash
# 마지막 100줄 보기 (기본값)
./bin/wemixvisor logs

# 마지막 50줄 보기
./bin/wemixvisor logs --tail=50

# 실시간 로그 따라가기
./bin/wemixvisor logs --follow
```

## CLI 플래그 테스트

### Wemixvisor 전용 플래그
```bash
--home <path>      # 홈 디렉토리 설정
--network <name>   # 네트워크 설정 (mainnet/testnet)
--debug            # 디버그 모드
--json             # JSON 출력
--quiet            # 출력 억제
--daemon           # 백그라운드 실행
```

### Geth 호환 인자 (패스스루)
```bash
# 데이터 디렉토리
--datadir <path>

# 동기화 모드
--syncmode full|fast|light

# 네트워크 포트
--port 30303

# RPC 설정
--http
--http.addr 0.0.0.0
--http.port 8545
--http.api eth,net,web3

# WebSocket 설정
--ws
--ws.addr 0.0.0.0
--ws.port 8546
--ws.api eth,net,web3
```

## 고급 테스트 시나리오

### 1. 플래그 분리 테스트
```bash
# wemixvisor 플래그와 geth 플래그 혼합
./bin/wemixvisor start \
  --home ~/.wemixd_test \     # wemixvisor
  --network testnet \          # wemixvisor
  --debug \                    # wemixvisor
  --datadir /data \           # geth
  --syncmode full \           # geth
  --port 30303 \              # geth
  --http \                    # geth
  --http.port 8545            # geth
```

### 2. 환경 변수 테스트
```bash
# 환경 변수로 설정
export DAEMON_HOME=~/.wemixd_test
export DAEMON_NAME=wemixd
export DAEMON_NETWORK=testnet
export DAEMON_DEBUG=true

# 명령 실행
./bin/wemixvisor start
```

### 3. JSON 출력 테스트
```bash
# JSON 형식 상태
./bin/wemixvisor status --json | jq .

# 파싱 예제
./bin/wemixvisor status --json | jq '.state_string'
./bin/wemixvisor status --json | jq '.pid'
```

### 4. 에러 케이스 테스트
```bash
# 이미 실행 중일 때 start
./bin/wemixvisor start
./bin/wemixvisor start  # 에러: already running

# 실행 중이 아닐 때 stop
./bin/wemixvisor stop
./bin/wemixvisor stop  # 에러: not running

# 잘못된 명령어
./bin/wemixvisor invalid  # 에러: unknown command

# stop에 노드 인자 전달
./bin/wemixvisor stop --datadir /data  # 에러: does not accept node arguments
```

## 로그 확인

### Mock 노드 로그
```bash
# mock 노드가 받은 인자 확인
cat /tmp/mock_node.log
```

### Wemixvisor 로그
```bash
# 설정된 로그 파일 확인
cat ~/.wemixd_test/logs/node.log
```

## 프로세스 확인

```bash
# 실행 중인 프로세스 확인
ps aux | grep wemixd

# 프로세스 트리 확인
pstree -p $(pgrep wemixvisor)
```

## 테스트 정리

```bash
# 노드 중지
./bin/wemixvisor stop

# 테스트 환경 제거 (옵션)
rm -rf ~/.wemixd_test

# 로그 정리
rm -f /tmp/mock_node.log
```

## 유닛 테스트 실행

```bash
# CLI 패키지 테스트
go test ./internal/cli/ -v

# 커버리지 확인
go test ./internal/cli/ -cover

# 특정 테스트만 실행
go test ./internal/cli/ -run TestCLI_Integration -v

# 전체 프로젝트 테스트
go test ./... -v
```

## 테스트 체크리스트

- [ ] `wemixvisor --help` 명령어가 도움말을 표시하는가?
- [ ] `wemixvisor version` 명령어가 버전을 표시하는가?
- [ ] `wemixvisor start`로 노드가 시작되는가?
- [ ] Geth 인자가 올바르게 전달되는가?
- [ ] `wemixvisor status`가 현재 상태를 표시하는가?
- [ ] `wemixvisor status --json`이 JSON 형식으로 출력되는가?
- [ ] `wemixvisor stop`으로 노드가 중지되는가?
- [ ] `wemixvisor restart`로 노드가 재시작되는가?
- [ ] 새로운 인자로 재시작이 가능한가?
- [ ] wemixvisor 플래그와 geth 인자가 올바르게 분리되는가?
- [ ] 에러 상황에서 적절한 메시지가 표시되는가?

## 문제 해결

### 1. "binary not found" 에러
```bash
# 바이너리 경로 확인
ls -la $DAEMON_HOME/wemixvisor/current/bin/

# 심볼릭 링크 확인
ls -la $DAEMON_HOME/wemixvisor/current
```

### 2. 권한 문제
```bash
# 실행 권한 부여
chmod +x $DAEMON_HOME/wemixvisor/genesis/bin/wemixd
```

### 3. 환경 변수 확인
```bash
echo $DAEMON_HOME
echo $DAEMON_NAME
```