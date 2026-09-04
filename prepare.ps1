cd eif

git ls-files 'scripts/*.sh' 'scripts/**/*.sh' |
while read -r f; do
  git update-index --chmod=+x "$f"
done

gofmt -w internal/core/middleware/requestid.go

git status
git diff --summary
git diff