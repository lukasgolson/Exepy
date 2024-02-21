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

echo Checking for git installation...
where git > nul

if %errorlevel% neq 0 (
    echo Git is not installed. Please install git and try again.
    exit /b %errorlevel%
) else (
    echo Git is installed.
)

set DOWNLOAD_FLAG=false

echo checking for bootstrap source file...
if exist bootstrap.go (
    echo bootstrap.go found.
) else (
    echo bootstrap.go not found. Downloading bootstrap.go...
    curl -o bootstrap.go https://raw.githubusercontent.com/lukasgolson/Installer/master/bootstrap.go
	set DOWNLOAD_FLAG=true
)




git clone https://github.com/lukasgolson/PhotogrammetryPipeline.git repo
cd repo
echo | set /p="version: "
git describe --long --dirty --abbrev=10 --tags

git archive --format zip --output ../payload.zip master
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


echo Cleaning up...

if %DOWNLOAD_FLAG%==true (
    DEL /F /Q bootstrap.go
)


DEL /F /Q pipeline.zip


echo running bootstrap...
bootstrap.exe