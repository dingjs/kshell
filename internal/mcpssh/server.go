package mcpssh

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type rpcReq struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// Handle 处理一条 MCP JSON-RPC 消息。通知（无 id）返回 ok=false，调用方不回包。
func Handle(msg []byte, tools *Tools) ([]byte, bool) {
	var req rpcReq
	if err := json.Unmarshal(msg, &req); err != nil {
		return rpcResult(nil, nil, err), true
	}
	if len(req.ID) == 0 {
		return nil, false
	}
	switch req.Method {
	case "initialize":
		return rpcResult(req.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "kshell-mcp-ssh", "version": "0.1.0"},
		}, nil), true
	case "tools/list":
		defs := tools.ToolDefs()
		out := make([]any, len(defs))
		for i, d := range defs {
			out[i] = d
		}
		return rpcResult(req.ID, map[string]any{"tools": out}, nil), true
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return rpcResult(req.ID, nil, err), true
		}
		text, isError, err := tools.Call(context.Background(), p.Name, p.Arguments)
		if err != nil && !isError {
			return rpcResult(req.ID, nil, err), true
		}
		if isError {
			msg := err.Error()
			if err == nil {
				msg = "tool error"
			}
			return rpcResult(req.ID, map[string]any{
				"content": []any{map[string]string{"type": "text", "text": msg}},
				"isError": true,
			}, nil), true
		}
		return rpcResult(req.ID, map[string]any{
			"content": []any{map[string]string{"type": "text", "text": text}},
		}, nil), true
	default:
		return rpcResult(req.ID, nil, fmt.Errorf("unknown method %s", req.Method)), true
	}
}

func rpcResult(id json.RawMessage, result any, err error) []byte {
	env := map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(id)}
	if err != nil {
		env["error"] = map[string]any{"code": -32000, "message": err.Error()}
	} else {
		env["result"] = result
	}
	b, _ := json.Marshal(env)
	return b
}

// ServeIO 按 Content-Length 帧读写 stdio，直到输入结束。
func ServeIO(r io.Reader, w io.Writer, tools *Tools) error {
	br := bufio.NewReader(r)
	for {
		msg, err := readFrame(br)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		resp, ok := Handle(msg, tools)
		if !ok {
			continue
		}
		if err := writeFrame(w, resp); err != nil {
			return err
		}
	}
}

func readFrame(r *bufio.Reader) ([]byte, error) {
	var length int
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			n, err := strconv.Atoi(strings.TrimSpace(line[len("content-length:"):]))
			if err != nil {
				return nil, err
			}
			length = n
		}
	}
	if length < 0 {
		return nil, fmt.Errorf("bad content-length")
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func writeFrame(w io.Writer, body []byte) error {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Content-Length: %d\r\n\r\n", len(body))
	buf.Write(body)
	_, err := w.Write(buf.Bytes())
	return err
}
