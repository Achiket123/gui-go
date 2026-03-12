package screens

import (
	"context"
	"fmt"

	"github.com/achiket123/gui-go/canvas"
	"github.com/achiket123/gui-go/ui"
	"github.com/achiket123/taskflow/internal/models"
	"github.com/achiket123/taskflow/ui/state"
	"github.com/achiket123/taskflow/ui/styles"
)

// ═══════════════════════════════════════════════════════════════════════════════
// ProjectBoard — Kanban columns
// ═══════════════════════════════════════════════════════════════════════════════

var kanbanColumns = []models.TaskStatus{
	models.StatusBacklog,
	models.StatusTodo,
	models.StatusInProgress,
	models.StatusInReview,
	models.StatusDone,
}

type ProjectBoard struct {
	app         *state.App
	projectID   string
	workspaceID string
	tasks       []models.Task
	loading     bool
	error       string
	bounds      canvas.Rect

	// Per-column virtual lists.
	colScrolls []*ui.ScrollView
	colTasks   [][]models.Task

	// Header buttons.
	addBtn     *ui.Button
	refreshBtn *ui.Button

	// Drawer.
	detailDrawer *TaskDetailDrawer
	showDrawer   bool
}

func NewProjectBoard(app *state.App, projectID, workspaceID string) *ProjectBoard {
	b := &ProjectBoard{
		app:         app,
		projectID:   projectID,
		workspaceID: workspaceID,
	}

	b.addBtn = ui.NewButton("+ New Task", func() {
		b.detailDrawer = NewTaskDetailDrawer(b.app, nil, projectID, workspaceID, func() {
			b.showDrawer = false
			b.reload()
		})
		b.showDrawer = true
	})
	b.addBtn.Style = styles.PrimaryButtonStyle()

	b.refreshBtn = ui.NewButton("⟳", func() { b.reload() })
	b.refreshBtn.Style = styles.SecondaryButtonStyle()

	// One scroll view per column.
	b.colScrolls = make([]*ui.ScrollView, len(kanbanColumns))
	b.colTasks = make([][]models.Task, len(kanbanColumns))
	for i := range b.colScrolls {
		ci := i
		b.colScrolls[ci] = ui.NewScrollView(2000, func(c *canvas.Canvas, x, y, w, _ float32) {
			b.drawColumn(c, ci, x, y, w)
		})
	}

	b.reload()
	return b
}

func (b *ProjectBoard) reload() {
	b.loading = true
	b.error = ""
	go func() {
		result, err := b.app.API.ListTasks(context.Background(), b.projectID, nil)
		b.loading = false
		if err != nil {
			b.error = err.Error()
			return
		}
		b.tasks = result.Items
		b.buildColumns()
	}()
}

func (b *ProjectBoard) buildColumns() {
	for i := range b.colTasks {
		b.colTasks[i] = nil
	}
	for _, t := range b.tasks {
		for i, col := range kanbanColumns {
			if t.Status == col {
				b.colTasks[i] = append(b.colTasks[i], t)
				break
			}
		}
	}
	// Resize scroll heights.
	for i, sv := range b.colScrolls {
		n := len(b.colTasks[i])
		sv.ContentH = float32(n)*110 + 20
	}
}

func (b *ProjectBoard) Bounds() canvas.Rect { return b.bounds }
func (b *ProjectBoard) Tick(d float64) {
	b.addBtn.Tick(d)
	b.refreshBtn.Tick(d)
	for _, sv := range b.colScrolls {
		sv.Tick(d)
	}
	if b.detailDrawer != nil {
		b.detailDrawer.Tick(d)
	}
}
func (b *ProjectBoard) HandleEvent(e ui.Event) bool {
	if b.showDrawer && b.detailDrawer != nil {
		return b.detailDrawer.HandleEvent(e)
	}
	if b.addBtn.HandleEvent(e) || b.refreshBtn.HandleEvent(e) {
		return true
	}
	for _, sv := range b.colScrolls {
		if sv.HandleEvent(e) {
			return true
		}
	}
	return false
}

