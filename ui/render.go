package main

import (
	"errors"
	"strconv"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

// errStatus returns a simple error describing a non-2xx HTTP status line. Kept
// for parity with the networking layer; not currently used directly.
func errStatus(status string) error { return errors.New("request failed: " + status) }

// utoa converts a uint to a string for URL path segments.
func utoa(u uint) string { return strconv.FormatUint(uint64(u), 10) }

// Render builds the SPA. A single root component switches between the auth
// view and the tasks view based on whether a token is set.
func (t *TaskApp) Render() app.UI {
	children := []app.UI{
		app.H1().Text("Task Manager"),
	}
	if t.err != "" {
		children = append(children, app.Div().Class("err").Text(t.err))
	}
	if t.token == "" {
		children = append(children, t.renderAuth()...)
	} else {
		children = append(children, t.renderTasks()...)
	}
	return app.Div().Class("wrap").Body(children...)
}

// renderAuth returns the login + register forms side by side.
func (t *TaskApp) renderAuth() []app.UI {
	login := app.Div().Class("card").Body(
		app.H2().Text("Sign in"),
		app.Input().Type("email").Placeholder("Email").
			Value(t.loginEmail).OnChange(t.ValueTo(&t.loginEmail)),
		app.Input().Type("password").Placeholder("Password").
			Value(t.loginPassword).OnChange(t.ValueTo(&t.loginPassword)),
		app.Button().Text("Login").OnClick(t.login).Disabled(t.loading),
	)

	register := app.Div().Class("card").Body(
		app.H2().Text("Create an account"),
		app.Input().Type("text").Placeholder("Name").
			Value(t.regName).OnChange(t.ValueTo(&t.regName)),
		app.Input().Type("email").Placeholder("Email").
			Value(t.regEmail).OnChange(t.ValueTo(&t.regEmail)),
		app.Input().Type("password").Placeholder("Password (min 6)").
			Value(t.regPassword).OnChange(t.ValueTo(&t.regPassword)),
		app.Button().Text("Register").OnClick(t.register).Disabled(t.loading),
	)

	return []app.UI{
		app.Div().Class("row").Body(login, register),
	}
}

// renderTasks returns the authenticated view: logout header, create form, list.
func (t *TaskApp) renderTasks() []app.UI {
	header := app.Div().Class("row").Body(
		app.Span().Class("muted").Text("Signed in"),
		app.Button().Class("secondary").Text("Logout").OnClick(t.logout),
	)

	create := app.Div().Class("card").Body(
		app.H2().Text("New task"),
		app.Input().Type("text").Placeholder("Title").
			Value(t.newTitle).OnChange(t.ValueTo(&t.newTitle)),
		app.Input().Type("text").Placeholder("Description").
			Value(t.newDesc).OnChange(t.ValueTo(&t.newDesc)),
		statusSelect("status-new", t.newStatus, t.ValueTo(&t.newStatus)),
		app.Button().Text("Add").OnClick(t.create),
	)

	return []app.UI{
		header,
		create,
		t.renderTable(),
	}
}

// renderTable renders the list of tasks, with inline edit + delete per row.
func (t *TaskApp) renderTable() app.UI {
	if t.loading && len(t.tasks) == 0 {
		return app.Div().Class("muted").Text("Loading…")
	}
	if len(t.tasks) == 0 {
		return app.Div().Class("card muted").Text("No tasks yet — add one above.")
	}

	rows := make([]app.UI, 0, len(t.tasks))
	for _, task := range t.tasks {
		task := task // capture per-iteration for the closures below
		if t.editingID == task.ID {
			rows = append(rows, app.Tr().Body(
				app.Td().Body(app.Input().Type("text").
					Value(t.editingTitle).OnChange(t.ValueTo(&t.editingTitle))),
				app.Td().Body(statusSelect("status-edit", t.editingStatus, t.ValueTo(&t.editingStatus))),
				app.Td().Body(
					app.Button().Text("Save").OnClick(t.saveEdit),
					app.Button().Class("secondary").Text("Cancel").OnClick(t.cancelEdit),
				),
			))
			continue
		}

		rows = append(rows, app.Tr().Body(
			app.Td().Body(
				app.Div().Text(task.Title),
				app.If(task.Description != "",
					app.Div().Class("muted").Text(task.Description)),
			),
			app.Td().Body(statusBadge(task.Status)),
			app.Td().Body(
				app.Button().Class("secondary").Text("Edit").OnClick(func(ctx app.Context, _ app.Event) {
					t.editingID = task.ID
					t.editingTitle = task.Title
					t.editingStatus = task.Status
					t.Update()
				}),
				app.Button().Class("danger").Text("Delete").
					OnClick(func(ctx app.Context, _ app.Event) {
						t.remove(ctx, task.ID)
					}),
			),
		))
	}

	return app.Div().Class("card").Body(
		app.H2().Text("Your tasks"),
		app.Table().Body(
			app.THead().Body(app.Tr().Body(
				app.Th().Text("Task"),
				app.Th().Text("Status"),
				app.Th().Text("Actions"),
			)),
			app.TBody().Body(rows...),
		),
	)
}

// statusSelect renders a <select> with the three allowed status values.
func statusSelect(id, current string, onChange app.EventHandler) app.UI {
	return app.Select().ID(id).OnChange(onChange).Body(
		option("pending", "Pending", current == "pending" || current == ""),
		option("in_progress", "In progress", current == "in_progress"),
		option("done", "Done", current == "done"),
	)
}

func option(value, label string, selected bool) app.UI {
	o := app.Option().Value(value).Body(app.Text(label))
	if selected {
		o = o.Selected(true)
	}
	return o
}

func statusBadge(status string) app.UI {
	class := "badge"
	switch status {
	case "done":
		class += " done"
	case "in_progress":
		class += " in_progress"
	}
	return app.Span().Class(class).Text(prettyStatus(status))
}

func prettyStatus(s string) string {
	switch s {
	case "in_progress":
		return "in progress"
	case "":
		return "pending"
	}
	return s
}
