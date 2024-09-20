package errs

import (
	"errors"
	"fmt"
	"github.com/lib/pq"
	"regexp"
	"strings"
)

type DBError struct {
	Err error
}

func (e *DBError) Error() string {
	return e.Err.Error()
}

func NewDBError(err error) *DBError {
	return &DBError{Err: err}
}

func ConvertError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pq.Error
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		re := regexp.MustCompile(`\([^()]+\)`)
		matches := re.FindAllString(pgErr.Detail, -1)
		var strs []string
		for i := 0; i < len(matches); i = i + 2 {
			strs = append(strs, fmt.Sprintf("%s=%s", matches[i], matches[i+1]))
		}
		return NewDBError(errors.New("unique constraint violation: " + strings.Join(strs, ", ")))
	}

	return err
}
