import { describe, expect, it } from "vitest";
import { parseSseChunk } from "./chatApi";

describe("chat api sse parser", () => {
  it("parses data lines and ignores empty lines", () => {
    const events = parseSseChunk('data: {"summary":"ok"}\n\n\n');
    expect(events).toEqual([{ summary: "ok" }]);
  });

  it("keeps raw payload when json parse fails", () => {
    const events = parseSseChunk("data: plain text\n\n");
    expect(events).toEqual([{ raw: "plain text" }]);
  });
});
