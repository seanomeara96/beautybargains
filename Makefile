dev: build quickserve

run: build serve

deploy: style build serve

style:
	npx tailwindcss -i ./assets/src/input.css -o ./assets/dist/output.css --minify && npx webpack --config webpack.config.js
	 
build:
	go build -o bin/server.exe ./cmd/server 

serve:
	./bin/server.exe -port 3000

quickserve:
	./bin/server.exe -port 3000 -mode dev --skip

scrape: scrape2 scrape3

scrape2:
	go run cmd/scrape/main.go -file true -wid 2 &

scrape3:
	go run cmd/scrape/main.go -file true -wid 3 &
