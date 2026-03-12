# GORM SQLite Driver

[![Go Reference](https://pkg.go.dev/badge/image)](https://pkg.go.dev/github.com/ncruces/go-sqlite3/gormlite)

## Usage

```go
import (
	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
)

db, err := gorm.Open(gormlite.Open("gorm.db"), &gorm.Config{})
```

Checkout [https://gorm.io](https://gorm.io) for details.