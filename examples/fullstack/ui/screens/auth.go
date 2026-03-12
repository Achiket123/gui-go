// Package screens — application screens.
package screens

import (
	"context"

	"github.com/achiket123/gui-go/canvas"
	"github.com/achiket123/gui-go/ui"
	"github.com/achiket123/taskflow/ui/state"
	"github.com/achiket123/taskflow/ui/styles"
)

// ═══════════════════════════════════════════════════════════════════════════════
// LoginScreen
// ═══════════════════════════════════════════════════════════════════════════════

// LoginScreen renders the sign-in form and handles authentication.
type LoginScreen struct {
	ui.BaseScreen
	app *state.App

	emailInput    *ui.TextInput
	passwordInput *ui.TextInput
	loginBtn      *ui.Button
	registerLink  *ui.Button

	errorLabel string
	loading    bool
	bounds     canvas.Rect
}

func NewLoginScreen(app *state.App) *LoginScreen {
	s := &LoginScreen{app: app}

	s.emailInput = ui.NewTextInput("")
	s.emailInput.Hint = "Email address"

	s.passwordInput = ui.NewTextInput("")
	s.passwordInput.Hint = "Password"

	s.loginBtn = ui.NewButton("Sign In", func() { s.doLogin() })
	s.loginBtn.Style = styles.PrimaryButtonStyle()

	s.registerLink = ui.NewButton("Create account →", func() {
		if s.Nav != nil {
			s.Nav.Push(NewRegisterScreen(s.app))
		}
	})
	s.registerLink.Style = styles.SecondaryButtonStyle()

	return s
}

func (s *LoginScreen) Bounds() canvas.Rect { return s.bounds }

func (s *LoginScreen) Tick(delta float64) {
	s.emailInput.Tick(delta)
	s.passwordInput.Tick(delta)
	s.loginBtn.Tick(delta)
	s.registerLink.Tick(delta)
}

func (s *LoginScreen) HandleEvent(e ui.Event) bool {
	if s.loading {
		return true
	}
	if e.Type == ui.EventKeyDown && e.Key == "Return" {
		s.doLogin()
		return true
	}
	return s.emailInput.HandleEvent(e) ||
		s.passwordInput.HandleEvent(e) ||
		s.loginBtn.HandleEvent(e) ||
		s.registerLink.HandleEvent(e)
}

func (s *LoginScreen) Draw(c *canvas.Canvas, x, y, w, h float32) {
	s.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}

	// Background gradient-ish.
	c.DrawRect(x, y, w, h, canvas.FillPaint(styles.BgBase))
	// Decorative glow.
	c.DrawCircle(x+w*0.5, y+h*0.3, 280,
		canvas.FillPaint(canvas.Color{R: styles.Indigo.R, G: styles.Indigo.G, B: styles.Indigo.B, A: 0.05}))

	// Card.
	cardW := float32(400)
	cardH := float32(520)
	if cardW > w-40 {
		cardW = w - 40
	}
	cx := x + (w-cardW)/2
	cy := y + (h-cardH)/2

	styles.DrawCard(c, cx, cy, cardW, cardH, nil)

	// Logo / title area.
	c.DrawText(cx+cardW/2-c.MeasureText("TaskFlow", styles.H1).W/2, cy+48, "TaskFlow", styles.H1)
	subtitleStyle := canvas.TextStyle{Color: styles.TextMuted, Size: 13}
	subtitle := "Project management for teams"
	c.DrawText(cx+cardW/2-c.MeasureText(subtitle, subtitleStyle).W/2, cy+76, subtitle, subtitleStyle)

	// Divider.
	c.DrawRect(cx+24, cy+96, cardW-48, 1, canvas.FillPaint(styles.BorderSubtle))

	// Form fields.
	fieldW := cardW - 48
	s.emailInput.Draw(c, cx+24, cy+120, fieldW, 42)
	s.passwordInput.Draw(c, cx+24, cy+174, fieldW, 42)

	// Error.
	if s.errorLabel != "" {
		errStyle := canvas.TextStyle{Color: styles.ColCancelled, Size: 12}
		c.DrawText(cx+24, cy+228, s.errorLabel, errStyle)
	}

	// Buttons.
	s.loginBtn.Draw(c, cx+24, cy+250, fieldW, 44)
	s.registerLink.Draw(c, cx+24, cy+308, fieldW, 40)

	// Footer note.
	noteStyle := canvas.TextStyle{Color: styles.TextMuted, Size: 11}
	note := "Secure · End-to-end · Open Source"
	c.DrawText(cx+cardW/2-c.MeasureText(note, noteStyle).W/2, cy+cardH-20, note, noteStyle)

	// Loading overlay.
	if s.loading {
		c.DrawRect(cx, cy, cardW, cardH, canvas.FillPaint(canvas.Color{A: 0.45}))
		c.DrawCenteredText(canvas.Rect{X: cx, Y: cy, W: cardW, H: cardH},
			"Signing in…", styles.Body)
	}
}

