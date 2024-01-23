run:
	npx webpack --config webpack.config.js && go run ./cmd/server/ -port 3000


scrape: scrape2 scrape3

scrape2:
	go run cmd/scrape/main.go -file true -wid 2 &

scrape3:
	go run cmd/scrape/main.go -file true -wid 3 &
