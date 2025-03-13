#!/bin/bash

PACKAGES=(
    # "api.codprotect.app/src/pkg/services/email"
)

COVERPKG=$(IFS=, ; echo "${PACKAGES[*]}")
echo $COVERPKG
# Run tests and generate coverage
go test -coverpkg=$COVERPKG ./... -coverprofile=cover.out

# clean up coverage file to remove all the lines that are not covered
go-ignore-cov --file cover.out

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