func (s *LoginScreen) doLogin() {
	email := s.emailInput.Text
	password := s.passwordInput.Text
	if email == "" || password == "" {
		s.errorLabel = "Email and password are required."
		return
	}
	s.loading = true
	s.errorLabel = ""

	go func() {
		resp, err := s.app.API.Login(context.Background(), email, password)
		s.loading = false
		if err != nil {
			s.errorLabel = "Invalid credentials. Please try again."
			return
		}
		s.app.SetUser(resp.User, resp.Tokens.AccessToken, resp.Tokens.RefreshToken)
		// Navigate to the main app.
		if s.Nav != nil {
			s.Nav.Replace(NewWorkspaceScreen(s.app))
		}
	}()
}

// ═══════════════════════════════════════════════════════════════════════════════
// RegisterScreen
// ═══════════════════════════════════════════════════════════════════════════════

// RegisterScreen renders the sign-up form.
type RegisterScreen struct {
	ui.BaseScreen
	app *state.App

	nameInput     *ui.TextInput
	usernameInput *ui.TextInput
	emailInput    *ui.TextInput
	passwordInput *ui.TextInput
	registerBtn   *ui.Button
	backBtn       *ui.Button

	errorLabel string
	loading    bool
	bounds     canvas.Rect
}

func NewRegisterScreen(app *state.App) *RegisterScreen {
	s := &RegisterScreen{app: app}

	s.nameInput = ui.NewTextInput("")
	s.nameInput.Hint = "Full name"

	s.usernameInput = ui.NewTextInput("")
	s.usernameInput.Hint = "Username (e.g. alice)"

	s.emailInput = ui.NewTextInput("")
	s.emailInput.Hint = "Email address"

	s.passwordInput = ui.NewTextInput("")
	s.passwordInput.Hint = "Password (min 8 characters)"

	s.registerBtn = ui.NewButton("Create Account", func() { s.doRegister() })
	s.registerBtn.Style = styles.PrimaryButtonStyle()

	s.backBtn = ui.NewButton("← Sign In", func() {
		if s.Nav != nil {
			s.Nav.Pop()
		}
	})
	s.backBtn.Style = styles.SecondaryButtonStyle()

	return s
}

func (s *RegisterScreen) Bounds() canvas.Rect { return s.bounds }
func (s *RegisterScreen) Tick(delta float64) {
	s.nameInput.Tick(delta)
	s.usernameInput.Tick(delta)
	s.emailInput.Tick(delta)
	s.passwordInput.Tick(delta)
	s.registerBtn.Tick(delta)
	s.backBtn.Tick(delta)
}
func (s *RegisterScreen) HandleEvent(e ui.Event) bool {
	if s.loading {
		return true
	}
	return s.nameInput.HandleEvent(e) || s.usernameInput.HandleEvent(e) ||
		s.emailInput.HandleEvent(e) || s.passwordInput.HandleEvent(e) ||
		s.registerBtn.HandleEvent(e) || s.backBtn.HandleEvent(e)
}

func (s *RegisterScreen) Draw(c *canvas.Canvas, x, y, w, h float32) {
	s.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	c.DrawRect(x, y, w, h, canvas.FillPaint(styles.BgBase))

	cardW := float32(420)
	cardH := float32(580)
	if cardW > w-40 {
		cardW = w - 40
	}
	cx := x + (w-cardW)/2
	cy := y + (h-cardH)/2

	styles.DrawCard(c, cx, cy, cardW, cardH, nil)

	c.DrawText(cx+24, cy+36, "Create your account", styles.H2)
	c.DrawText(cx+24, cy+62, "Free forever · No credit card", styles.Small)

	fw := cardW - 48
	s.nameInput.Draw(c, cx+24, cy+96, fw, 42)
	s.usernameInput.Draw(c, cx+24, cy+148, fw, 42)
	s.emailInput.Draw(c, cx+24, cy+200, fw, 42)
	s.passwordInput.Draw(c, cx+24, cy+252, fw, 42)

	if s.errorLabel != "" {
		c.DrawText(cx+24, cy+308, s.errorLabel, canvas.TextStyle{Color: styles.ColCancelled, Size: 12})
	}

	s.registerBtn.Draw(c, cx+24, cy+320, fw, 44)
	s.backBtn.Draw(c, cx+24, cy+374, fw, 40)

	if s.loading {
		c.DrawRect(cx, cy, cardW, cardH, canvas.FillPaint(canvas.Color{A: 0.45}))
		c.DrawCenteredText(canvas.Rect{X: cx, Y: cy, W: cardW, H: cardH}, "Creating account…", styles.Body)
	}
}

func (s *RegisterScreen) doRegister() {
	name := s.nameInput.Text
	username := s.usernameInput.Text
	email := s.emailInput.Text
	password := s.passwordInput.Text
	if name == "" || email == "" || password == "" {
		s.errorLabel = "All fields are required."
		return
	}
	if len(password) < 8 {
		s.errorLabel = "Password must be at least 8 characters."
		return
	}
	s.loading = true
	s.errorLabel = ""
	go func() {
		resp, err := s.app.API.Register(context.Background(), email, username, name, password)
		s.loading = false
		if err != nil {
			s.errorLabel = "Registration failed: " + err.Error()
			return
		}
		s.app.SetUser(resp.User, resp.Tokens.AccessToken, resp.Tokens.RefreshToken)
		if s.Nav != nil {
			s.Nav.Replace(NewWorkspaceScreen(s.app))
		}
	}()
}
