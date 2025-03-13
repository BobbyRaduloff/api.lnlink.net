#!/bin/bash

PACKAGES=(
    "api.codprotect.app/src/pkg/services/auth"
    "api.codprotect.app/src/pkg/services/email"
    "api.codprotect.app/src/pkg/services/files"
    "api.codprotect.app/src/pkg/services/calendar"
    "api.codprotect.app/src/pkg/models/product"
    "api.codprotect.app/src/pkg/models/users"
    "api.codprotect.app/src/pkg/models/events"
    "api.codprotect.app/src/pkg/models/reports"
    "api.codprotect.app/src/pkg/models/entity"
    "api.codprotect.app/src/pkg/models/competitor"
)

COVERPKG=$(IFS=, ; echo "${PACKAGES[*]}")
echo $COVERPKG
# Run tests and generate coverage
go test -coverpkg=$COVERPKG ./... -coverprofile=cover.out

# Generate HTML coverage report
go tool cover -html cover.out -o cover.html

# Open coverage report based on OS
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    open cover.html &
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # Linux
    if command -v firefox &> /dev/null; then
        firefox cover.html &
    else
        xdg-open cover.html &
    fi
fi
