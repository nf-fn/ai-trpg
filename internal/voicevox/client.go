package voicevox

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type Client interface {
	Synthesize(ctx context.Context, text string) ([]byte, error)
}

type client struct {
	url        string
	speaker    int
	httpClient *http.Client
}

func NewClient(baseURL string, speaker int) Client {
	return &client{
		url:        baseURL,
		speaker:    speaker,
		httpClient: &http.Client{},
	}
}

func (c *client) Synthesize(ctx context.Context, text string) ([]byte, error) {
	query, err := c.audioQuery(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("audio_query: %w", err)
	}

	wav, err := c.synthesis(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("synthesis: %w", err)
	}

	return wav, nil
}

func (c *client) audioQuery(ctx context.Context, text string) ([]byte, error) {
	params := url.Values{}
	params.Set("text", text)
	params.Set("speaker", strconv.Itoa(c.speaker))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/audio_query?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("audio_query returned %d: %s", resp.StatusCode, string(b))
	}

	return io.ReadAll(resp.Body)
}

func (c *client) synthesis(ctx context.Context, query []byte) ([]byte, error) {
	params := url.Values{}
	params.Set("speaker", strconv.Itoa(c.speaker))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/synthesis?"+params.Encode(), bytes.NewReader(query))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("synthesis returned %d: %s", resp.StatusCode, string(b))
	}

	return io.ReadAll(resp.Body)
}
