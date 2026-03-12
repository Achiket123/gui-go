// Package state — application-wide reactive state.
// All signals are goroutine-safe and can be subscribed to by any screen.
package state

import (
	uistream "github.com/achiket123/gui-go/state"
	"github.com/achiket123/taskflow/internal/api/client"
	"github.com/achiket123/taskflow/internal/models"
)

// App holds every piece of shared state for the GUI application.
type App struct {
	// Auth.
	CurrentUser *uistream.SignalAny[*models.User]
	AccessToken *uistream.Signal[string]
	IsLoggedIn  *uistream.Signal[bool]

	// Navigation.
	ActiveWorkspace *uistream.SignalAny[*models.Workspace]
	ActiveProject   *uistream.SignalAny[*models.Project]

	// Lists (refreshed on navigation).
	Workspaces *uistream.SignalAny[[]models.Workspace]
	Projects   *uistream.SignalAny[[]models.Project]
	Tasks      *uistream.SignalAny[[]models.Task]

	// UI state.
	Loading        *uistream.Signal[bool]
	ErrorMessage   *uistream.Signal[string]
	SuccessMessage *uistream.Signal[string]

	// API client.
	API *client.Client

	// Event bus for cross-screen notifications.
	Bus *uistream.EventBus
}

// NewApp creates and wires the global application state.
func NewApp(apiClient *client.Client) *App {
	a := &App{
		CurrentUser:     uistream.NewSignalAny[*models.User](nil),
		AccessToken:     uistream.New(""),
		IsLoggedIn:      uistream.New(false),
		ActiveWorkspace: uistream.NewSignalAny[*models.Workspace](nil),
		ActiveProject:   uistream.NewSignalAny[*models.Project](nil),
		Workspaces:      uistream.NewSignalAny[[]models.Workspace](nil),
		Projects:        uistream.NewSignalAny[[]models.Project](nil),
		Tasks:           uistream.NewSignalAny[[]models.Task](nil),
		Loading:         uistream.New(false),
		ErrorMessage:    uistream.New(""),
		SuccessMessage:  uistream.New(""),
		API:             apiClient,
		Bus:             uistream.NewEventBus(),
	}
	return a
}

// SetUser stores a logged-in user and flips IsLoggedIn.
func (a *App) SetUser(u *models.User, access, refresh string) {
	a.CurrentUser.Set(u)
	a.AccessToken.Set(access)
	a.API.SetTokens(access, refresh)
	a.IsLoggedIn.Set(u != nil)
}

// ClearSession logs out and clears all state.
func (a *App) ClearSession() {
	a.CurrentUser.Set(nil)
	a.AccessToken.Set("")
	a.IsLoggedIn.Set(false)
	a.ActiveWorkspace.Set(nil)
	a.ActiveProject.Set(nil)
	a.Workspaces.Set(nil)
	a.Projects.Set(nil)
	a.Tasks.Set(nil)
}

// ShowError sets the error message (cleared after 4 s by the toast widget).
func (a *App) ShowError(msg string) { a.ErrorMessage.Set(msg) }

// ShowSuccess sets the success message.
func (a *App) ShowSuccess(msg string) { a.SuccessMessage.Set(msg) }

// SetLoading toggles the global loading indicator.
func (a *App) SetLoading(v bool) { a.Loading.Set(v) }