func (b *ProjectBoard) Draw(c *canvas.Canvas, x, y, w, h float32) {
	b.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	c.DrawRect(x, y, w, h, canvas.FillPaint(styles.BgBase))

	// Header bar.
	hdrH := float32(56)
	c.DrawRect(x, y, w, hdrH, canvas.FillPaint(styles.BgSurface))
	c.DrawRect(x, y+hdrH, w, 1, canvas.FillPaint(styles.BorderSubtle))

	proj := b.app.ActiveProject.Get()
	title := "Project Board"
	if proj != nil {
		title = proj.Name
		accentCol := canvas.Hex(proj.Color)
		c.DrawCircle(x+22, y+hdrH/2, 7, canvas.FillPaint(accentCol))
		c.DrawText(x+36, y+34, title, styles.H3)
	} else {
		c.DrawText(x+20, y+34, title, styles.H3)
	}

	b.addBtn.Draw(c, x+w-160, y+10, 140, 36)
	b.refreshBtn.Draw(c, x+w-196, y+10, 32, 36)

	if b.loading {
		c.DrawCenteredText(canvas.Rect{X: x, Y: y + hdrH, W: w, H: h - hdrH}, "Loading tasks…", styles.Label)
		return
	}
	if b.error != "" {
		errStyle := canvas.TextStyle{Color: styles.ColCancelled, Size: 13}
		c.DrawCenteredText(canvas.Rect{X: x, Y: y + hdrH, W: w, H: h - hdrH}, b.error, errStyle)
		return
	}

	// Kanban columns.
	colW := float32(220)
	gap := float32(12)
	totalW := float32(len(kanbanColumns))*(colW+gap) - gap
	startX := x + 16
	if totalW < w-32 {
		startX = x + (w-totalW)/2
	}

	for i, status := range kanbanColumns {
		cx := startX + float32(i)*(colW+gap)
		cy := y + hdrH + 12

		// Column header.
		statusCol := styles.StatusColor(string(status))
		c.DrawRoundedRect(cx, cy, colW, h-hdrH-24, styles.RadiusMD, canvas.FillPaint(styles.BgCard))
		c.DrawRoundedRect(cx, cy, colW, 38, styles.RadiusMD,
			canvas.FillPaint(canvas.Color{R: statusCol.R, G: statusCol.G, B: statusCol.B, A: 0.12}))
		c.DrawRect(cx, cy+26, colW, 12, canvas.FillPaint(styles.BgCard))

		label := models.TaskStatusLabel[status]
		c.DrawText(cx+12, cy+24, label, canvas.TextStyle{Color: statusCol, Size: 12})
		countStr := fmt.Sprintf("%d", len(b.colTasks[i]))
		c.DrawText(cx+colW-c.MeasureText(countStr, styles.Tiny).W-10, cy+24, countStr, styles.Tiny)

		// Scrollable task list.
		b.colScrolls[i].Draw(c, cx+4, cy+40, colW-8, h-hdrH-70)
	}

	// Task detail drawer overlay.
	if b.showDrawer && b.detailDrawer != nil {
		// Dim background.
		c.DrawRect(x, y, w, h, canvas.FillPaint(canvas.Color{A: 0.5}))
		drawerW := float32(500)
		if drawerW > w-80 {
			drawerW = w - 80
		}
		b.detailDrawer.Draw(c, x+w-drawerW, y, drawerW, h)
	}
}

// drawColumn draws the task cards for one column index.
func (b *ProjectBoard) drawColumn(c *canvas.Canvas, colIdx int, x, y, w float32) {
	tasks := b.colTasks[colIdx]
	for i, t := range tasks {
		ty := y + float32(i)*110 + 8
		b.drawTaskCard(c, t, x, ty, w-8)
	}
}

func (b *ProjectBoard) drawTaskCard(c *canvas.Canvas, t models.Task, x, y, w float32) {
	cardH := float32(96)
	c.DrawRoundedRect(x, y, w, cardH, styles.RadiusSM, canvas.FillPaint(styles.BgSurface))
	c.DrawRoundedRect(x, y, w, cardH, styles.RadiusSM, canvas.StrokePaint(styles.BorderSubtle, 1))

	// Priority dot.
	priColor := styles.PriorityColor(string(t.Priority))
	c.DrawCircle(x+12, y+14, 4, canvas.FillPaint(priColor))

	// Title (truncated).
	title := t.Title
	if len(title) > 40 {
		title = title[:37] + "…"
	}
	c.DrawText(x+22, y+18, title, styles.Body)

	// Description snippet.
	if t.Description != "" {
		desc := t.Description
		if len(desc) > 55 {
			desc = desc[:52] + "…"
		}
		c.DrawText(x+10, y+36, desc, styles.Small)
	}

	// Bottom row: priority badge + assignee.
	priLabel := models.TaskPriorityLabel[t.Priority]
	styles.DrawBadge(c, priLabel, canvas.Color{R: priColor.R, G: priColor.G, B: priColor.B, A: 0.2}, x+10, y+cardH-26)

	if t.Assignee != nil {
		styles.DrawAvatar(c, t.Assignee.DisplayName, styles.AvatarColor(t.Assignee.DisplayName), x+w-28, y+cardH-28, 20)
	}

	// Tap area — open detail drawer.
	// (handled via HandleEvent hit-test on the scroll view bounds)
}

