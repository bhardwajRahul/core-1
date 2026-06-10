package postgresql

import (
	"fmt"
	"strings"

	"github.com/staticbackendhq/core/model"
)

func (pg *PostgreSQL) AddFile(dbName string, f model.File) (id string, err error) {
	qry := fmt.Sprintf(`
		INSERT INTO %s.sb_files(account_id, key, url, size, uploaded)
		VALUES($1, $2, $3, $4, $5)
		RETURNING id;
	`, dbName)

	err = pg.DB.QueryRow(
		qry,
		f.AccountID,
		f.Key,
		f.URL,
		f.Size,
		f.Uploaded,
	).Scan(&id)
	return
}

func (pg *PostgreSQL) GetFileByID(dbName, fileID string) (f model.File, err error) {
	qry := fmt.Sprintf(`
		SELECT * 
		FROM %s.sb_files 
		WHERE id = $1
	`, dbName)

	row := pg.DB.QueryRow(qry, fileID)

	err = scanFile(row, &f)
	return

}

func (pg *PostgreSQL) DeleteFile(dbName, fileID string) error {
	qry := fmt.Sprintf(`
		DELETE FROM %s.sb_files 
		WHERE id = $1
	`, dbName)

	if _, err := pg.DB.Exec(qry, fileID); err != nil {
		return err
	}
	return nil
}

func (pg *PostgreSQL) ListAllFiles(dbName, accountID string) (results []model.File, err error) {
	where := "WHERE account_id = $1"

	// if no accountID is specify, the admin UI
	// display all files uploaded.
	if len(accountID) == 0 {
		where = "WHERE $1 = $1"
	}

	qry := fmt.Sprintf(`
		SELECT * 
		FROM %s.sb_files
		%s
	`, dbName, where)

	rows, err := pg.DB.Query(qry, accountID)
	if err != nil {
		return
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var f model.File
		if err = scanFile(rows, &f); err != nil {
			return
		}

		results = append(results, f)
	}

	err = rows.Err()

	return
}

func (pg *PostgreSQL) GetTotalFileBytes(dbName, accountID string) (total int64, err error) {
	qry := fmt.Sprintf(`
		SELECT COALESCE(SUM(size), 0)
		FROM %s.sb_files
		WHERE account_id = $1
	`, dbName)

	err = pg.DB.QueryRow(qry, accountID).Scan(&total)
	if err != nil && !isTableExists(err) {
		return 0, nil
	}

	return
}

func (pg *PostgreSQL) ListFiles(dbName, accountID string, params model.ListParams) (results []model.File, total int64, err error) {
	orderBy := fileOrderBy(params)
	offset := (params.Page - 1) * params.Size

	qry := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM %s.sb_files
		WHERE account_id = $1
	`, dbName)

	if err = pg.DB.QueryRow(qry, accountID).Scan(&total); err != nil {
		if !isTableExists(err) {
			return []model.File{}, 0, nil
		}
		return nil, 0, err
	}

	qry = fmt.Sprintf(`
		SELECT *
		FROM %s.sb_files
		WHERE account_id = $1
		ORDER BY %s
		LIMIT $2 OFFSET $3
	`, dbName, orderBy)

	rows, err := pg.DB.Query(qry, accountID, params.Size, offset)
	if err != nil {
		if !isTableExists(err) {
			return []model.File{}, total, nil
		}
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var f model.File
		if err = scanFile(rows, &f); err != nil {
			return nil, 0, err
		}

		results = append(results, f)
	}

	if results == nil {
		results = []model.File{}
	}

	err = rows.Err()
	return
}

func fileOrderBy(params model.ListParams) string {
	field := "uploaded"
	if strings.EqualFold(params.SortBy, "size") {
		field = "size"
	}

	direction := "ASC"
	if params.SortDescending {
		direction = "DESC"
	}

	return fmt.Sprintf("%s %s", field, direction)
}

func scanFile(rows Scanner, f *model.File) error {
	return rows.Scan(
		&f.ID,
		&f.AccountID,
		&f.Key,
		&f.URL,
		&f.Size,
		&f.Uploaded,
	)
}
