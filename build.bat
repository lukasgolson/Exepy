@ echo off
echo building...
@echo off


echo Checking for go installation...
where go > nul

if %errorlevel% neq 0 (
    echo Go is not installed. Please install go and try again.
    exit /b %errorlevel%
) else (
    echo Go is installed.
)

cd main
go build -o ..\creator.exe main.go