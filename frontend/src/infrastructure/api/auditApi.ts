import type { AuditLog } from "../../domain/audit";
import { apiRequest, type ApiAuth } from "./client";

export function listAuditLogs(auth: ApiAuth): Promise<AuditLog[]> {
  return apiRequest<AuditLog[]>("/api/admin/audit-logs", { auth });
}
