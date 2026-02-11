import { create } from "zustand";
import { StartTunnel, StopTunnel } from "../../wailsjs/go/main/App"; // Adjust path as needed

type TunnelStatus = "disconnected" | "connecting" | "connected" | "error";

interface TunnelState {
  status: TunnelStatus;
  logs: string[];
  ip: string;
  setIp: (ip: string) => void;
  port: string;
  setPort: (port: string) => void;
  addLog: (message: string) => void;
  setStatus: (status: TunnelStatus) => void;
  startHost: () => Promise<void>;
  stopHost: () => Promise<void>;
}

export const useTunnelStore = create<TunnelState>((set, get) => ({
  status: "disconnected",
  logs: [],
  ip: "localhost", // Default or saved pref
  setIp: (ip) => set({ ip }),
  port: "25345",
  setPort: (port) => set({ port }),
  addLog: (message) => set((state) => ({ logs: [...state.logs, message] })),
  setStatus: (status) => set({ status }),
  startHost: async () => {
    const { ip, port } = get();
    set({ status: "connecting", logs: [] }); // Clear logs on new run?
    try {
      // Calls the Go backend
      await StartTunnel(ip, "25565");
    } catch (err: any) {
      set({ status: "error" });
      get().addLog(`Error: ${err.message || err}`);
    }
  },
  stopHost: async () => {
    await StopTunnel();
    set({ status: "disconnected" });
  },
}));
