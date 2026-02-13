package providers

func newOpenRouterClient(opts ClientOptions) Client {
	if opts.BaseURL == "" {
		opts.BaseURL = "https://openrouter.ai/api/v1"
	}
	if opts.Headers == nil {
		opts.Headers = map[string]string{}
	}
	return newOpenAICompatibleClient(OpenAICompatibleSettings{Name: "openrouter", RequireAPIKey: true}, opts)
}
