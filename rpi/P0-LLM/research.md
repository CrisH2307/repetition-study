# Research (SDK APIs read from the module cache; none run against a live API)
- anthropic-sdk-go v1.84.0: `MessageNewParams.OutputConfig.Format = JSONOutputFormatParam{Schema}` (structured output); `Messages.NewStreaming` + `Message.Accumulate` carry usage. Which Claude ids accept structured output: unverified.
- genai v1.77.0: `GenerateContentConfig.ThinkingConfig{ThinkingBudget *int32}`, `ResponseJsonSchema`, `UsageMetadata.ThoughtsTokenCount`. Whether the student model accepts budget 0: unverified.
- invopop/jsonschema v0.14.0 (already in go.sum) builds the schema from the Go type. It ignores pointer nullability, so `schema.go` wraps pointer fields as anyOf [T, null].
- Price in models.json: only claude-sonnet-5-5 ($2/$10 per Mtok) from the strategy review. Gemini prices not known: unpriced.
