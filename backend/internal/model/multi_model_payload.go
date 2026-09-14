package model

import (
	"encoding/json"
	"strings"
)

const MultiModelPayloadKind = "multi_model_reply_v1"

type PersistedAssistantPayload struct {
	Kind          string                   `json:"kind"`
	SelectedModel string                   `json:"selected_model"`
	Responses     []ModelAssistantResponse `json:"responses"`
}

func BuildPersistedAssistantContent(
	selectedModel string,
	responses []ModelAssistantResponse,
) (string, error) {
	payload := PersistedAssistantPayload{
		Kind:          MultiModelPayloadKind,
		SelectedModel: strings.TrimSpace(selectedModel),
		Responses:     responses,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ParsePersistedAssistantContent(raw string) (PersistedAssistantPayload, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
		return PersistedAssistantPayload{}, false
	}
	var payload PersistedAssistantPayload
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return PersistedAssistantPayload{}, false
	}
	if payload.Kind != MultiModelPayloadKind || len(payload.Responses) == 0 {
		return PersistedAssistantPayload{}, false
	}
	return payload, true
}
