package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// ModelsFetchView 是拉取模型列表的结果：Models 为双端合并去重升序；
// Errors 为失败端点的稳定 wire key（单端失败不影响另一端结果）。
type ModelsFetchView struct {
	Models []string `json:"Models"`
	Errors []string `json:"Errors"`
}

// fetchTimeout 拉取模型列表的单请求超时。
const fetchTimeout = 10 * time.Second

// FetchAvailableModels 从已保存配置的 OpenAI 兼容端点与 Anthropic 端点并发拉取
// 可用模型列表并合并去重。只信已保存配置（已过 URL 校验），不接受前端传 URL。
func (a *App) FetchAvailableModels() (ModelsFetchView, error) {
	m := a.snapshot().Config.Model
	oaiURL := strings.TrimRight(m.OpenAIBaseURL, "/")
	antURL := strings.TrimRight(m.AnthropicBaseURL, "/")
	if oaiURL == "" && antURL == "" {
		return ModelsFetchView{}, fmt.Errorf("err.model_fetch.no_endpoint")
	}

	var (
		mu     sync.Mutex
		models []string
		errs   []string
		wg     sync.WaitGroup
		seen   = map[string]bool{}
	)
	collect := func(list []string) {
		mu.Lock()
		for _, id := range list {
			if id != "" && !seen[id] {
				seen[id] = true
				models = append(models, id)
			}
		}
		mu.Unlock()
	}
	fail := func(key string) {
		mu.Lock()
		errs = append(errs, key)
		mu.Unlock()
	}

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	wg.Add(2)
	go func() {
		defer wg.Done()
		if oaiURL == "" {
			return
		}
		list, err := fetchOpenAIModels(ctx, oaiURL, m.APIKey)
		if err != nil {
			fail("err.model_fetch.openai_failed")
			return
		}
		collect(list)
	}()
	go func() {
		defer wg.Done()
		if antURL == "" {
			return
		}
		list, err := fetchAnthropicModels(ctx, antURL, m.APIKey)
		if err != nil {
			fail("err.model_fetch.anthropic_failed")
			return
		}
		collect(list)
	}()
	wg.Wait()

	sort.Strings(models)
	out := ModelsFetchView{Models: models, Errors: errs}
	if models == nil {
		out.Models = []string{}
	}
	if len(errs) > 0 && len(models) == 0 {
		return out, fmt.Errorf("err.model_fetch.all_failed")
	}
	return out, nil
}

// fetchOpenAIModels 请求 OpenAI 兼容端点的 GET {base}/models。
func fetchOpenAIModels(ctx context.Context, base, apiKey string) ([]string, error) {
	return fetchModelList(ctx, base+"/models", func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	})
}

// fetchAnthropicModels 请求 Anthropic 端点的 GET {base}/v1/models。
func fetchAnthropicModels(ctx context.Context, base, apiKey string) ([]string, error) {
	return fetchModelList(ctx, base+"/v1/models", func(req *http.Request) {
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	})
}

// fetchModelList 发起请求并解析 OpenAI 风格 {"data":[{"id":...}]} 响应。
func fetchModelList(ctx context.Context, url string, setHeaders func(*http.Request)) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	setHeaders(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&body); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(body.Data))
	for _, m := range body.Data {
		if id := strings.TrimSpace(m.ID); id != "" {
			out = append(out, id)
		}
	}
	return out, nil
}
