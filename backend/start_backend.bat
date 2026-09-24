@echo off
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0..\start_local.ps1" -BackendOnly
if errorlevel 1 pause
