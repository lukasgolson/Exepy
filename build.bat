@echo off
git clone https://github.com/lukasgolson/PhotogrammetryPipeline.git repo
cd repo
git archive --format zip --output ../pipeline.zip master
cd ..
rmdir /S /Q repo

echo Executing go build bootstrap.go...
go build bootstrap.go

if %errorlevel% neq 0 (
    echo Compilation failed.
    exit /b %errorlevel%
) else (
    echo Compilation successful.
)

echo Script execution completed.

DEL /F /Q pipeline.zip