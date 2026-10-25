package application

import (
	"encoding/json"
	"strings"
)

// llmServiceReply answers like llm-service: the reply as it was written and,
// when the request expects JSON, its first JSON object; a reply with none is
// the malformed error llm-service would have returned.
func llmServiceReply(req ChatRequest, content string) (ChatResult, error) {
	res := ChatResult{Content: content, CallID: "call-test"}
	if !req.ExpectJSON {
		return res, nil
	}
	if start := strings.Index(content, "{"); start >= 0 {
		var raw json.RawMessage
		if json.NewDecoder(strings.NewReader(content[start:])).Decode(&raw) == nil {
			res.JSON = raw
			return res, nil
		}
	}
	return ChatResult{}, &LLMError{
		Kind: ErrKindMalformed, Provider: "llm-service", CallID: "call-test",
		Err: errNoJSON,
	}
}

var errNoJSON = errorString("khối JSON không đọc được")

type errorString string

func (e errorString) Error() string { return string(e) }
