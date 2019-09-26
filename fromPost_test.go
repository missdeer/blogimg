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

	result, err = extractFromPostContent(`2004-12-11-%E4%BB%8A%E5%A4%A9%E4%B8%8A%E7%BD%91%E8%AE%A2%E4%BA%86%E5%A5%97%E4%B8%89%E5%8D%B7%E6%9C%AC%E7%9A%84%E3%80%8ATCP%2FIP%E8%AF%A6%E8%A7%A3%E3%80%8B.md`)
	if err != nil {
		t.Log(err)
	} else {
		t.Error(result)
	}
}
