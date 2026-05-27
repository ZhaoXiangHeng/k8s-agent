export type Permission = {
  id?: string;
  namespace: string;
  apiGroup: string;
  resource: string;
  verbs: string[];
  enabled?: boolean;
};

export type PermissionFormRow = {
  namespace: string;
  apiGroup: string;
  resource: string;
  verbsText: string;
};

export type UpdatePermissionsRequest = {
  permissions: Array<{
    namespace: string;
    apiGroup: string;
    resource: string;
    verbs: string[];
  }>;
};

export function parseVerbs(value: string): string[] {
  return value
    .split(",")
    .map((verb) => verb.trim())
    .filter(Boolean);
}

export function buildPermissionPayload(rows: PermissionFormRow[]): UpdatePermissionsRequest {
  return {
    permissions: rows
      .filter((row) => row.namespace.trim() && row.resource.trim())
      .map((row) => ({
        namespace: row.namespace.trim(),
        apiGroup: row.apiGroup.trim(),
        resource: row.resource.trim(),
        verbs: parseVerbs(row.verbsText)
      }))
  };
}
