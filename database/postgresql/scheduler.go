package postgresql

import (
	"database/sql"
	"fmt"

	"github.com/staticbackendhq/core/model"
)

func (pg *PostgreSQL) ListTasks() (results []model.Task, err error) {
	bases, err := pg.ListDatabases()
	if err != nil {
		return
	}

	for _, base := range bases {
		tasks, err := pg.ListTasksByBase(base.Name)
		if err != nil {
			return results, err
		}

		results = append(results, tasks...)
	}

	return
}

func (pg *PostgreSQL) ListTasksByBase(dbName string) (results []model.Task, err error) {
	qry := fmt.Sprintf(`
		SELECT * 
		FROM %s.sb_tasks 
	`, dbName)

	rows, err := pg.DB.Query(qry)
	if err != nil {
		return
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var t model.Task
		if err = scanTask(rows, &t); err != nil {
			return
		}
		t.BaseName = dbName

		results = append(results, t)
	}

	err = rows.Err()
	return
}

func (pg *PostgreSQL) GetTask(dbName, id string) (task model.Task, err error) {
	qry := fmt.Sprintf(`
		SELECT *
		FROM %s.sb_tasks
		WHERE id = $1
	`, dbName)

	err = scanTask(pg.DB.QueryRow(qry, id), &task)
	if err == sql.ErrNoRows {
		err = fmt.Errorf("task not found: %s", id)
	}
	task.BaseName = dbName
	return
}

func (pg *PostgreSQL) AddTask(dbName string, task model.Task) (id string, err error) {
	qry := fmt.Sprintf(`
	INSERT INTO %s.sb_tasks(id, name, type, value, meta, interval, last_run)
	VALUES($1, $2, $3, $4, $5, $6, $7);
	`, dbName)

	id = pg.NewID()

	_, err = pg.DB.Exec(
		qry,
		id,
		task.Name,
		task.Type,
		task.Value,
		task.Meta,
		task.Interval,
		task.LastRun,
	)
	return
}

func (pg *PostgreSQL) UpdateTask(dbName string, task model.Task) error {
	qry := fmt.Sprintf(`
	UPDATE %s.sb_tasks
	SET name = $2, type = $3, value = $4, meta = $5, interval = $6, last_run = $7
	WHERE id = $1;
	`, dbName)

	_, err := pg.DB.Exec(
		qry,
		task.ID,
		task.Name,
		task.Type,
		task.Value,
		task.Meta,
		task.Interval,
		task.LastRun,
	)
	return err
}

func (sl *PostgreSQL) DeleteTask(dbName, id string) error {
	qry := fmt.Sprintf(`
	DELETE FROM %s.sb_tasks
	WHERE id = $1;
	`, dbName)

	if _, err := sl.DB.Exec(qry, id); err != nil {
		return err
	}
	return nil
}

func scanTask(rows Scanner, t *model.Task) error {
	return rows.Scan(
		&t.ID,
		&t.Name,
		&t.Type,
		&t.Value,
		&t.Meta,
		&t.Interval,
		&t.LastRun,
	)
}
