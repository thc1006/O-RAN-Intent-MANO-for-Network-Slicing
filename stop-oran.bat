@echo off
REM O-RAN Intent MANO - Stop All Services

title O-RAN Intent MANO - Stop Services

echo.
echo ╔══════════════════════════════════════════════════════════════╗
echo ║         O-RAN Intent MANO - 停止所有服務                     ║
echo ╚══════════════════════════════════════════════════════════════╝
echo.

echo 正在停止服務...
echo.

REM 停止 Python 進程
echo [1/4] 停止 NLP Service...
for /f "tokens=2" %%a in ('netstat -ano ^| findstr ":8082" ^| findstr "LISTENING"') do (
    taskkill /F /PID %%a >nul 2>&1
)
echo [✓] NLP Service 已停止

echo.
echo [2/4] 停止 WebSocket Server...
for /f "tokens=2" %%a in ('netstat -ano ^| findstr ":8081" ^| findstr "LISTENING"') do (
    taskkill /F /PID %%a >nul 2>&1
)
echo [✓] WebSocket Server 已停止

echo.
echo [3/4] 停止 Orchestrator...
for /f "tokens=2" %%a in ('netstat -ano ^| findstr ":8080" ^| findstr "LISTENING"') do (
    taskkill /F /PID %%a >nul 2>&1
)
taskkill /F /IM orchestrator.exe >nul 2>&1
echo [✓] Orchestrator 已停止

echo.
echo [4/4] 停止 Web UI...
for /f "tokens=2" %%a in ('netstat -ano ^| findstr ":8000" ^| findstr "LISTENING"') do (
    taskkill /F /PID %%a >nul 2>&1
)
echo [✓] Web UI 已停止

echo.
echo ╔══════════════════════════════════════════════════════════════╗
echo ║                  ✅ 所有服務已停止                           ║
echo ╚══════════════════════════════════════════════════════════════╝
echo.
timeout /t 2 /nobreak >nul
