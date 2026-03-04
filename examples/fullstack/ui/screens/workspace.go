package screens

import (
	"context"
	"fmt"

	"github.com/achiket/gui-go/canvas"
	"github.com/achiket/gui-go/ui"
	"github.com/achiket/taskflow/internal/models"
	"github.com/achiket/taskflow/ui/state"
	"github.com/achiket/taskflow/ui/styles"
)

// ═══════════════════════════════════════════════════════════════════════════════
// WorkspaceScreen — workspace picker / onboarding
// ═══════════════════════════════════════════════════════════════════════════════

type WorkspaceScreen struct {
	ui.BaseScreen
	app        *state.App
	workspaces []models.Workspace
	loading    bool
	error      string
	bounds     canvas.Rect

	createBtn *ui.Button
	wsBtns    []*ui.Button
}

func NewWorkspaceScreen(app *state.App) *WorkspaceScreen {
	s := &WorkspaceScreen{app: app}
	s.createBtn = ui.NewButton("+ New Workspace", func() {
		s.Nav.Push(NewCreateWorkspaceScreen(s.app))
	})
	s.createBtn.Style = styles.PrimaryButtonStyle()
	return s
}

func (s *WorkspaceScreen) OnEnter(nav *ui.Navigator) {
	s.BaseScreen.OnEnter(nav)
	s.loadWorkspaces()
}

func (s *WorkspaceScreen) loadWorkspaces() {
	s.loading = true
	go func() {
		ws, err := s.app.API.ListWorkspaces(context.Background())
		s.loading = false
		if err != nil {
			s.error = err.Error()
			return
		}
		s.workspaces = ws
		s.wsBtns = nil
		for _, w := range ws {
			w := w
			btn := ui.NewButton(w.Name, func() {
				s.app.ActiveWorkspace.Set(&w)
				s.Nav.Replace(NewMainScreen(s.app))
			})
			btn.Style = styles.SecondaryButtonStyle()
			s.wsBtns = append(s.wsBtns, btn)
		}
	}()
}

func (s *WorkspaceScreen) Bounds() canvas.Rect { return s.bounds }
func (s *WorkspaceScreen) Tick(d float64) {
	s.createBtn.Tick(d)
	for _, b := range s.wsBtns {
		b.Tick(d)
	}
}
func (s *WorkspaceScreen) HandleEvent(e ui.Event) bool {
	if s.createBtn.HandleEvent(e) {
		return true
	}
	for _, b := range s.wsBtns {
		if b.HandleEvent(e) {
			return true
		}
	}
	return false
}
func (s *WorkspaceScreen) Draw(c *canvas.Canvas, x, y, w, h float32) {
	s.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	c.DrawRect(x, y, w, h, canvas.FillPaint(styles.BgBase))

	cx := x + w/2
	c.DrawText(cx-c.MeasureText("Your Workspaces", styles.H2).W/2, y+56, "Your Workspaces", styles.H2)

	if s.loading {
		c.DrawCenteredText(canvas.Rect{X: x, Y: y + 100, W: w, H: 40}, "Loading…", styles.Label)
		return
	}
	if s.error != "" {
		c.DrawCenteredText(canvas.Rect{X: x, Y: y + 100, W: w, H: 40}, s.error, canvas.TextStyle{Color: styles.ColCancelled, Size: 13})
		return
	}

	if len(s.workspaces) == 0 {
		c.DrawCenteredText(canvas.Rect{X: x, Y: y + 120, W: w, H: 40}, "No workspaces yet.", styles.Label)
	}

	btnW := float32(360)
	bx := cx - btnW/2
	by := y + 100.0

	for i, btn := range s.wsBtns {
		ws := s.workspaces[i]
		accentCol := styles.Indigo
		styles.DrawCard(c, bx, by, btnW, 52, &accentCol)
		_ = ws
		btn.Draw(c, bx+12, by+6, btnW-24, 40)
		c.DrawText(bx+btnW-70, by+20, fmt.Sprintf("%d projects", 0), styles.Tiny)
		by += 62
	}

	s.createBtn.Draw(c, bx, by+12, btnW, 44)
}

// ═══════════════════════════════════════════════════════════════════════════════
// CreateWorkspaceScreen
// ═══════════════════════════════════════════════════════════════════════════════

type CreateWorkspaceScreen struct {
	ui.BaseScreen
	app         *state.App
	nameInput   *ui.TextInput
	descInput   *ui.TextInput
	createBtn   *ui.Button
	cancelBtn   *ui.Button
	errorLabel  string
	loading     bool
	bounds      canvas.Rect
}

