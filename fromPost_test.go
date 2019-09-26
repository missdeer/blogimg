package main

import "testing"

func TestExtractFromPostContent(t *testing.T) {
	result, err := extractFromPostContent(`2019-09-17-a-new-markdown-editor.md`)
	if err != nil {
		t.Error(err)
	}
	t.Log(result)

	result, err = extractFromPostContent(`2018-04-27-access-internal-network-seamless.md`)
	if err != nil {
		t.Error(err)
	}
	t.Log(result)
}
