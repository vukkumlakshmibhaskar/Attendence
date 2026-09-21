@echo off
title Attendance Backend (PostgreSQL + FastAPI)
cd /d "%~dp0"
echo ============================================================
echo   Starting Classroom Attendance Backend
echo   Database: PostgreSQL (127.0.0.1:5432/attendance)
echo   API URL:  http://localhost:8080
echo ============================================================
echo.
python server.py
pause
