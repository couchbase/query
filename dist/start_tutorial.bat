@echo off

pushd "%~dp0"
echo.
echo In your web browser open http://localhost:8093/tutorial/
echo.

start "cbq" "http://localhost:8093/tutorial/"

cbq-engine -couchbase dir:data

popd