func NewCreateWorkspaceScreen(app *state.App) *CreateWorkspaceScreen {
	s := &CreateWorkspaceScreen{app: app}
	s.nameInput = ui.NewTextInput("")
	s.nameInput.Hint = "Workspace name"
	s.descInput = ui.NewTextInput("")
	s.descInput.Hint = "Description (optional)"
	s.createBtn = ui.NewButton("Create Workspace", func() { s.doCreate() })
	s.createBtn.Style = styles.PrimaryButtonStyle()
	s.cancelBtn = ui.NewButton("Cancel", func() { s.Nav.Pop() })
	s.cancelBtn.Style = styles.SecondaryButtonStyle()
	return s
}

func (s *CreateWorkspaceScreen) Bounds() canvas.Rect { return s.bounds }
func (s *CreateWorkspaceScreen) Tick(d float64) {
	s.nameInput.Tick(d); s.descInput.Tick(d)
	s.createBtn.Tick(d); s.cancelBtn.Tick(d)
}
func (s *CreateWorkspaceScreen) HandleEvent(e ui.Event) bool {
	return s.nameInput.HandleEvent(e) || s.descInput.HandleEvent(e) ||
		s.createBtn.HandleEvent(e) || s.cancelBtn.HandleEvent(e)
}
func (s *CreateWorkspaceScreen) Draw(c *canvas.Canvas, x, y, w, h float32) {
	s.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	c.DrawRect(x, y, w, h, canvas.FillPaint(styles.BgBase))

	cw := float32(400)
	cx := x + (w-cw)/2
	cy := y + 80.0
	styles.DrawCard(c, cx, cy, cw, 340, nil)

	c.DrawText(cx+24, cy+36, "Create Workspace", styles.H2)

	fw := cw - 48
	s.nameInput.Draw(c, cx+24, cy+72, fw, 42)
	s.descInput.Draw(c, cx+24, cy+124, fw, 42)

	if s.errorLabel != "" {
		c.DrawText(cx+24, cy+176, s.errorLabel, canvas.TextStyle{Color: styles.ColCancelled, Size: 12})
	}
	s.createBtn.Draw(c, cx+24, cy+190, fw, 44)
	s.cancelBtn.Draw(c, cx+24, cy+244, fw, 40)

	if s.loading {
		c.DrawRect(cx, cy, cw, 340, canvas.FillPaint(canvas.Color{A: 0.5}))
		c.DrawCenteredText(canvas.Rect{X: cx, Y: cy, W: cw, H: 340}, "Creating…", styles.Body)
	}
}
func (s *CreateWorkspaceScreen) doCreate() {
	name := s.nameInput.Text
	if name == "" {
		s.errorLabel = "Name is required."
		return
	}
	s.loading = true
	go func() {
		_, err := s.app.API.CreateWorkspace(context.Background(), name, s.descInput.Text)
		s.loading = false
		if err != nil {
			s.errorLabel = err.Error()
			return
		}
		s.Nav.Pop()
	}()
}

// ═══════════════════════════════════════════════════════════════════════════════
// MainScreen — the root app shell with sidebar + content area
// ═══════════════════════════════════════════════════════════════════════════════

const sidebarW = float32(220)

type MainScreen struct {
	ui.BaseScreen
	app      *state.App
	sidebar  *SidebarWidget
	content  ui.Component // current content panel
	bounds   canvas.Rect
}

func NewMainScreen(app *state.App) *MainScreen {
	s := &MainScreen{app: app}
	s.sidebar = NewSidebarWidget(app, func(panel ui.Component) {
		s.content = panel
	})
	s.content = NewDashboardPanel(app)
	return s
}

func (s *MainScreen) OnEnter(nav *ui.Navigator) {
	s.BaseScreen.OnEnter(nav)
	s.sidebar.nav = nav
}

func (s *MainScreen) Bounds() canvas.Rect { return s.bounds }
func (s *MainScreen) Tick(d float64) {
	s.sidebar.Tick(d)
	if s.content != nil {
		s.content.Tick(d)
	}
}
func (s *MainScreen) HandleEvent(e ui.Event) bool {
	if s.sidebar.HandleEvent(e) {
		return true
	}
	if s.content != nil {
		return s.content.HandleEvent(e)
	}
	return false
}
func (s *MainScreen) Draw(c *canvas.Canvas, x, y, w, h float32) {
	s.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	c.DrawRect(x, y, w, h, canvas.FillPaint(styles.BgBase))

	// Sidebar.
	c.DrawRect(x, y, sidebarW, h, canvas.FillPaint(styles.BgSurface))
	c.DrawRect(x+sidebarW, y, 1, h, canvas.FillPaint(styles.BorderSubtle))
	s.sidebar.Draw(c, x, y, sidebarW, h)

	// Content.
	if s.content != nil {
		s.content.Draw(c, x+sidebarW+1, y, w-sidebarW-1, h)
	}
}

