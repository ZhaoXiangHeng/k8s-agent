export type User = {
  id: string;
  username: string;
  displayName: string;
  email?: string;
  role: "admin" | "operator";
  status: string;
};

export type CreateUserRequest = {
  username: string;
  email: string;
  role: "admin" | "operator";
  displayName: string;
  password: string;
};

export type LoginRequest = {
  username: string;
  password: string;
};
