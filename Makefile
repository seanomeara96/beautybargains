run: build serve
	 

build:
	npx tailwindcss -i ./assets/src/input.css -o ./assets/dist/output.css --minify && npx webpack --config webpack.config.js && go build -o bin/server.exe ./cmd/server 

serve:
	./bin/server -port 3000

scrape: scrape2 scrape3

scrape2:
	go run cmd/scrape/main.go -file true -wid 2 &

scrape3:
	go run cmd/scrape/main.go -file true -wid 3 &
