package handler

import "testing"

// TestListEventsToolNameMatchesDocumentedMCPTool 验证工具名称与文档一致。
func TestListEventsToolNameMatchesDocumentedMCPTool(t *testing.T) {
	tool := ListEventsTool()
	if tool.Name != "list_events" {
		t.Fatalf("expected documented tool name list_events, got %s", tool.Name)
	}
}
