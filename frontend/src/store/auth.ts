// Mock user accounts with passwords
export interface User {
  id: string;
  username: string;
  password: string;
  role: "admin" | "operator";
  displayName: string;
  email: string;
  notes: string;
  createdAt: string;
}

export interface Model {
  id: string;
  name: string;
  provider: string;
}

const defaultUsers: User[] = [
  {
    id: "user-admin",
    username: "admin",
    password: "admin123",
    role: "admin",
    displayName: "管理员",
    email: "admin@example.com",
    notes: "",
    createdAt: "2026-01-01",
  },
  {
    id: "user-operator",
    username: "operator",
    password: "operator123",
    role: "operator",
    displayName: "操作员",
    email: "operator@example.com",
    notes: "",
    createdAt: "2026-01-15",
  },
];

let users: User[] = [...defaultUsers];

// Per-user model bindings: userId -> Set<modelId>
const bindings = new Map<string, Set<string>>();
const defaultModel = new Map<string, string>();

// Default bindings: operator has gpt-4.1 and claude-sonnet-4-5
bindings.set("user-operator", new Set(["model-gpt4", "model-claude-sonnet"]));
defaultModel.set("user-operator", "model-gpt4");
bindings.set("user-admin", new Set(["model-gpt4", "model-claude-sonnet", "model-deepseek"]));

const allModels: Model[] = [
  { id: "model-gpt4", name: "gpt-4.1", provider: "OpenAI" },
  { id: "model-claude-sonnet", name: "claude-sonnet-4-5", provider: "Anthropic" },
  { id: "model-deepseek", name: "deepseek-v3", provider: "DeepSeek" },
];

export function login(username: string, password: string): User | null {
  const user = users.find((u) => u.username === username && u.password === password);
  return user ?? null;
}

export function listUsers(): User[] {
  return [...users];
}

export function createUser(
  username: string,
  password: string,
  role: "admin" | "operator",
  displayName: string,
  email: string
): User {
  const newUser: User = {
    id: `user-${Date.now()}`,
    username,
    password,
    role,
    displayName,
    email,
    notes: "",
    createdAt: new Date().toISOString().split("T")[0],
  };
  users.push(newUser);
  return newUser;
}

export function resetPassword(userId: string, newPassword: string): boolean {
  const user = users.find((u) => u.id === userId);
  if (!user) return false;
  user.password = newPassword;
  return true;
}

export function updateUser(
  userId: string,
  fields: { username?: string; role?: "admin" | "operator"; displayName?: string; email?: string; notes?: string }
): User | null {
  const user = users.find((u) => u.id === userId);
  if (!user) return null;
  if (fields.username !== undefined) user.username = fields.username;
  if (fields.role !== undefined) user.role = fields.role;
  if (fields.displayName !== undefined) user.displayName = fields.displayName;
  if (fields.email !== undefined) user.email = fields.email;
  if (fields.notes !== undefined) user.notes = fields.notes;
  return user;
}

export function getAllModels(): Model[] {
  return [...allModels];
}

export function getAssignedModels(userId: string): Model[] {
  const boundIds = bindings.get(userId);
  if (!boundIds) return [];
  return allModels.filter((m) => boundIds.has(m.id));
}

export function getAvailableModels(userId: string): Model[] {
  const boundIds = bindings.get(userId);
  if (!boundIds) return allModels;
  return allModels.filter((m) => !boundIds.has(m.id));
}

export function getDefaultModel(userId: string): string | undefined {
  return defaultModel.get(userId);
}

export function updateBindings(
  userId: string,
  add: string[],
  remove: string[],
  newDefault?: string
): void {
  let bound = bindings.get(userId);
  if (!bound) {
    bound = new Set<string>();
    bindings.set(userId, bound);
  }
  for (const id of add) bound.add(id);
  for (const id of remove) bound.delete(id);
  if (newDefault && bound.has(newDefault)) {
    defaultModel.set(userId, newDefault);
  }
}
