# U2 research

Verified from module cache source (genai v1.77.0, anthropic-sdk-go v1.84.0). No live call made: no keys.

- genai: `genai.NewClient(ctx, &genai.ClientConfig{APIKey, HTTPOptions})`; stream is
  `client.Models.GenerateContentStream(ctx, model, []*genai.Content, *GenerateContentConfig) iter.Seq2[*GenerateContentResponse, error]`;
  `resp.Text()` joins text parts; `SystemInstruction *Content`, `MaxOutputTokens int32`, `ThinkingConfig{ThinkingBudget *int32}`.
- anthropic: `anthropic.NewClient(option.WithAPIKey, WithMaxRetries, WithBaseURL)`; `client.Messages.New(ctx, MessageNewParams{Model, MaxTokens, System []TextBlockParam, Messages []MessageParam})`;
  reply text via `msg.Content[i].AsText().Text` (ContentBlockUnion.Type == "text").
- No domain interface covers session persistence or concept/source loading. U2 defines `teach.Store`
  (memory impl for tests). A pgx impl is NOT built here: INT must supply one (tables exist in 0001_core.sql).
- STUDENT_MODEL and CHECKER_MODEL are not in .env.example (U0 file). Unverified: model ids that work with the keys.
- Checker (U3) is async and may return results per turn; U2 treats any result set as "latest wins by turn number".
- Report type has no separate citation/correction/gap fields: they live inside ClaimResult; gaps = non-supported labels.
