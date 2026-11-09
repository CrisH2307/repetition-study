# U3 research

Verified (read from module cache source, not run against the live APIs)
- genai v1.77.0: `genai.NewClient(ctx, &genai.ClientConfig{APIKey})`, `client.Models.GenerateContent(ctx, model, []*genai.Content, *genai.GenerateContentConfig)`,
  config has `SystemInstruction *Content`, `ResponseMIMEType`; `resp.Text()`, `resp.UsageMetadata.PromptTokenCount/CandidatesTokenCount` (int32).
- anthropic-sdk-go v1.84.0: `anthropic.NewClient(option.WithAPIKey(k))`, `client.Messages.New(ctx, anthropic.MessageNewParams{Model, MaxTokens, System []TextBlockParam, Messages})`,
  `anthropic.NewUserMessage(anthropic.NewTextBlock(s))`, `msg.Content[i].AsText().Text`, `msg.Usage.InputTokens/OutputTokens` (int64). Model const `ModelClaudeSonnet5_5` exists.
- domain.LLM.JSON has no usage return, so evalcheck wraps its own provider adapters to meter latency and tokens. internal/llm is U2's and not on this branch.

Unverified (all real SDK calls, plus these assumed shapes, since U1 data is not merged)
- eval/data/explanations.jsonl: one object per line with `id`, `concept_id`, `text`.
- eval/data/labels_final.csv: header with `id` and `label` (supported|partial|contradicted|absent), one row per explanation.
- content/unit.json: `{"sources": [domain.Source], "concepts": [domain.Concept]}`.
- Prices: none hardcoded. Cost needs --price-in and --price-out (USD per million tokens), otherwise tokens only.
- Explanation level label = worst result over the concept's required claims, order contradicted > absent > partial > supported.
- Hint overlap threshold (5 consecutive words) and no minimum quote length are my choices, not in the prompt.
