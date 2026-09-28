.PHONY: api web test

api:
	cd apps/api && go run ./cmd/opspilot

web:
	cd apps/web && npm run dev

test:
	cd apps/api && gofmt -l . && go test ./... && go build -o /tmp/opspilot ./cmd/opspilot
	cd apps/web && npm test && npm run lint && npm run typecheck
