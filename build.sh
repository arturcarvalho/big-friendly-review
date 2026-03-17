go build -o bfr -ldflags "-X main.BuildTime=$(date +%Y-%m-%dT%H:%M)" 2>&1
mv bfr /usr/local/bin/
