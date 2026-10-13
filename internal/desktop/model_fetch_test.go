package desktop

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yangk/kshell/internal/config"
)

func setFetchModelConfig(t *testing.T, app *App, oai, ant string) {
	t.Helper()
	app.mu.Lock()
	app.opts.Config.Model = config.ModelConfig{
		Enabled: true, OpenAIBaseURL: oai, AnthropicBaseURL: ant, APIKey: "k",
	}
	app.mu.Unlock()
}

func TestFetchAvailableModelsMergesEndpoints(t *testing.T) {
	var oaiPath, antPath string
	oai := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		oaiPath = r.URL.Path
		if got := r.Header.Get("Authorization"); got != "Bearer k" {
			t.Errorf("OpenAI 鉴权头 = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": "m-b"}, {"id": "m-a"}}})
	}))
	defer oai.Close()
	ant := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		antPath = r.URL.Path
		if got := r.Header.Get("x-api-key"); got != "k" {
			t.Errorf("Anthropic 鉴权头 = %q", got)
		}
		if got := r.Header.Get("anthropic-version"); got == "" {
			t.Error("缺 anthropic-version 头")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": "m-a"}, {"id": "m-c"}}})
	}))
	defer ant.Close()

	app, _ := newModelApp(t)
	setFetchModelConfig(t, app, oai.URL, ant.URL)
	v, err := app.FetchAvailableModels()
	if err != nil {
		t.Fatalf("FetchAvailableModels: %v", err)
	}
	if oaiPath != "/models" {
		t.Fatalf("OpenAI 路径 = %q", oaiPath)
	}
	if antPath != "/v1/models" {
		t.Fatalf("Anthropic 路径 = %q", antPath)
	}
	if len(v.Errors) != 0 {
		t.Fatalf("不应有错误: %v", v.Errors)
	}
	if strings.Join(v.Models, ",") != "m-a,m-b,m-c" {
		t.Fatalf("合并去重排序失败: %#v", v.Models)
	}
}

func TestFetchAvailableModelsPartialFailure(t *testing.T) {
	oai := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer oai.Close()
	ant := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": "m-c"}}})
	}))
	defer ant.Close()

	app, _ := newModelApp(t)
	setFetchModelConfig(t, app, oai.URL, ant.URL)
	v, err := app.FetchAvailableModels()
	if err != nil {
		t.Fatalf("单端失败不应整体报错: %v", err)
	}
	if strings.Join(v.Models, ",") != "m-c" {
		t.Fatalf("应仅保留成功端结果: %#v", v.Models)
	}
	if len(v.Errors) != 1 || !strings.Contains(v.Errors[0], "openai") {
		t.Fatalf("应记录 openai 失败 key: %v", v.Errors)
	}
}

func TestFetchAvailableModelsNoEndpoint(t *testing.T) {
	app, _ := newModelApp(t)
	setFetchModelConfig(t, app, "", "")
	if _, err := app.FetchAvailableModels(); err == nil {
		t.Fatal("两端都未配置应报错")
	}
}
