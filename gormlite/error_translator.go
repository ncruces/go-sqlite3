package gormlite

import (
	"errors"

	"github.com/ncruces/go-sqlite3"
	"gorm.io/gorm"
)

func (dialector Dialector) Translate(err error) error {
	if errors.Is(err, sqlite3.CONSTRAINT_UNIQUE) {
		return gorm.ErrDuplicatedKey
	}
	return err
}
