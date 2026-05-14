package app

// App holds shared application state.
type App struct {
	APIKey  string
	AppId   string
	History []Message
}

type Message struct {
	Role    string
	Content string
}

func New(apiKey, appId string) *App {
	return &App{APIKey: apiKey, AppId: appId}
}
