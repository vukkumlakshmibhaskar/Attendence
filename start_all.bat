@echo off
title Attendance System Launcher
cd /d "%~dp0"

echo ============================================================
echo   Starting Classroom Attendance & Cognitive Analysis System
echo ============================================================
echo.

echo [1/2] Launching Backend Server (PostgreSQL + API)...
start "Attendance Backend" cmd /k "cd /d %~dp0backend && python server.py"

echo Waiting 2 seconds for backend initialization...
timeout /t 2 /nobreak >nul

echo [2/2] Launching Frontend Application (React + Vite)...
start "Attendance Frontend" cmd /k "cd /d %~dp0frontend && npm run dev"

echo.
echo ============================================================
echo   Both services are starting!
echo   Frontend UI: http://localhost:5173
echo   Backend API: http://localhost:8080
echo ============================================================
echo.
pause
