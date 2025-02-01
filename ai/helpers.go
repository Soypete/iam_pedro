package ai

import "strings"

// CleanResponse removes any newlines from the response
func CleanResponse(resp string) string {
	// remove any newlines
	resp = strings.ReplaceAll(resp, "\n", " ")
	resp = strings.ReplaceAll(resp, "<|im_start|>", "")
	resp = strings.ReplaceAll(resp, "<|im_end|>", "")
	resp = strings.TrimPrefix(resp, "!") // remove any leading ! so that we dont trigger commands
	resp = strings.TrimPrefix(resp, "/") // remove any leading / so that we dont trigger commands
	return strings.TrimSpace(resp)
}
