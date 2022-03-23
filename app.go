package azugo

type App struct {
	env Environment

	// AppVer settings
	AppVer       string
	AppBuiltWith string
	AppName      string
}

func New() *App {
	return &App{}
}

// SetVersion sets application version and built with tags
func (a *App) SetVersion(version, builtWith string) {
	a.AppVer = version
	a.AppBuiltWith = builtWith
}

// Env returns the current application environment
func (a *App) Env() Environment {
	return a.env
}
