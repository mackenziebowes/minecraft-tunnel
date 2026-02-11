// store.ts
import { create } from "zustand";

export type ValidRoutes = "/" | "/host" | "/join";

// Define types for state & actions
interface AppState {
  route: ValidRoutes;
  setRoute: (newRoute: ValidRoutes) => void;
}

// Create store using the curried form of `create`
export const useAppStore = create<AppState>()((set) => ({
  route: "/",
  setRoute: (newRoute: ValidRoutes) => set(() => ({ route: newRoute })),
}));
