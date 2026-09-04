cd eif

git ls-files 'scripts/*.sh' 'scripts/**/*.sh' |
while read -r f; do
  git update-index --chmod=+x "$f"
done

gofmt -w internal/core/middleware/requestid.go

git status
git diff --summary
git diff


#

git add internal/core/middleware/requestid.go
git commit -m "fix(ci): mark scripts executable and format Go"
git push

#

go get golang.org/x/crypto@v0.56.0
go mod tidy

#

go test ./...
go vet ./...
go mod verify

#

govulncheck ./...

# Hoặc

go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...

git diff go.mod go.sum

#

git add go.mod go.sum
git commit -m "fix(security): update golang.org/x/crypto"
git push

#

go mod edit -go=1.26.8
go get github.com/quic-go/quic-go@v0.59.1
go mod tidy