.PHONY: dev-backend dev-frontend test build

dev-backend:
	cd backend && go run .

dev-frontend:
	cd frontend && npm run dev

test:
	cd backend && go test ./...
	cd frontend && npm run type-check

build:
	cd backend && go build ./...
	cd frontend && npm run build
