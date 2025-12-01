package errs

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
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

func parsePgError(detail string) (props []string, table string) {
	re := regexp.MustCompile(`Key \(([^)]+)\)=\(([^)]+)\)`)
	m := re.FindStringSubmatch(detail)
	if len(m) == 3 {
		fields := strings.Split(m[1], ", ")
		values := strings.Split(m[2], ", ")
		for i, field := range fields {
			v := ""
			if i < len(values) {
				v = values[i]
			}
			props = append(props, fmt.Sprintf("%s='%s'", field, v))
		}
	}

	re = regexp.MustCompile(`table "([^"]+)"`)
	m = re.FindStringSubmatch(detail)
	if len(m) == 2 {
		table = m[1]
	}

	return
}

func ConvertError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.ForeignKeyViolation:
			props, table := parsePgError(pgErr.Detail)
			var message string
			if strings.Contains(pgErr.Detail, "is not present in table") {
				message = fmt.Sprintf("{%s} does not reference an existing record in '%s'", strings.Join(props, ","), table)
			} else {
				message = pgErr.Detail
			}
			return NewDBError(errors.New("foreign key violation: " + message))
		case pgerrcode.UniqueViolation:
			props, _ := parsePgError(pgErr.Detail)
			var message string
			if strings.Contains(pgErr.Detail, "already exists") {
				message = fmt.Sprintf("{%s} already exists", strings.Join(props, ","))
			} else {
				message = pgErr.Detail
			}
			return NewDBError(errors.New("unique constraint violation: " + message))
		}
	}

	return err
}
