package memory

import (
	"errors"
	"fmt"

	"github.com/staticbackendhq/core/model"
)

func (m *Memory) ListTasks() (results []model.Task, err error) {
	bases, err := m.ListDatabases()
	if err != nil {
		return
	}

	for _, base := range bases {
		tasks, err := m.ListTasksByBase(base.Name)
		if err != nil {
			return nil, err
		}

		results = append(results, tasks...)
	}

	return
}

func (m *Memory) ListTasksByBase(dbName string) ([]model.Task, error) {
	tasks, err := all[model.Task](m, dbName, "sb_tasks")
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		tasks[i].BaseName = dbName
	}
	return tasks, nil
}

func (m *Memory) GetTask(dbName, id string) (model.Task, error) {
	var task model.Task
	if err := getByID(m, dbName, "sb_tasks", id, &task); err != nil {
		return model.Task{}, err
	}
	task.BaseName = dbName
	return task, nil
}

func (m *Memory) AddTask(dbName string, task model.Task) (id string, err error) {
	id = m.NewID()
	task.ID = id

	err = create(m, dbName, "sb_tasks", id, task)
	return
}

func (m *Memory) UpdateTask(dbName string, task model.Task) error {
	key := fmt.Sprintf("%s_sb_tasks", dbName)
	tasks, ok := m.DB[key]
	if !ok {
		return errors.New("cannot find repo")
	}
	if _, ok := tasks[task.ID]; !ok {
		return errors.New("task not found")
	}

	mx.Lock()
	tasks[task.ID] = mustEnc(task)
	m.DB[key] = tasks
	mx.Unlock()
	return nil
}

func (m *Memory) DeleteTask(dbName, id string) error {
	key := fmt.Sprintf("%s_sb_tasks", dbName)
	tasks, ok := m.DB[key]
	if !ok {
		return errors.New("cannot find repo")
	}

	delete(tasks, id)

	mx.Lock()
	m.DB[key] = tasks
	mx.Unlock()
	return nil
}
