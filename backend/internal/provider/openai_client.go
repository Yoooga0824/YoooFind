package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"gemini-clone/backend/internal/model"
	"gemini-clone/backend/internal/websearch"
)

// OpenAICompatibleClient calls any provider that supports
// OpenAI-style chat completions endpoint.
//
// Example providers:
// - DeepSeek official
// - Some OpenAI-compatible gateways
type OpenAICompatibleClient struct {
	baseURL          string
	path             string
	apiKey           string
	model            string
	maxTokens        int
	temperature      float64
	upstreamRequestTimeout time.Duration
	httpClient       *http.Client
	searchClient     websearch.Client
	searchMaxResults int
}

const maxToolLoopSteps = 4
const webSearchToolName = "web_search"
const maxSameQueryRepeats = 2
const deepSearchHeartbeatInterval = 5 * time.Second

// NewOpenAICompatibleClient creates a reusable API client.
func NewOpenAICompatibleClient(
	baseURL, path, apiKey, model string,
	maxTokens int,
	temperature float64,
	upstreamRequestTimeout time.Duration,
	searchClient websearch.Client,
	searchMaxResults int,
) *OpenAICompatibleClient {
	if upstreamRequestTimeout <= 0 {
		upstreamRequestTimeout = 120 * time.Second
	}
	return &OpenAICompatibleClient{
		baseURL:     strings.TrimRight(baseURL, "/"),
		path:        path,
		apiKey:      apiKey,
		model:       model,
		maxTokens:   maxTokens,
		temperature: temperature,
		upstreamRequestTimeout: upstreamRequestTimeout,
		httpClient: &http.Client{
			// Streaming requests should not be cut off by a fixed client timeout.
			// We keep a per-request timeout for non-stream calls.
			Timeout: 0,
		},
		searchClient:     searchClient,
		searchMaxResults: searchMaxResults,
	}
}

