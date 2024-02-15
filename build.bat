@echo off


echo Checking for go installation...
where go > nul

if %errorlevel% neq 0 (
    echo Go is not installed. Please install go and try again.
    exit /b %errorlevel%
) else (
    echo Go is installed.
)

echo Checking for git installation...
where git > nul

if %errorlevel% neq 0 (
    echo Git is not installed. Please install git and try again.
    exit /b %errorlevel%
) else (
    echo Git is installed.
)

echo checking for bootstrap source file...
if exist bootstrap.go (
    echo bootstrap.go found.
) else (
    echo bootstrap.go not found. Downloading bootstrap.go...
    curl -o bootstrap.go https://raw.githubusercontent.com/lukasgolson/Installer/master/bootstrap.go
)




git clone https://github.com/lukasgolson/PhotogrammetryPipeline.git repo
cd repo
git archive --format zip --output ../pipeline.zip master
cd ..
rmdir /S /Q repo

echo Executing go build bootstrap...
go build bootstrap.go

if %errorlevel% neq 0 (
    echo Compilation failed.
    exit /b %errorlevel%
) else (
    echo Compilation successful.
)

echo Script execution completed.

DEL /F /Q pipeline.zip