// ═══════════════════════════════════════════════════════════════════════════════
// TaskDetailDrawer — slide-in panel to view/create/edit a task
// ═══════════════════════════════════════════════════════════════════════════════

type TaskDetailDrawer struct {
	app         *state.App
	task        *models.Task // nil = new task
	projectID   string
	workspaceID string
	onDone      func()
	bounds      canvas.Rect

	titleInput   *ui.TextInput
	descInput    *ui.TextInput
	statusDrop   *ui.Dropdown
	priorityDrop *ui.Dropdown
	saveBtn      *ui.Button
	deleteBtn    *ui.Button
	closeBtn     *ui.Button

	comments      []models.Comment
	commentInput  *ui.TextInput
	addCommentBtn *ui.Button

	loading    bool
	errorLabel string
}

func NewTaskDetailDrawer(app *state.App, task *models.Task, projectID, workspaceID string, onDone func()) *TaskDetailDrawer {
	d := &TaskDetailDrawer{
		app: app, task: task,
		projectID: projectID, workspaceID: workspaceID, onDone: onDone,
	}

	d.titleInput = ui.NewTextInput("")
	d.titleInput.Hint = "Task title…"

	d.descInput = ui.NewTextInput("")
	d.descInput.Hint = "Description (optional)"

	statuses := []string{"Backlog", "To Do", "In Progress", "In Review", "Done", "Cancelled"}
	d.statusDrop = ui.NewDropdown(statuses, ui.DefaultDropdownStyle())
	d.statusDrop.Placeholder = "Status"

	priorities := []string{"Urgent", "High", "Medium", "Low", "None"}
	d.priorityDrop = ui.NewDropdown(priorities, ui.DefaultDropdownStyle())
	d.priorityDrop.Placeholder = "Priority"
	d.priorityDrop.Selected = 2 // Medium

	d.saveBtn = ui.NewButton("Save Task", func() { d.doSave() })
	d.saveBtn.Style = styles.PrimaryButtonStyle()

	d.deleteBtn = ui.NewButton("Delete", func() { d.doDelete() })
	d.deleteBtn.Style = styles.DangerButtonStyle()

	d.closeBtn = ui.NewButton("✕", func() {
		if d.onDone != nil {
			d.onDone()
		}
	})
	d.closeBtn.Style = styles.SecondaryButtonStyle()

	d.commentInput = ui.NewTextInput("")
	d.commentInput.Hint = "Add a comment…"

	d.addCommentBtn = ui.NewButton("Post", func() { d.doAddComment() })
	d.addCommentBtn.Style = styles.PrimaryButtonStyle()

	// Pre-fill if editing.
	if task != nil {
		d.titleInput.Text = task.Title
		d.descInput.Text = task.Description
		d.loadComments()
	}

	return d
}

func (d *TaskDetailDrawer) loadComments() {
	if d.task == nil {
		return
	}
	go func() {
		cmts, err := d.app.API.ListComments(context.Background(), d.task.ID)
		if err == nil {
			d.comments = cmts
		}
	}()
}

func (d *TaskDetailDrawer) Bounds() canvas.Rect { return d.bounds }
func (d *TaskDetailDrawer) Tick(delta float64) {
	d.titleInput.Tick(delta)
	d.descInput.Tick(delta)
	d.statusDrop.Tick(delta)
	d.priorityDrop.Tick(delta)
	d.saveBtn.Tick(delta)
	d.deleteBtn.Tick(delta)
	d.closeBtn.Tick(delta)
	d.commentInput.Tick(delta)
	d.addCommentBtn.Tick(delta)
}
func (d *TaskDetailDrawer) HandleEvent(e ui.Event) bool {
	return d.titleInput.HandleEvent(e) || d.descInput.HandleEvent(e) ||
		d.statusDrop.HandleEvent(e) || d.priorityDrop.HandleEvent(e) ||
		d.saveBtn.HandleEvent(e) || d.deleteBtn.HandleEvent(e) ||
		d.closeBtn.HandleEvent(e) || d.commentInput.HandleEvent(e) ||
		d.addCommentBtn.HandleEvent(e)
}

