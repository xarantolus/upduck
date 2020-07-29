mkdir -p releases
rm releases/*

PROGRAM_NAME="upduck"

GOOS=windows go build -o "releases/$PROGRAM_NAME-windows.exe"

GOOS=linux go build -o "releases/$PROGRAM_NAME-linux"

GOOS=linux GOARCH=arm GOARM=5 go build -o "releases/$PROGRAM_NAME-raspberrypi-arm5"
GOOS=linux GOARCH=arm GOARM=6 go build -o "releases/$PROGRAM_NAME-raspberrypi-arm6"
GOOS=linux GOARCH=arm GOARM=7 go build -o "releases/$PROGRAM_NAME-raspberrypi-arm7"

