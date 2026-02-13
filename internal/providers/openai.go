package providers

func newOpenAIClient(opts ClientOptions) Client {
	if opts.BaseURL == "" {
		opts.BaseURL = "https://api.openai.com/v1"
	}
	return newOpenAICompatibleClient(OpenAICompatibleSettings{Name: "openai", RequireAPIKey: true}, opts)
}