type chatRequest struct {
	Model         string             `json:"model"`
	Messages      []chatMessage      `json:"messages"`
	MaxTokens     int                `json:"max_tokens,omitempty"`
	Temperature   float64            `json:"temperature,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
	StreamOptions *chatStreamOptions `json:"stream_options,omitempty"`
	Tools         []chatTool         `json:"tools,omitempty"`
	ToolChoice    string             `json:"tool_choice,omitempty"`
}

type chatStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type chatMessage struct {
	Role             string         `json:"role"`
	Content          string         `json:"content"`
	ReasoningContent string         `json:"reasoning_content,omitempty"`
	ToolCallID       string         `json:"tool_call_id,omitempty"`
	ToolCalls        []chatToolCall `json:"tool_calls,omitempty"`
}

type chatTool struct {
	Type     string           `json:"type"`
	Function chatToolFunction `json:"function"`
}

type chatToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type chatToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function chatToolCallDetail `json:"function"`
}

type chatToolCallDetail struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// upstreamAPIError accepts both OpenAI-style {"message":"..."} and plain string
// error values returned by some providers (e.g. Kimi).
type upstreamAPIError struct {
	Message string
}

func (e *upstreamAPIError) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		e.Message = asString
		return nil
	}
	var asObject struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(data, &asObject); err != nil {
		return err
	}
	e.Message = asObject.Message
	return nil
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
	Error *upstreamAPIError `json:"error,omitempty"`
}

type chatStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content          string         `json:"content"`
			ReasoningContent string         `json:"reasoning_content"`
			Reasoning        string         `json:"reasoning"`
			ToolCalls        []chatToolCall `json:"tool_calls"`
		} `json:"delta"`
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
	Error *upstreamAPIError `json:"error,omitempty"`
}

var thinkTagPattern = regexp.MustCompile(`(?s)<think>\s*(.*?)\s*</think>`)
var explicitDateTimeQuestionPattern = regexp.MustCompile(
	`(?i)(今天(是)?几号|今天(是)?几月几日|今天(是)?星期几|今天(是)?周几|今天日期|现在几点|当前时间|当前日期|what('?| i)?s the date|what time is it|today'?s date|current date|current time)`,
)
var strictDateTimeQuestionPattern = regexp.MustCompile(
	`(?i)^(请问|麻烦问下|想问下|告诉我)?\s*(今天|现在|当前|today|now|current)?\s*(日期|时间|几号|几月几日|星期几|周几|几点|date|time)\s*(是多少|是啥|是什么|吗|\?)?$`,
)
var dateSensitiveQueryPattern = regexp.MustCompile(`(?i)(今天|明天|昨天|本周|本月|本季度|今年|去年|近期|最近|最新|当前|目前|截至|as of|today|now|current|latest|recent)`)

// GenerateReply sends user message to upstream LLM and returns separated answer/reasoning text.
func (c *OpenAICompatibleClient) GenerateReply(
	ctx context.Context,
	userMessage string,
	replyOptions model.ReplyOptions,
) (model.AssistantReply, error) {
	if shouldAnswerWithLocalDate(userMessage) {
		return model.AssistantReply{
			Content: buildLocalDateAnswer(),
		}, nil
	}

	if replyOptions.DeepSearch {
		reply, err := c.generateWithToolLoop(ctx, userMessage, nil)
		if err == nil {
			return reply, nil
		}
		// 深度搜索失败时自动降级，避免影响主聊天能力。
		fallback, fallbackErr := c.generateOnce(ctx, []chatMessage{
			{Role: "user", Content: userMessage},
		}, nil)
		if fallbackErr != nil {
			return model.AssistantReply{}, err
		}
		if strings.TrimSpace(fallback.ReasoningContent) == "" {
			fallback.ReasoningContent = "深度搜索未成功，已自动切换为常规回答。"
		} else {
			fallback.ReasoningContent = "深度搜索未成功，已自动切换为常规回答。\n\n" + fallback.ReasoningContent
		}
		return fallback, nil
	}

	return c.generateOnce(ctx, []chatMessage{
		{Role: "user", Content: userMessage},
	}, nil)
}

func (c *OpenAICompatibleClient) generateOnce(
	ctx context.Context,
	messages []chatMessage,
	tools []chatTool,
) (model.AssistantReply, error) {
	decoded, err := c.executeChatCompletion(ctx, messages, tools)
	if err != nil {
		return model.AssistantReply{}, err
	}
	content, reasoning := splitReasoningAndContent(
		decoded.Choices[0].Message.Content,
		decoded.Choices[0].Message.ReasoningContent,
	)
	return model.AssistantReply{
		Content:          content,
		ReasoningContent: reasoning,
		Usage:            toTokenUsage(decoded.Usage),
	}, nil
}

func (c *OpenAICompatibleClient) executeChatCompletion(
	ctx context.Context,
	messages []chatMessage,
	tools []chatTool,
) (chatResponse, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, c.upstreamRequestTimeout)
	defer cancel()

	payload := chatRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   c.maxTokens,
		Temperature: c.temperature,
		Tools:       tools,
		ToolChoice:  toolChoiceValue(tools),
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return chatResponse{}, fmt.Errorf("marshal upstream request: %w", err)
	}

	url := c.baseURL + c.path
	req, err := http.NewRequestWithContext(ctxWithTimeout, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return chatResponse{}, fmt.Errorf("create upstream request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return chatResponse{}, fmt.Errorf("call upstream: %w", err)
	}
	defer resp.Body.Close()

	rawRespBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return chatResponse{}, fmt.Errorf("read upstream response: %w", err)
	}

	var decoded chatResponse
	if err := json.Unmarshal(rawRespBody, &decoded); err != nil {
		return chatResponse{}, fmt.Errorf("decode upstream response: %w", err)
	}

	if resp.StatusCode >= 400 {
		msg := "upstream request failed"
		if decoded.Error != nil && decoded.Error.Message != "" {
			msg = decoded.Error.Message
		}
		return chatResponse{}, fmt.Errorf("%s (status %d)", msg, resp.StatusCode)
	}

	if len(decoded.Choices) == 0 {
		return chatResponse{}, fmt.Errorf("upstream returned no choices")
	}
	return decoded, nil
}

// StreamReply streams chunks from upstream and returns final normalized assistant reply.
func (c *OpenAICompatibleClient) StreamReply(
	ctx context.Context,
	userMessage string,
	replyOptions model.ReplyOptions,
	onDelta func(model.AssistantReplyDelta) error,
) (model.AssistantReply, error) {
	if shouldAnswerWithLocalDate(userMessage) {
		answer := buildLocalDateAnswer()
		if onDelta != nil {
			if err := onDelta(model.AssistantReplyDelta{Content: answer}); err != nil {
				return model.AssistantReply{}, err
			}
		}
		return model.AssistantReply{
			Content: answer,
		}, nil
	}

	if replyOptions.DeepSearch {
		return c.streamDeepSearchFallback(ctx, userMessage, onDelta)
	}

	payload := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "user", Content: userMessage},
		},
		MaxTokens:   c.maxTokens,
		Temperature: c.temperature,
		Stream:      true,
		StreamOptions: &chatStreamOptions{
			IncludeUsage: true,
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return model.AssistantReply{}, fmt.Errorf("marshal upstream stream request: %w", err)
	}

	url := c.baseURL + c.path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return model.AssistantReply{}, fmt.Errorf("create upstream stream request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return model.AssistantReply{}, fmt.Errorf("call upstream stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		rawBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return model.AssistantReply{}, fmt.Errorf("read upstream stream error response: %w", readErr)
		}
		var decoded chatResponse
		_ = json.Unmarshal(rawBody, &decoded)
		msg := "upstream stream request failed"
		if decoded.Error != nil && decoded.Error.Message != "" {
			msg = decoded.Error.Message
		}
		return model.AssistantReply{}, fmt.Errorf("%s (status %d)", msg, resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	var fullContent strings.Builder
	var fullReasoning strings.Builder
	var usage *model.TokenUsage
	var emittedContentLen int
	var emittedReasoningLen int

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}

		payloadLine := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payloadLine == "" || payloadLine == "[DONE]" {
			continue
		}

		var chunk chatStreamChunk
		if err := json.Unmarshal([]byte(payloadLine), &chunk); err != nil {
			continue
		}
		if chunk.Error != nil && chunk.Error.Message != "" {
			return model.AssistantReply{}, fmt.Errorf("%s", chunk.Error.Message)
		}
		if len(chunk.Choices) == 0 {
			continue
		}

		if chunk.Usage != nil {
			usage = toTokenUsage(chunk.Usage)
		}

		deltaContent := chunk.Choices[0].Delta.Content
		deltaReasoning := chunk.Choices[0].Delta.ReasoningContent
		if deltaReasoning == "" {
			deltaReasoning = chunk.Choices[0].Delta.Reasoning
		}

		// Some providers send final data in message, not delta.
		if deltaContent == "" && chunk.Choices[0].Message.Content != "" {
			deltaContent = chunk.Choices[0].Message.Content
		}
		if deltaReasoning == "" && chunk.Choices[0].Message.ReasoningContent != "" {
			deltaReasoning = chunk.Choices[0].Message.ReasoningContent
		}

		if deltaContent != "" {
			fullContent.WriteString(deltaContent)
		}
		if deltaReasoning != "" {
			fullReasoning.WriteString(deltaReasoning)
		}

		streamContent, streamReasoning := splitReasoningAndContent(
			fullContent.String(),
			fullReasoning.String(),
		)

		contentDeltaOut := ""
		reasoningDeltaOut := ""

		if len(streamContent) >= emittedContentLen {
			contentDeltaOut = streamContent[emittedContentLen:]
		} else {
			contentDeltaOut = streamContent
			emittedContentLen = 0
		}
		if len(streamReasoning) >= emittedReasoningLen {
			reasoningDeltaOut = streamReasoning[emittedReasoningLen:]
		} else {
			reasoningDeltaOut = streamReasoning
			emittedReasoningLen = 0
		}

		emittedContentLen = len(streamContent)
		emittedReasoningLen = len(streamReasoning)

		if onDelta != nil && (contentDeltaOut != "" || reasoningDeltaOut != "") {
			if err := onDelta(model.AssistantReplyDelta{
				Content:          contentDeltaOut,
				ReasoningContent: reasoningDeltaOut,
			}); err != nil {
				return model.AssistantReply{}, err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return model.AssistantReply{}, fmt.Errorf("read upstream stream: %w", err)
	}

	content, reasoning := splitReasoningAndContent(fullContent.String(), fullReasoning.String())
	if content == "" && reasoning == "" {
		return model.AssistantReply{}, fmt.Errorf("upstream stream returned empty reply")
	}

	return model.AssistantReply{
		Content:          content,
		ReasoningContent: reasoning,
		Usage:            usage,
	}, nil
}

func (c *OpenAICompatibleClient) streamDeepSearchFallback(
	ctx context.Context,
	userMessage string,
	onDelta func(model.AssistantReplyDelta) error,
) (model.AssistantReply, error) {
	pushReasoning := func(text string) {
		if onDelta == nil {
			return
		}
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			return
		}
		_ = onDelta(model.AssistantReplyDelta{
			ReasoningContent: trimmed + "\n\n",
		})
	}

	pushReasoning("已开启联网搜索，正在联网检索可用信息...")
	reply, err := c.generateWithToolLoop(ctx, userMessage, pushReasoning)
	if err != nil {
		return model.AssistantReply{}, err
	}
	emitDeltaSlowly := func(text string, isReasoning bool) {
		if onDelta == nil {
			return
		}
		chunkSize := 18
		delay := 20 * time.Millisecond
		if isReasoning {
			chunkSize = 24
			delay = 24 * time.Millisecond
		}
		chunks := chunkTextByRune(text, chunkSize)
		for _, item := range chunks {
			if isReasoning {
				_ = onDelta(model.AssistantReplyDelta{ReasoningContent: item})
			} else {
				_ = onDelta(model.AssistantReplyDelta{Content: item})
			}
			time.Sleep(delay)
		}
	}

	if strings.TrimSpace(reply.ReasoningContent) != "" {
		emitDeltaSlowly(reply.ReasoningContent, true)
	}
	if strings.TrimSpace(reply.Content) != "" {
		emitDeltaSlowly(reply.Content, false)
	}
	return reply, nil
}

func (c *OpenAICompatibleClient) generateWithToolLoop(
	ctx context.Context,
	userMessage string,
	onProgress func(text string),
) (model.AssistantReply, error) {
	if c.searchClient == nil {
		return model.AssistantReply{}, fmt.Errorf("web search is not configured")
	}

	messages := []chatMessage{
		{Role: "user", Content: attachLocalDateContextIfNeeded(userMessage)},
	}
	reasoningLogs := make([]string, 0, 6)
	tools := []chatTool{deepSearchToolDef()}
	searchQueryCounter := map[string]int{}
	reportProgress := func(text string) {
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			return
		}
		if onProgress != nil {
			onProgress(trimmed)
		}
		reasoningLogs = append(reasoningLogs, trimmed)
	}

	for step := 0; step < maxToolLoopSteps; step++ {
		reportProgress(fmt.Sprintf("第 %d 轮：模型分析中...", step+1))
		decoded, err := c.executeChatCompletionWithHeartbeat(ctx, messages, tools, step+1, reportProgress)
		if err != nil {
			return model.AssistantReply{}, err
		}
		message := decoded.Choices[0].Message
		content, reasoning := splitReasoningAndContent(
			message.Content,
			message.ReasoningContent,
		)
		reply := model.AssistantReply{
			Content:          content,
			ReasoningContent: reasoning,
			Usage:            toTokenUsage(decoded.Usage),
		}

		toolCalls := message.ToolCalls
		if len(toolCalls) == 0 {
			if len(reasoningLogs) == 0 && strings.TrimSpace(reply.ReasoningContent) == "" {
				return reply, nil
			}
			if onProgress != nil {
				// 流式分支已经实时推送过阶段进度，这里只返回模型最终原始内容，避免重复大段回灌。
				return reply, nil
			}
			combinedReasoning := strings.TrimSpace(strings.Join(append(reasoningLogs, reply.ReasoningContent), "\n\n"))
			reply.ReasoningContent = combinedReasoning
			return reply, nil
		}

		// 到达最大轮次时，强制收敛为“直接回答”，避免继续循环触发 exceeded limit。
		if step == maxToolLoopSteps-1 {
			reportProgress("已达到搜索轮次上限，正在基于已检索信息生成最终回答...")
			return c.forceFinalAnswerWithoutTools(ctx, messages, reasoningLogs, onProgress != nil)
		}

		assistantMsg := chatMessage{
			Role:      "assistant",
			Content:   message.Content,
			ToolCalls: toolCalls,
		}
		if strings.TrimSpace(reply.ReasoningContent) != "" {
			assistantMsg.ReasoningContent = reply.ReasoningContent
			reportProgress(reply.ReasoningContent)
		}
		messages = append(messages, assistantMsg)

		for _, toolCall := range toolCalls {
			if strings.TrimSpace(toolCall.Function.Arguments) != "" {
				reportProgress("模型请求调用联网搜索工具，开始检索...")
			}
			if query := extractSearchQuery(toolCall); query != "" {
				searchQueryCounter[query]++
				if searchQueryCounter[query] > maxSameQueryRepeats {
					reportProgress("检测到重复检索，正在停止继续搜索并直接汇总回答...")
					return c.forceFinalAnswerWithoutTools(ctx, messages, reasoningLogs, onProgress != nil)
				}
			}
			resultText, summaryText, err := c.executeWebSearchToolWithHeartbeat(ctx, toolCall, reportProgress)
			if err != nil {
				return model.AssistantReply{}, err
			}
			reportProgress(summaryText)
			messages = append(messages, chatMessage{
				Role:       "tool",
				ToolCallID: toolCall.ID,
				Content:    resultText,
			})
		}
	}
	// 理论上不会走到这里，兜底时也尽量返回可用结果。
	return c.forceFinalAnswerWithoutTools(ctx, messages, reasoningLogs, onProgress != nil)
}

func (c *OpenAICompatibleClient) executeChatCompletionWithHeartbeat(
	ctx context.Context,
	messages []chatMessage,
	tools []chatTool,
	step int,
	reportProgress func(text string),
) (chatResponse, error) {
	if reportProgress == nil {
		return c.executeChatCompletion(ctx, messages, tools)
	}
	type completionResult struct {
		resp chatResponse
		err  error
	}
	resultCh := make(chan completionResult, 1)
	startedAt := time.Now()
	go func() {
		resp, err := c.executeChatCompletion(ctx, messages, tools)
		resultCh <- completionResult{resp: resp, err: err}
	}()

	ticker := time.NewTicker(deepSearchHeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case result := <-resultCh:
			return result.resp, result.err
		case <-ticker.C:
			waitedSeconds := int(time.Since(startedAt).Seconds())
			reportProgress(fmt.Sprintf("第 %d 轮：模型分析中（已等待 %d 秒）...", step, waitedSeconds))
		case <-ctx.Done():
			return chatResponse{}, ctx.Err()
		}
	}
}

func (c *OpenAICompatibleClient) executeWebSearchToolWithHeartbeat(
	ctx context.Context,
	toolCall chatToolCall,
	reportProgress func(text string),
) (string, string, error) {
	if reportProgress == nil {
		return c.executeWebSearchTool(ctx, toolCall)
	}
	type searchResult struct {
		raw     string
		summary string
		err     error
	}
	resultCh := make(chan searchResult, 1)
	startedAt := time.Now()
	go func() {
		raw, summary, err := c.executeWebSearchTool(ctx, toolCall)
		resultCh <- searchResult{raw: raw, summary: summary, err: err}
	}()

	ticker := time.NewTicker(deepSearchHeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case result := <-resultCh:
			return result.raw, result.summary, result.err
		case <-ticker.C:
			waitedSeconds := int(time.Since(startedAt).Seconds())
			reportProgress(fmt.Sprintf("联网检索进行中（已等待 %d 秒）...", waitedSeconds))
		case <-ctx.Done():
			return "", "", ctx.Err()
		}
	}
}

func (c *OpenAICompatibleClient) forceFinalAnswerWithoutTools(
	ctx context.Context,
	messages []chatMessage,
	reasoningLogs []string,
	isStreaming bool,
) (model.AssistantReply, error) {
	finalMessages := append([]chatMessage{}, messages...)
	finalMessages = append(finalMessages, chatMessage{
		Role:    "user",
		Content: "请停止继续调用任何工具。基于已有检索结果，直接输出最终答案，并清晰给出关键依据。",
	})
	finalReply, err := c.generateOnce(ctx, finalMessages, nil)
	if err != nil {
		return model.AssistantReply{}, err
	}
	if isStreaming {
		return finalReply, nil
	}
	if len(reasoningLogs) == 0 {
		return finalReply, nil
	}
	finalReply.ReasoningContent = strings.TrimSpace(
		strings.Join(append(reasoningLogs, finalReply.ReasoningContent), "\n\n"),
	)
	return finalReply, nil
}

func extractSearchQuery(toolCall chatToolCall) string {
	if strings.TrimSpace(toolCall.Function.Name) != webSearchToolName {
		return ""
	}
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(args.Query))
}

func chunkTextByRune(text string, chunkSize int) []string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}
	if chunkSize <= 0 {
		return []string{trimmed}
	}
	runes := []rune(trimmed)
	if len(runes) <= chunkSize {
		return []string{trimmed}
	}
	chunks := make([]string, 0, (len(runes)/chunkSize)+1)
	for start := 0; start < len(runes); start += chunkSize {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
	}
	return chunks
}

func (c *OpenAICompatibleClient) executeWebSearchTool(
	ctx context.Context,
	toolCall chatToolCall,
) (string, string, error) {
	if strings.TrimSpace(toolCall.Function.Name) != webSearchToolName {
		return "", "", fmt.Errorf("unsupported tool call: %s", toolCall.Function.Name)
	}

	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return "", "", fmt.Errorf("decode web_search arguments: %w", err)
	}
	query := strings.TrimSpace(args.Query)
	if query == "" {
		return "", "", fmt.Errorf("web_search query is empty")
	}
	queryForSearch := query
	if dateSensitiveQueryPattern.MatchString(query) {
		queryForSearch = enrichDateSensitiveSearchQuery(query)
	}

	results, err := c.searchClient.Search(ctx, queryForSearch, c.searchMaxResults)
	if err != nil {
		return "", "", fmt.Errorf("web search failed: %w", err)
	}
	if len(results) == 0 {
		return `{"results":[]}`, fmt.Sprintf("联网搜索关键词：%s\n检索结果为空。", queryForSearch), nil
	}
	for idx := range results {
		if len([]rune(results[idx].Content)) > 800 {
			results[idx].Content = string([]rune(results[idx].Content)[:800]) + "..."
		}
	}

	payload := map[string]any{
		"query":   queryForSearch,
		"results": results,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("encode web search result: %w", err)
	}

	var summary strings.Builder
	summary.WriteString("联网搜索关键词：")
	summary.WriteString(queryForSearch)
	summary.WriteString("\n已检索到以下来源：")
	for idx, item := range results {
		if idx >= 5 {
			break
		}
		summary.WriteString(fmt.Sprintf("\n%d. %s (%s)", idx+1, item.Title, item.URL))
	}
	return string(data), summary.String(), nil
}

func shouldAnswerWithLocalDate(userMessage string) bool {
	currentQuestion := extractCurrentQuestion(userMessage)
	trimmed := strings.TrimSpace(currentQuestion)
	if trimmed == "" {
		return false
	}

	normalized := strings.ToLower(strings.Join(strings.Fields(trimmed), " "))

	// 明确问日期/时间本身，直接返回本地时间。
	if explicitDateTimeQuestionPattern.MatchString(normalized) {
		return true
	}

	// 更严格的兜底：仅匹配“短问句 + 纯日期时间意图”。
	if len([]rune(normalized)) > 24 {
		return false
	}
	sanitized := strings.NewReplacer("，", "", "。", "", "？", "", "?", "", "！", "", "!", "", "：", "", ":", "").Replace(normalized)
	return strictDateTimeQuestionPattern.MatchString(strings.TrimSpace(sanitized))
}

func buildLocalDateAnswer() string {
	now := time.Now()
	zoneName, zoneOffset := now.Zone()
	weekdayMap := map[time.Weekday]string{
		time.Sunday:    "星期日",
		time.Monday:    "星期一",
		time.Tuesday:   "星期二",
		time.Wednesday: "星期三",
		time.Thursday:  "星期四",
		time.Friday:    "星期五",
		time.Saturday:  "星期六",
	}
	weekday := weekdayMap[now.Weekday()]
	if weekday == "" {
		weekday = now.Weekday().String()
	}
	offsetHours := zoneOffset / 3600
	offsetMinutes := (zoneOffset % 3600) / 60
	return fmt.Sprintf(
		"当前本地日期时间：%s（%s，UTC%+03d:%02d）。今天是%s。",
		now.Format("2006-01-02 15:04:05"),
		zoneName,
		offsetHours,
		absInt(offsetMinutes),
		weekday,
	)
}

func attachLocalDateContextIfNeeded(userMessage string) string {
	currentQuestion := extractCurrentQuestion(userMessage)
	if !dateSensitiveQueryPattern.MatchString(currentQuestion) {
		return userMessage
	}
	now := time.Now()
	zoneName, zoneOffset := now.Zone()
	return fmt.Sprintf(
		"%s\n\n[系统提供的本地日期参考]\n当前本地日期时间：%s（%s，UTC%+03d:%02d）。若问题涉及“今天/最近/最新/本周”等相对时间，请严格以此为准。",
		userMessage,
		now.Format("2006-01-02 15:04:05"),
		zoneName,
		zoneOffset/3600,
		absInt((zoneOffset%3600)/60),
	)
}

func extractCurrentQuestion(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	const currentQuestionMarker = "[当前问题]"
	markerIndex := strings.LastIndex(trimmed, currentQuestionMarker)
	if markerIndex == -1 {
		return trimmed
	}
	afterMarker := strings.TrimSpace(trimmed[markerIndex+len(currentQuestionMarker):])
	if afterMarker == "" {
		return trimmed
	}
	return afterMarker
}

func enrichDateSensitiveSearchQuery(query string) string {
	now := time.Now()
	zoneName, zoneOffset := now.Zone()
	return fmt.Sprintf(
		"%s（当前本地日期时间：%s，时区：%s，UTC%+03d:%02d）",
		query,
		now.Format("2006-01-02 15:04:05"),
		zoneName,
		zoneOffset/3600,
		absInt((zoneOffset%3600)/60),
	)
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func deepSearchToolDef() chatTool {
	return chatTool{
		Type: "function",
		Function: chatToolFunction{
			Name:        webSearchToolName,
			Description: "联网搜索实时信息，返回标题、链接与摘要。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "要检索的关键词",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

func toolChoiceValue(tools []chatTool) string {
	if len(tools) == 0 {
		return ""
	}
	return "auto"
}

func toTokenUsage(raw *struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}) *model.TokenUsage {
	if raw == nil {
		return nil
	}
	return &model.TokenUsage{
		PromptTokens:     raw.PromptTokens,
		CompletionTokens: raw.CompletionTokens,
		TotalTokens:      raw.TotalTokens,
	}
}

func splitReasoningAndContent(content, reasoning string) (string, string) {
	cleanContent := strings.TrimSpace(content)
	cleanReasoning := strings.TrimSpace(reasoning)
	parts := make([]string, 0, 4)
	if cleanReasoning != "" {
		parts = append(parts, cleanReasoning)
	}

	firstThinkTagIndex := strings.Index(cleanContent, "<think>")
	lastThinkEndTagIndex := strings.LastIndex(cleanContent, "</think>")

	if firstThinkTagIndex != -1 && lastThinkEndTagIndex > firstThinkTagIndex {
		thinkBlockStart := firstThinkTagIndex + len("<think>")
		thinkBlock := cleanContent[thinkBlockStart:lastThinkEndTagIndex]
		thinkBlock = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(thinkBlock), "<think>"))
		thinkBlock = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(thinkBlock), "</think>"))
		if thinkBlock != "" {
			parts = append(parts, thinkBlock)
		}
		cleanContent = strings.TrimSpace(
			cleanContent[:firstThinkTagIndex] + cleanContent[lastThinkEndTagIndex+len("</think>"):],
		)
	} else if firstThinkTagIndex != -1 {
		// Stream may still be inside an unfinished <think> block.
		partialThink := strings.TrimSpace(cleanContent[firstThinkTagIndex+len("<think>"):])
		partialThink = strings.TrimSpace(strings.TrimPrefix(partialThink, "<think>"))
		if partialThink != "" {
			parts = append(parts, partialThink)
		}
		cleanContent = strings.TrimSpace(cleanContent[:firstThinkTagIndex])
	} else {
		matches := thinkTagPattern.FindAllStringSubmatch(cleanContent, -1)
		if len(matches) > 0 {
			for _, match := range matches {
				if len(match) < 2 {
					continue
				}
				segment := strings.TrimSpace(match[1])
				if segment != "" {
					parts = append(parts, segment)
				}
			}
			cleanContent = strings.TrimSpace(thinkTagPattern.ReplaceAllString(cleanContent, ""))
		}
	}

	cleanContent = strings.TrimSpace(
		strings.NewReplacer("<think>", "", "</think>", "").Replace(cleanContent),
	)

	if cleanReasoning == "" && len(parts) > 0 {
		deduped := make([]string, 0, len(parts))
		seen := make(map[string]struct{}, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			deduped = append(deduped, p)
		}
		cleanReasoning = strings.Join(deduped, "\n\n")
	}

	cleanContent = stripReasoningPrefix(cleanContent, cleanReasoning)

	return cleanContent, cleanReasoning
}

func stripReasoningPrefix(content, reasoning string) string {
	content = strings.TrimSpace(content)
	reasoning = strings.TrimSpace(reasoning)
	if content == "" || reasoning == "" {
		return content
	}

	if strings.HasPrefix(content, reasoning) {
		return strings.TrimSpace(content[len(reasoning):])
	}

	normalizedReasoning := normalizeSpaces(reasoning)
	if normalizedReasoning == "" {
		return content
	}

	normalizedContent := normalizeSpaces(content)
	if normalizedContent == normalizedReasoning {
		return ""
	}
	if strings.HasPrefix(normalizedContent, normalizedReasoning) {
		contentFields := strings.Fields(content)
		reasoningFields := strings.Fields(reasoning)
		if len(contentFields) >= len(reasoningFields) {
			prefixMatched := true
			for i := range reasoningFields {
				if contentFields[i] != reasoningFields[i] {
					prefixMatched = false
					break
				}
			}
			if prefixMatched {
				return strings.TrimSpace(strings.Join(contentFields[len(reasoningFields):], " "))
			}
		}
	}

	return content
}

func normalizeSpaces(input string) string {
	return strings.Join(strings.Fields(input), " ")
}
