import { create } from "zustand";

export type AppEnv = "development" | "production";

interface AppState {
  version: string;
  backendReachable: boolean;
  backendMessage: string;
  setVersion: (v: string) => void;
  setBackendStatus: (reachable: boolean, message: string) => void;
}

export const useAppStore = create<AppState>((set) => ({
  version: "0.1.0",
  backendReachable: false,
  backendMessage: "Checking…",
  setVersion: (version) => set({ version }),
  setBackendStatus: (backendReachable, backendMessage) =>
    set({ backendReachable, backendMessage }),
}));
