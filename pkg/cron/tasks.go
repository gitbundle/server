// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package cron

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/gitbundle/modules/graceful"
	"github.com/gitbundle/modules/log"
	"github.com/gitbundle/modules/process"
	"github.com/gitbundle/modules/setting"
	"github.com/gitbundle/modules/translation"
	admin_model "github.com/gitbundle/server/pkg/admin"
	"github.com/gitbundle/server/pkg/db"
	user_model "github.com/gitbundle/server/pkg/user"
)

var (
	lock     = sync.Mutex{}
	started  = false
	tasks    = []*Task{}
	tasksMap = map[string]*Task{}
)

// Task represents a Cron task
type Task struct {
	lock        sync.Mutex
	Name        string
	config      Config
	fun         func(context.Context, *user_model.User, Config) error
	Status      string
	LastMessage string
	LastDoer    string
	ExecTimes   int64
}

// DoRunAtStart returns if this task should run at the start
func (t *Task) DoRunAtStart() bool {
	return t.config.DoRunAtStart()
}

// IsEnabled returns if this task is enabled as cron task
func (t *Task) IsEnabled() bool {
	return t.config.IsEnabled()
}

// GetConfig will return a copy of the task's config
func (t *Task) GetConfig() Config {
	if reflect.TypeOf(t.config).Kind() == reflect.Ptr {
		// Pointer:
		return reflect.New(reflect.ValueOf(t.config).Elem().Type()).Interface().(Config)
	}
	// Not pointer:
	return reflect.New(reflect.TypeOf(t.config)).Elem().Interface().(Config)
}

// Run will run the task incrementing the cron counter with no user defined
func (t *Task) Run() {
	t.RunWithUser(&user_model.User{
		ID:        -1,
		Name:      "(Cron)",
		LowerName: "(cron)",
	}, t.config)
}

// RunWithUser will run the task incrementing the cron counter at the time with User
func (t *Task) RunWithUser(doer *user_model.User, config Config) {
	if !taskStatusTable.StartIfNotRunning(t.Name) {
		return
	}
	t.lock.Lock()
	if config == nil {
		config = t.config
	}
	t.ExecTimes++
	t.lock.Unlock()
	defer func() {
		taskStatusTable.Stop(t.Name)
		if err := recover(); err != nil {
			// Recover a panic within the
			combinedErr := fmt.Errorf("%s\n%s", err, log.Stack(2))
			log.Error("PANIC whilst running task: %s Value: %v", t.Name, combinedErr)
		}
	}()
	graceful.GetManager().RunWithShutdownContext(func(baseCtx context.Context) {
		pm := process.GetManager()
		doerName := ""
		if doer != nil && doer.ID != -1 {
			doerName = doer.Name
		}

		ctx, _, finished := pm.AddContext(baseCtx, config.FormatMessage("en-US", t.Name, "process", doerName))
		defer finished()

		if err := t.fun(ctx, doer, config); err != nil {
			var message string
			var status string
			if db.IsErrCancelled(err) {
				status = "cancelled"
				message = err.(db.ErrCancelled).Message
			} else {
				status = "error"
				message = err.Error()
			}

			t.lock.Lock()
			t.LastMessage = message
			t.Status = status
			t.LastDoer = doerName
			t.lock.Unlock()

			if err := admin_model.CreateNotice(ctx, admin_model.NoticeTask, config.FormatMessage("en-US", t.Name, "cancelled", doerName, message)); err != nil {
				log.Error("CreateNotice: %v", err)
			}
			return
		}

		t.lock.Lock()
		t.Status = "finished"
		t.LastMessage = ""
		t.LastDoer = doerName
		t.lock.Unlock()

		if config.DoNoticeOnSuccess() {
			if err := admin_model.CreateNotice(ctx, admin_model.NoticeTask, config.FormatMessage("en-US", t.Name, "finished", doerName)); err != nil {
				log.Error("CreateNotice: %v", err)
			}
		}
	})
}

// GetTask gets the named task
func GetTask(name string) *Task {
	lock.Lock()
	defer lock.Unlock()
	log.Info("Getting %s in %v", name, tasksMap[name])

	return tasksMap[name]
}

// RegisterTask allows a task to be registered with the cron service
func RegisterTask(name string, config Config, fun func(context.Context, *user_model.User, Config) error) error {
	log.Debug("Registering task: %s", name)

	i18nKey := "admin.dashboard." + name
	if _, ok := translation.TryTr("en-US", i18nKey); !ok {
		// return fmt.Errorf("translation is missing for task %q, please add translation for %q", name, i18nKey)
		log.Error("translation is missing for task %q, please add translation for %q", name, i18nKey)
	}

	_, err := setting.GetCronSettings(name, config)
	if err != nil {
		log.Error("Unable to register cron task with name: %s Error: %v", name, err)
		return err
	}

	task := &Task{
		Name:   name,
		config: config,
		fun:    fun,
	}
	lock.Lock()
	locked := true
	defer func() {
		if locked {
			lock.Unlock()
		}
	}()
	if _, has := tasksMap[task.Name]; has {
		log.Error("A task with this name: %s has already been registered", name)
		return fmt.Errorf("duplicate task with name: %s", task.Name)
	}

	if config.IsEnabled() {
		// We cannot use the entry return as there is no way to lock it
		if _, err = c.AddJob(name, config.GetSchedule(), task); err != nil {
			log.Error("Unable to register cron task with name: %s Error: %v", name, err)
			return err
		}
	}

	tasks = append(tasks, task)
	tasksMap[task.Name] = task
	if started && config.IsEnabled() && config.DoRunAtStart() {
		lock.Unlock()
		locked = false
		task.Run()
	}

	return nil
}

// RegisterTaskFatal will register a task but if there is an error log.Fatal
func RegisterTaskFatal(name string, config Config, fun func(context.Context, *user_model.User, Config) error) {
	if err := RegisterTask(name, config, fun); err != nil {
		log.Fatal("Unable to register cron task %s Error: %v", name, err)
	}
}
