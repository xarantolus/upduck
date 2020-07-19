mkdir -p releases
rm releases/*

PROGRAM_NAME="upduck"

GOOS=windows go build -o "releases/$PROGRAM_NAME-windows.exe"

GOOS=linux go build -o "releases/$PROGRAM_NAME-linux"

GOOS=linux GOARCH=arm GOARM=7 go build -o "releases/$PROGRAM_NAME-raspberrypi"