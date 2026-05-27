import { describe, expect, it } from "vitest";
import { buildPermissionPayload, parseVerbs } from "./permission";

describe("permission domain helpers", () => {
  it("splits comma separated verbs and removes blanks", () => {
    expect(parseVerbs("get, list,watch, ")).toEqual(["get", "list", "watch"]);
  });

  it("builds backend permission update payload", () => {
    const payload = buildPermissionPayload([
      {
        namespace: "dev",
        apiGroup: "",
        resource: "pods",
        verbsText: "get,list,watch"
      }
    ]);

    expect(payload).toEqual({
      permissions: [
        {
          namespace: "dev",
          apiGroup: "",
          resource: "pods",
          verbs: ["get", "list", "watch"]
        }
      ]
    });
  });
});