func (d *TaskDetailDrawer) Draw(c *canvas.Canvas, x, y, w, h float32) {
	d.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	c.DrawRoundedRect(x, y, w, h, styles.RadiusLG, canvas.FillPaint(styles.BgSurface))
	c.DrawRoundedRect(x, y, w, h, styles.RadiusLG, canvas.StrokePaint(styles.BorderNormal, 1))

	heading := "New Task"
	if d.task != nil {
		heading = "Edit Task"
	}
	c.DrawText(x+20, y+36, heading, styles.H2)
	d.closeBtn.Draw(c, x+w-46, y+10, 32, 32)

	c.DrawRect(x+12, y+50, w-24, 1, canvas.FillPaint(styles.BorderSubtle))

	fw := w - 40
	d.titleInput.Draw(c, x+20, y+64, fw, 42)
	d.descInput.Draw(c, x+20, y+116, fw, 42)

	c.DrawText(x+20, y+170, "Status", styles.Label)
	d.statusDrop.Draw(c, x+20, y+186, fw/2-6, 38)

	c.DrawText(x+20+fw/2+6, y+170, "Priority", styles.Label)
	d.priorityDrop.Draw(c, x+20+fw/2+6, y+186, fw/2-6, 38)

	if d.errorLabel != "" {
		c.DrawText(x+20, y+238, d.errorLabel, canvas.TextStyle{Color: styles.ColCancelled, Size: 12})
	}

	d.saveBtn.Draw(c, x+20, y+252, fw, 40)
	if d.task != nil {
		d.deleteBtn.Draw(c, x+20, y+302, fw, 36)
	}

	// Comments section.
	c.DrawRect(x+12, y+354, w-24, 1, canvas.FillPaint(styles.BorderSubtle))
	c.DrawText(x+20, y+366, "Comments", styles.H3)

	cy := y + 390.0
	for _, cmt := range d.comments {
		name := "Unknown"
		if cmt.Author != nil {
			name = cmt.Author.DisplayName
		}
		styles.DrawAvatar(c, name, styles.AvatarColor(name), x+20, cy, 26)
		c.DrawText(x+52, cy+14, name, styles.Label)
		c.DrawText(x+20, cy+34, cmt.Body, styles.Body)
		c.DrawRect(x+20, cy+54, w-40, 1, canvas.FillPaint(styles.BorderSubtle))
		cy += 64
	}

	d.commentInput.Draw(c, x+20, cy+8, fw-72, 36)
	d.addCommentBtn.Draw(c, x+20+fw-68, cy+8, 64, 36)
}

func (d *TaskDetailDrawer) doSave() {
	title := d.titleInput.Text
	if title == "" {
		d.errorLabel = "Title is required."
		return
	}

	statusMap := []models.TaskStatus{
		models.StatusBacklog, models.StatusTodo, models.StatusInProgress,
		models.StatusInReview, models.StatusDone, models.StatusCancelled,
	}
	priorityMap := []models.TaskPriority{
		models.PriorityUrgent, models.PriorityHigh, models.PriorityMedium,
		models.PriorityLow, models.PriorityNone,
	}

	statusIdx := d.statusDrop.Selected
	if statusIdx < 0 {
		statusIdx = 0
	}
	priorityIdx := d.priorityDrop.Selected
	if priorityIdx < 0 {
		priorityIdx = 2
	}

	d.loading = true
	d.errorLabel = ""
	go func() {
		_, err := d.app.API.CreateTask(
			context.Background(),
			d.projectID, d.workspaceID,
			title, d.descInput.Text,
			statusMap[statusIdx], priorityMap[priorityIdx], "",
		)
		d.loading = false
		if err != nil {
			d.errorLabel = err.Error()
			return
		}
		if d.onDone != nil {
			d.onDone()
		}
	}()
}

func (d *TaskDetailDrawer) doDelete() {
	if d.task == nil {
		return
	}
	go func() {
		_ = d.app.API.DeleteTask(context.Background(), d.task.ID, d.workspaceID)
		if d.onDone != nil {
			d.onDone()
		}
	}()
}

func (d *TaskDetailDrawer) doAddComment() {
	body := d.commentInput.Text
	if body == "" {
		return
	}
	taskID := ""
	if d.task != nil {
		taskID = d.task.ID
	}
	if taskID == "" {
		return
	}
	go func() {
		cmt, err := d.app.API.AddComment(context.Background(), taskID, d.workspaceID, body)
		if err == nil {
			d.comments = append(d.comments, *cmt)
			d.commentInput.Text = ""
		}
	}()
}

// ═══════════════════════════════════════════════════════════════════════════════
// CreateProjectPanel
// ═══════════════════════════════════════════════════════════════════════════════

type CreateProjectPanel struct {
	app       *state.App
	nameInput *ui.TextInput
	descInput *ui.TextInput
	colorDrop *ui.Dropdown
	createBtn *ui.Button
	cancelBtn *ui.Button
	onDone    func()
	error     string
	loading   bool
	bounds    canvas.Rect
}

