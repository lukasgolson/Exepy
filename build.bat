@echo off
set url=https://github.com/lukasgolson/PhotogrammetryPipeline/archive/refs/heads/MultiChunking.zip
set filename=pipeline.zip

echo Downloading %url% to %filename%...
curl -o %filename% -LJ %url%

if %errorlevel% neq 0 (
    echo Download failed.
    exit /b %errorlevel%
) else (
    echo Download successful.
)

echo Executing go build install.go...
go build install.go

if %errorlevel% neq 0 (
    echo Compilation failed.
    exit /b %errorlevel%
) else (
    echo Compilation successful.
)

echo Script execution completed.
