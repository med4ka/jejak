@echo off
rem Jalankan API dari folder backend/ (go run). .env dibaca dari root repo.
set PORT=8081
cd /d "%~dp0"
go run ./cmd/server
