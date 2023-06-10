watch:
	# Compile and minify your CSS for production
	# The watcher will auto compile when changes are detected
	tailwindcss -i input.css -o style.css --minify --watch

windows:
	env GOOS=windows GOARCH=amd64 go build server.go

dev:
	light-server -s . -p 8080 \
		-w ui.js \
		-w style.css \
		-w index.html

install-server-env:
	sudo apt install make
	wget "https://go.dev/dl/$$(curl 'https://go.dev/VERSION?m=text').linux-amd64.tar.gz"
	sudo rm -rf /usr/local/go
	sudo tar --directory /usr/local -xzf go*.linux-amd64.tar.gz
	sudo ln -s /usr/local/go/bin/go /usr/local/bin/go
	sudo ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt


live:
	sudo pkill server || true
	sudo -b nohup go run server.go

# install cert here:
# https://certbot.eff.org/
renew-cert:
	sudo certbot certonly --webroot
