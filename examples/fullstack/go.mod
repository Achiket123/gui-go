module github.com/achiket123/taskflow

go 1.24.0

require (
	github.com/achiket123/gui-go v0.0.0
	github.com/go-sql-driver/mysql v1.8.1
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	golang.org/x/crypto v0.22.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	golang.org/x/image v0.36.0 // indirect
	golang.org/x/text v0.34.0 // indirect
)

// Local replace so the GUI library resolves from your workspace.
replace github.com/achiket123/gui-go => ../..
