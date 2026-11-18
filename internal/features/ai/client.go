package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrEmptyCompletion dikembalikan bila OpenAI tidak mengirim pilihan jawaban.
var ErrEmptyCompletion = errors.New("respons OpenAI kosong")

const maxBody = 4 << 20

// doJSON mengirim request HTTP berisi payload JSON dan men-decode balasan ke out.
func doJSON(ctx context.Context, hc *http.Client, method, target, key string, payload []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("layanan eksternal membalas status %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(out)
}

// ChatClient memanggil OpenAI Chat Completions.
type ChatClient struct {
	Endpoint string
	Key      string
	Model    string
	HTTP     *http.Client
}

// NewChatClient membuat ChatClient dengan timeout 90 detik.
func NewChatClient(endpoint, key, model string) *ChatClient {
	return &ChatClient{Endpoint: endpoint, Key: key, Model: model, HTTP: &http.Client{Timeout: 90 * time.Second}}
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

// Complete mengirim pesan ke {endpoint}/chat/completions dan mengembalikan teks jawaban.
func (c *ChatClient) Complete(ctx context.Context, messages []Message) (string, error) {
	// Struct berisi string/angka saja, sehingga Marshal tidak mungkin gagal.
	payload, _ := json.Marshal(chatRequest{Model: c.Model, Messages: messages, Temperature: 0.7})
	var out chatResponse
	target := strings.TrimRight(c.Endpoint, "/") + "/chat/completions"
	if err := doJSON(ctx, c.HTTP, http.MethodPost, target, c.Key, payload, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", ErrEmptyCompletion
	}
	return out.Choices[0].Message.Content, nil
}
