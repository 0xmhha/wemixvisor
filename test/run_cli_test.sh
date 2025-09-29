#!/bin/bash

# CLI 테스트 스크립트
set -e

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 테스트 카운터
PASSED=0
FAILED=0

# 테스트 함수
test_command() {
    local test_name="$1"
    local command="$2"
    local expected="$3"

    echo -n "Testing: $test_name ... "

    if $command 2>&1 | grep -q "$expected"; then
        echo -e "${GREEN}PASSED${NC}"
        ((PASSED++))
    else
        echo -e "${RED}FAILED${NC}"
        echo "  Command: $command"
        echo "  Expected: $expected"
        ((FAILED++))
    fi
}

echo "========================================="
echo "Wemixvisor CLI Test Suite"
echo "========================================="

# 환경 설정
export DAEMON_HOME=/tmp/wemixd_test
export DAEMON_NAME=wemixd
WEMIXVISOR="/Users/wm-it-22-00661/workspace/cosmovisor/wemixvisor/bin/wemixvisor"

# 테스트 환경 준비
echo "Setting up test environment..."
rm -rf /tmp/wemixd_test
/Users/wm-it-22-00661/workspace/cosmovisor/wemixvisor/test/setup_test_env.sh /tmp/wemixd_test > /dev/null 2>&1

echo ""
echo "Running tests..."
echo "-----------------------------------------"

# 1. Help 명령어 테스트
test_command "Help command" "$WEMIXVISOR --help" "WBFT Node Lifecycle Manager"

# 2. Version 명령어 테스트
test_command "Version command" "$WEMIXVISOR version" "wemixvisor version: v0.4.0"

# 3. Status 명령어 테스트 (초기 상태)
test_command "Initial status" "$WEMIXVISOR status" "Node Status: stopped"

# 4. Start 명령어 테스트
echo -n "Testing: Start command ... "
$WEMIXVISOR start --daemon --datadir /tmp/data --port 30303 > /dev/null 2>&1
sleep 2
if $WEMIXVISOR status 2>&1 | grep -q "Node Status: running"; then
    echo -e "${GREEN}PASSED${NC}"
    ((PASSED++))
else
    echo -e "${RED}FAILED${NC}"
    ((FAILED++))
fi

# 5. Status 명령어 테스트 (실행 중)
test_command "Running status" "$WEMIXVISOR status" "Node Status: running"

# 6. JSON Status 테스트
echo -n "Testing: JSON status ... "
if $WEMIXVISOR status --json 2>&1 | grep -q '"state_string"'; then
    echo -e "${GREEN}PASSED${NC}"
    ((PASSED++))
else
    echo -e "${RED}FAILED${NC}"
    ((FAILED++))
fi

# 7. 중복 Start 테스트 (에러 예상)
echo -n "Testing: Duplicate start (should fail) ... "
if $WEMIXVISOR start 2>&1 | grep -q "already running"; then
    echo -e "${GREEN}PASSED${NC}"
    ((PASSED++))
else
    echo -e "${RED}FAILED${NC}"
    ((FAILED++))
fi

# 8. Stop 명령어 테스트
echo -n "Testing: Stop command ... "
$WEMIXVISOR stop > /dev/null 2>&1
sleep 2
if $WEMIXVISOR status 2>&1 | grep -q "Node Status: stopped"; then
    echo -e "${GREEN}PASSED${NC}"
    ((PASSED++))
else
    echo -e "${RED}FAILED${NC}"
    ((FAILED++))
fi

# 9. 중복 Stop 테스트 (에러 예상)
echo -n "Testing: Duplicate stop (should fail) ... "
if $WEMIXVISOR stop 2>&1 | grep -q "not running"; then
    echo -e "${GREEN}PASSED${NC}"
    ((PASSED++))
else
    echo -e "${RED}FAILED${NC}"
    ((FAILED++))
fi

# 10. Restart 테스트
echo -n "Testing: Restart command ... "
$WEMIXVISOR start --daemon > /dev/null 2>&1
sleep 1
$WEMIXVISOR restart > /dev/null 2>&1
sleep 2
if $WEMIXVISOR status 2>&1 | grep -q "Node Status: running"; then
    echo -e "${GREEN}PASSED${NC}"
    ((PASSED++))
else
    echo -e "${RED}FAILED${NC}"
    ((FAILED++))
fi

# 11. 플래그 분리 테스트
echo -n "Testing: Mixed flags ... "
$WEMIXVISOR stop > /dev/null 2>&1
sleep 1
$WEMIXVISOR start --daemon --home /tmp/wemixd_test --network testnet --datadir /tmp/data --syncmode full > /dev/null 2>&1
sleep 2
if [ -f /tmp/mock_node.log ] && grep -q "syncmode full" /tmp/mock_node.log; then
    echo -e "${GREEN}PASSED${NC}"
    ((PASSED++))
else
    echo -e "${RED}FAILED${NC}"
    ((FAILED++))
fi

# 정리
echo -n "Cleaning up ... "
$WEMIXVISOR stop > /dev/null 2>&1 || true
sleep 1
echo -e "${GREEN}DONE${NC}"

echo ""
echo "========================================="
echo "Test Results"
echo "========================================="
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi