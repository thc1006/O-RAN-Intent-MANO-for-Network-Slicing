@echo off
REM O-RAN Intent MANO - Windows Launcher
REM 一鍵啟動所有服務

setlocal enabledelayedexpansion

title O-RAN Intent MANO - Launcher

echo.
echo ╔══════════════════════════════════════════════════════════════╗
echo ║         O-RAN Intent MANO for Network Slicing               ║
echo ║              Windows 一鍵啟動程序                            ║
echo ╚══════════════════════════════════════════════════════════════╝
echo.

REM 獲取腳本所在目錄
set "SCRIPT_DIR=%~dp0"
cd /d "%SCRIPT_DIR%"

REM 檢查依賴
echo [1/6] 檢查依賴...
where python >nul 2>&1
if %errorlevel% neq 0 (
    echo [錯誤] 未找到 Python，請先安裝 Python 3.11+
    pause
    exit /b 1
)

where go >nul 2>&1
if %errorlevel% neq 0 (
    echo [警告] 未找到 Go，將跳過 Orchestrator 編譯
    set "SKIP_GO=1"
)

echo [✓] 依賴檢查完成

REM 創建日誌目錄
if not exist "logs" mkdir logs
echo [✓] 日誌目錄已創建

REM 啟動 NLP Service
echo.
echo [2/6] 啟動 NLP Service (端口 8082)...
start "NLP Service" /min cmd /c "cd nlp && python nlp_service.py > ../logs/nlp.log 2>&1"
timeout /t 2 /nobreak >nul
echo [✓] NLP Service 已啟動

REM 編譯並啟動 Orchestrator
echo.
echo [3/6] 啟動 Orchestrator (端口 8080)...
if not defined SKIP_GO (
    if not exist "orchestrator\bin" mkdir orchestrator\bin
    cd orchestrator
    echo    正在編譯 Orchestrator...
    go build -o bin\orchestrator.exe cmd\orchestrator\main.go >nul 2>&1
    if %errorlevel% equ 0 (
        echo    [✓] 編譯成功
        start "Orchestrator" /min cmd /c "bin\orchestrator.exe --server > ../logs/orchestrator.log 2>&1"
    ) else (
        echo    [警告] 編譯失敗，使用 go run
        start "Orchestrator" /min cmd /c "go run cmd\orchestrator\main.go --server > ../logs/orchestrator.log 2>&1"
    )
    cd ..
) else (
    echo    [跳過] Go 未安裝
)
timeout /t 3 /nobreak >nul
echo [✓] Orchestrator 已啟動

REM 啟動 WebSocket Server
echo.
echo [4/6] 啟動 WebSocket Server (端口 8081)...
start "WebSocket Server" /min cmd /c "python websocket_server.py > logs\websocket.log 2>&1"
timeout /t 2 /nobreak >nul
echo [✓] WebSocket Server 已啟動

REM 啟動 Web UI
echo.
echo [5/6] 啟動 Web UI (端口 8000)...
start "Web UI" /min cmd /c "cd web && python -m http.server 8000 > ../logs/webui.log 2>&1"
timeout /t 2 /nobreak >nul
echo [✓] Web UI 已啟動

REM 驗證服務
echo.
echo [6/6] 驗證服務狀態...
timeout /t 3 /nobreak >nul

curl -s http://localhost:8082/health >nul 2>&1
if %errorlevel% equ 0 (
    echo [✓] NLP Service: 運行中
) else (
    echo [✗] NLP Service: 未響應
)

curl -s http://localhost:8080/health >nul 2>&1
if %errorlevel% equ 0 (
    echo [✓] Orchestrator: 運行中
) else (
    echo [✗] Orchestrator: 未響應
)

netstat -an | find "8081" | find "LISTENING" >nul 2>&1
if %errorlevel% equ 0 (
    echo [✓] WebSocket: 運行中
) else (
    echo [✗] WebSocket: 未響應
)

netstat -an | find "8000" | find "LISTENING" >nul 2>&1
if %errorlevel% equ 0 (
    echo [✓] Web UI: 運行中
) else (
    echo [✗] Web UI: 未響應
)

REM 顯示訪問信息
echo.
echo ╔══════════════════════════════════════════════════════════════╗
echo ║                    🎉 啟動完成！                             ║
echo ╚══════════════════════════════════════════════════════════════╝
echo.
echo 服務地址：
echo   📡 NLP Service:      http://localhost:8082
echo   🚀 Orchestrator:     http://localhost:8080
echo   💬 WebSocket:        ws://localhost:8081/ws
echo   🌐 Web UI:           http://localhost:8000/index.html
echo.
echo 日誌位置：
echo   logs\nlp.log
echo   logs\orchestrator.log
echo   logs\websocket.log
echo   logs\webui.log
echo.
echo 正在打開瀏覽器...
timeout /t 2 /nobreak >nul
start http://localhost:8000/index.html

echo.
echo ╔══════════════════════════════════════════════════════════════╗
echo ║  按任意鍵保持此窗口開啟（可查看狀態）                        ║
echo ║  要停止所有服務，請運行 stop-oran.bat                        ║
echo ╚══════════════════════════════════════════════════════════════╝
echo.
pause
