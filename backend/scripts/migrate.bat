@echo off
REM Migration helper script for Windows

setlocal enabledelayedexpansion

cd /d %~dp0\..

REM Load .env if exists
if exist .env (
    for /f "usebackq tokens=1,* delims==" %%a in (.env) do (
        set "%%a=%%b"
    )
)

if "%1"=="up" (
    echo Running migrations up...
    go run ./cmd/migrate -action=up
    goto :end
)

if "%1"=="down" (
    set STEPS=%2
    if "!STEPS!"=="" set STEPS=1
    echo Rolling back !STEPS! migration(s)...
    go run ./cmd/migrate -action=down -steps=!STEPS!
    goto :end
)

if "%1"=="status" (
    echo Checking migration status...
    go run ./cmd/migrate -action=status
    goto :end
)

if "%1"=="create" (
    if "%2"=="" (
        echo Usage: %0 create ^<migration_name^>
        exit /b 1
    )
    echo Creating new migration: %2
    go run ./cmd/migrate -action=create -name=%2
    goto :end
)

if "%1"=="reset" (
    echo Resetting database (down all + up all)...
    go run ./cmd/migrate -action=down -steps=999
    go run ./cmd/migrate -action=up
    goto :end
)

echo Usage: %0 {up^|down [steps]^|status^|create ^<name^>^|reset}
echo.
echo Commands:
echo   up              - Apply all pending migrations
echo   down [steps]    - Rollback migrations (default: 1)
echo   status          - Show migration status
echo   create ^<name^>   - Create new migration files
echo   reset           - Rollback all and reapply all migrations
exit /b 1

:end
endlocal

