export type AuditLog = {
  id: string;
  actorUserId: string;
  action: string;
  targetType: string;
  targetId: string;
  namespace?: string;
  resource?: string;
  verb?: string;
  allowed: boolean;
  reason: string;
  createdAt: string;
};