var projectColors = []string{
	"#6366F1", "#8B5CF6", "#EC4899", "#F59E0B",
	"#10B981", "#3B82F6", "#F97316", "#14B8A6",
}

func NewCreateProjectPanel(app *state.App, onDone func()) *CreateProjectPanel {
	p := &CreateProjectPanel{app: app, onDone: onDone}

	p.nameInput = ui.NewTextInput("")
	p.nameInput.Hint = "Project name"

	p.descInput = ui.NewTextInput("")
	p.descInput.Hint = "Description (optional)"

	colorNames := []string{"Indigo", "Purple", "Pink", "Amber", "Green", "Blue", "Orange", "Teal"}
	p.colorDrop = ui.NewDropdown(colorNames, ui.DefaultDropdownStyle())
	p.colorDrop.Selected = 0
	p.colorDrop.Placeholder = "Accent color"

	p.createBtn = ui.NewButton("Create Project", func() { p.doCreate() })
	p.createBtn.Style = styles.PrimaryButtonStyle()

	p.cancelBtn = ui.NewButton("Cancel", func() {
		if p.onDone != nil {
			p.onDone()
		}
	})
	p.cancelBtn.Style = styles.SecondaryButtonStyle()

	return p
}

func (p *CreateProjectPanel) Bounds() canvas.Rect { return p.bounds }
func (p *CreateProjectPanel) Tick(d float64) {
	p.nameInput.Tick(d)
	p.descInput.Tick(d)
	p.colorDrop.Tick(d)
	p.createBtn.Tick(d)
	p.cancelBtn.Tick(d)
}
func (p *CreateProjectPanel) HandleEvent(e ui.Event) bool {
	return p.nameInput.HandleEvent(e) || p.descInput.HandleEvent(e) ||
		p.colorDrop.HandleEvent(e) || p.createBtn.HandleEvent(e) || p.cancelBtn.HandleEvent(e)
}
func (p *CreateProjectPanel) Draw(c *canvas.Canvas, x, y, w, h float32) {
	p.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	c.DrawRect(x, y, w, h, canvas.FillPaint(styles.BgBase))

	cw := float32(420)
	cx := x + (w-cw)/2
	cy := y + 60.0
	styles.DrawCard(c, cx, cy, cw, 360, nil)

	c.DrawText(cx+24, cy+36, "New Project", styles.H2)
	fw := cw - 48
	p.nameInput.Draw(c, cx+24, cy+68, fw, 42)
	p.descInput.Draw(c, cx+24, cy+120, fw, 42)

	c.DrawText(cx+24, cy+174, "Accent Color", styles.Label)
	p.colorDrop.Draw(c, cx+24, cy+190, fw, 40)

	// Preview swatch.
	if p.colorDrop.Selected >= 0 && p.colorDrop.Selected < len(projectColors) {
		col := canvas.Hex(projectColors[p.colorDrop.Selected])
		c.DrawRoundedRect(cx+cw-52, cy+190, 28, 40, styles.RadiusSM, canvas.FillPaint(col))
	}

	if p.error != "" {
		c.DrawText(cx+24, cy+244, p.error, canvas.TextStyle{Color: styles.ColCancelled, Size: 12})
	}
	p.createBtn.Draw(c, cx+24, cy+258, fw, 44)
	p.cancelBtn.Draw(c, cx+24, cy+312, fw, 40)

	if p.loading {
		c.DrawRect(cx, cy, cw, 360, canvas.FillPaint(canvas.Color{A: 0.5}))
		c.DrawCenteredText(canvas.Rect{X: cx, Y: cy, W: cw, H: 360}, "Creating project…", styles.Body)
	}
}
func (p *CreateProjectPanel) doCreate() {
	ws := p.app.ActiveWorkspace.Get()
	if ws == nil {
		p.error = "No workspace selected."
		return
	}
	name := p.nameInput.Text
	if name == "" {
		p.error = "Name is required."
		return
	}
	color := "#6366F1"
	if p.colorDrop.Selected >= 0 && p.colorDrop.Selected < len(projectColors) {
		color = projectColors[p.colorDrop.Selected]
	}
	p.loading = true
	p.error = ""
	go func() {
		_, err := p.app.API.CreateProject(context.Background(), ws.ID, name, p.descInput.Text, color)
		p.loading = false
		if err != nil {
			p.error = err.Error()
			return
		}
		if p.onDone != nil {
			p.onDone()
		}
	}()
}