// ─── SidebarWidget ────────────────────────────────────────────────────────────

type SidebarWidget struct {
	app       *state.App
	nav       *ui.Navigator
	onSelect  func(ui.Component)
	projects  []models.Project
	loading   bool
	activeIdx int
	bounds    canvas.Rect

	dashBtn     *ui.Button
	newProjBtn  *ui.Button
	logoutBtn   *ui.Button
	projBtns    []*ui.Button
}

func NewSidebarWidget(app *state.App, onSelect func(ui.Component)) *SidebarWidget {
	s := &SidebarWidget{app: app, onSelect: onSelect, activeIdx: -1}

	s.dashBtn = ui.NewButton("Dashboard", func() {
		s.activeIdx = -1
		s.app.ActiveProject.Set(nil)
		s.onSelect(NewDashboardPanel(s.app))
	})
	s.dashBtn.Style = styles.SecondaryButtonStyle()

	s.newProjBtn = ui.NewButton("+ New Project", func() {
		// Push a modal-style create project panel.
		s.onSelect(NewCreateProjectPanel(s.app, func() {
			s.reloadProjects()
			s.onSelect(NewDashboardPanel(s.app))
		}))
	})
	s.newProjBtn.Style = styles.PrimaryButtonStyle()

	s.logoutBtn = ui.NewButton("Logout", func() {
		go func() {
			_ = s.app.API.Logout(context.Background())
			s.app.ClearSession()
			s.nav.Replace(NewLoginScreen(s.app))
		}()
	})
	s.logoutBtn.Style = styles.DangerButtonStyle()

	s.reloadProjects()
	return s
}

func (s *SidebarWidget) reloadProjects() {
	ws := s.app.ActiveWorkspace.Get()
	if ws == nil {
		return
	}
	s.loading = true
	go func() {
		projs, err := s.app.API.ListProjects(context.Background(), ws.ID)
		s.loading = false
		if err != nil {
			return
		}
		s.projects = projs
		s.projBtns = nil
		for i, p := range projs {
			i, p := i, p
			btn := ui.NewButton(p.Name, func() {
				s.activeIdx = i
				s.app.ActiveProject.Set(&p)
				s.onSelect(NewProjectBoard(s.app, p.ID, ws.ID))
			})
			btn.Style = styles.SecondaryButtonStyle()
			s.projBtns = append(s.projBtns, btn)
		}
	}()
}

func (s *SidebarWidget) Bounds() canvas.Rect { return s.bounds }
func (s *SidebarWidget) Tick(d float64) {
	s.dashBtn.Tick(d); s.newProjBtn.Tick(d); s.logoutBtn.Tick(d)
	for _, b := range s.projBtns {
		b.Tick(d)
	}
}
func (s *SidebarWidget) HandleEvent(e ui.Event) bool {
	if s.dashBtn.HandleEvent(e) || s.newProjBtn.HandleEvent(e) || s.logoutBtn.HandleEvent(e) {
		return true
	}
	for _, b := range s.projBtns {
		if b.HandleEvent(e) {
			return true
		}
	}
	return false
}
func (s *SidebarWidget) Draw(c *canvas.Canvas, x, y, w, h float32) {
	s.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}

	// Logo.
	c.DrawText(x+20, y+32, "TaskFlow", styles.H3)
	ws := s.app.ActiveWorkspace.Get()
	wsName := "No workspace"
	if ws != nil {
		wsName = ws.Name
	}
	c.DrawText(x+20, y+52, wsName, styles.Small)
	c.DrawRect(x+12, y+64, w-24, 1, canvas.FillPaint(styles.BorderSubtle))

	// Dashboard button.
	s.dashBtn.Draw(c, x+10, y+78, w-20, 36)

	// Projects header.
	c.DrawText(x+14, y+130, "PROJECTS", styles.Tiny)
	s.newProjBtn.Draw(c, x+10, y+144, w-20, 30)

	// Project list.
	py := y + 184.0
	if s.loading {
		c.DrawText(x+20, py+12, "Loading…", styles.Small)
	}
	for i, btn := range s.projBtns {
		p := s.projects[i]
		accentCol := styles.Indigo
		if p.Color != "" {
			accentCol = canvas.Hex(p.Color)
		}
		// Active indicator.
		if i == s.activeIdx {
			c.DrawRoundedRect(x+8, py, w-16, 34, styles.RadiusSM,
				canvas.FillPaint(canvas.Color{R: styles.Indigo.R, G: styles.Indigo.G, B: styles.Indigo.B, A: 0.15}))
			c.DrawRect(x+8, py, 3, 34, canvas.FillPaint(accentCol))
		}
		btn.Draw(c, x+18, py, w-28, 34)
		// Task count badge.
		countLabel := fmt.Sprintf("%d", p.TaskCount)
		tw := c.MeasureText(countLabel, styles.Tiny).W
		c.DrawRoundedRect(x+w-tw-16, py+9, tw+10, 16, styles.RadiusSM,
			canvas.FillPaint(styles.BgInput))
		c.DrawText(x+w-tw-11, py+22, countLabel, styles.Tiny)
		py += 38
	}

	// Bottom: user info.
	user := s.app.CurrentUser.Get()
	if user != nil {
		uy := y + h - 72
		c.DrawRect(x+12, uy-8, w-24, 1, canvas.FillPaint(styles.BorderSubtle))
		styles.DrawAvatar(c, user.DisplayName, styles.AvatarColor(user.DisplayName), x+14, uy+4, 32)
		c.DrawText(x+52, uy+14, user.DisplayName, styles.Body)
		c.DrawText(x+52, uy+30, string(user.Role), styles.Small)
		s.logoutBtn.Draw(c, x+10, uy+46, w-20, 28)
	}
}

