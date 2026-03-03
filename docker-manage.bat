@echo off
REM AmmanGate Docker Management Script for Windows
setlocal enabledelayedexpansion

set SCRIPT_DIR=%~dp0
set PROJECT_DIR=%SCRIPT_DIR%

echo ======================================
echo  AmmanGate Docker Setup
echo ======================================

if "%1"=="" goto :help
if /I "%1"=="build" goto :build
if /I "%1"=="start" goto :start
if /I "%1"=="stop" goto :stop
if /I "%1"=="restart" goto :restart
if /I "%1"=="logs" goto :logs
if /I "%1"=="status" goto :status
if /I "%1"=="shell" goto :shell
if /I "%1"=="update-av" goto :update_av
if /I "%1"=="clean" goto :clean
if /I "%1"=="help" goto :help
goto :help

:build
    echo [INFO] Building AmmanGate Docker images...
    cd /d %PROJECT_DIR%
    docker compose build
    goto :end

:start
    echo [INFO] Starting AmmanGate containers...
    cd /d %PROJECT_DIR%
    docker compose up -d
    echo [INFO] AmmanGate containers started
    goto :end

:stop
    echo [INFO] Stopping AmmanGate containers...
    cd /d %PROJECT_DIR%
    docker compose down
    echo [INFO] AmmanGate containers stopped
    goto :end

:restart
    call :stop
    call :start
    goto :end

:logs
    cd /d %PROJECT_DIR%
    docker compose logs -f
    goto :end

:status
    cd /d %PROJECT_DIR%
    docker compose ps
    goto :end

:shell
    echo [INFO] Opening shell in AmmanGate container...
    cd /d %PROJECT_DIR%
    docker compose exec bodyguard-core /bin/bash
    goto :end

:update-av
    echo [INFO] Updating ClamAV virus definitions...
    cd /d %PROJECT_DIR%
    docker compose exec bodyguard-core freshclam --datadir=/var/lib/clamav
    echo [INFO] ClamAV definitions updated
    goto :end

:clean
    echo [WARN] This will remove all containers and volumes!
    set /P CONFIRM="Are you sure? (y/N): "
    if /I "!CONFIRM!"=="y" (
        cd /d %PROJECT_DIR%
        docker compose down -v
        echo [INFO] Containers and volumes removed
    ) else (
        echo [INFO] Cancelled
    )
    goto :end

:help
    echo AmmanGate Docker Management Script
    echo.
    echo Usage: docker-manage.bat [COMMAND]
    echo.
    echo Commands:
    echo     build       Build Docker images
    echo     start       Start containers
    echo     stop        Stop containers
    echo     restart     Restart containers
    echo     logs        View container logs
    echo     status      Show container status
    echo     shell       Open shell in container
    echo     update-av   Update ClamAV virus definitions
    echo     clean       Remove containers and volumes
    echo     help        Show this help message
    echo.
    echo Examples:
    echo     docker-manage.bat build
    echo     docker-manage.bat start
    echo     docker-manage.bat logs
    echo.

:end
    endlocal
