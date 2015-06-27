
dcounter:
	go build

install: dcounter
	mkdir -p /var/lib/dcounter/ 
	cp dcounter /usr/local/bin/dcounter
	cp etc/init.d/dcounter /etc/init.d/dcounter


test:
	go test ./...