// ─── DashboardPanel ───────────────────────────────────────────────────────────

type DashboardPanel struct {
	app    *state.App
	stats  map[string]any
	loading bool
	bounds canvas.Rect
	scroll *ui.ScrollView
}

func NewDashboardPanel(app *state.App) *DashboardPanel {
	p := &DashboardPanel{app: app}
	p.scroll = ui.NewScrollView(800, func(c *canvas.Canvas, x, y, w, _ float32) {
		p.drawContent(c, x, y, w)
	})
	ws := app.ActiveWorkspace.Get()
	if ws != nil {
		p.reload(ws.ID)
	}
	return p
}

func (p *DashboardPanel) reload(wsID string) {
	p.loading = true
	go func() {
		stats, err := p.app.API.Dashboard(context.Background(), wsID)
		p.loading = false
		if err != nil {
			return
		}
		p.stats = stats
	}()
}

func (p *DashboardPanel) Bounds() canvas.Rect        { return p.bounds }
func (p *DashboardPanel) Tick(d float64)              { p.scroll.Tick(d) }
func (p *DashboardPanel) HandleEvent(e ui.Event) bool { return p.scroll.HandleEvent(e) }
func (p *DashboardPanel) Draw(c *canvas.Canvas, x, y, w, h float32) {
	p.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	p.scroll.Draw(c, x, y, w, h)
}

func (p *DashboardPanel) drawContent(c *canvas.Canvas, x, y, w float32) {
	// Header.
	ws := p.app.ActiveWorkspace.Get()
	title := "Dashboard"
	if ws != nil {
		title = ws.Name + " — Dashboard"
	}
	c.DrawText(x+24, y+40, title, styles.H1)

	if p.loading {
		c.DrawText(x+24, y+80, "Loading stats…", styles.Label)
		return
	}

	// Stat cards row.
	statW := float32(180)
	stats := []struct {
		label string
		key   string
		color canvas.Color
	}{
		{"Total Tasks", "TotalTasks", styles.Indigo},
		{"Completed", "DoneTasks", styles.ColDone},
		{"In Progress", "InProgress", styles.ColInProgress},
		{"Members", "Members", styles.ColInReview},
		{"Projects", "Projects", styles.ColMedium},
	}
	sx := x + 24.0
	for _, st := range stats {
		val := 0
		if p.stats != nil {
			if v, ok := p.stats[st.key]; ok {
				switch n := v.(type) {
				case float64:
					val = int(n)
				case int:
					val = n
				}
			}
		}
		styles.DrawCard(c, sx, y+68, statW, 80, &st.color)
		c.DrawText(sx+14, y+96, fmt.Sprintf("%d", val), styles.H1)
		c.DrawText(sx+14, y+118, st.label, styles.Small)
		sx += statW + 12
	}

	// Recent activity.
	c.DrawText(x+24, y+176, "Recent Activity", styles.H3)
	ay := y + 200.0
	if p.stats != nil {
		if acts, ok := p.stats["RecentActivity"]; ok {
			if actList, ok := acts.([]any); ok {
				for i, act := range actList {
					if i >= 8 {
						break
					}
					if m, ok := act.(map[string]any); ok {
						actor := "System"
						if a, ok := m["actor"].(map[string]any); ok {
							if dn, ok := a["display_name"].(string); ok {
								actor = dn
							}
						}
						action, _ := m["action"].(string)
						entity, _ := m["entity_type"].(string)
						line := fmt.Sprintf("%s  %s  %s", actor, action, entity)
						c.DrawText(x+24, ay, line, styles.Body)
						ay += 22
					}
				}
			}
		}
	}
}
