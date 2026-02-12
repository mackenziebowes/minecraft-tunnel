import { create } from "zustand";
import {
  CreateOffer,
  AcceptOffer,
  AcceptAnswer,
  StartHostProxy,
  StartJoinerProxy,
  ExportToFile,
  ImportFromFile,
} from "../../wailsjs/go/main/App";
import { useToastStore } from "./toastStore";

type TunnelStatus = "disconnected" | "connecting" | "connected" | "error" | "waiting-for-answer" | "waiting-for-host";

interface TunnelState {
  // State
  status: TunnelStatus;
  logs: string[];
  offerToken: string;
  answerToken: string;
  mcServerAddress: string;
  proxyPort: string;

  // Actions
  setMcServerAddress: (address: string) => void;
  setProxyPort: (port: string) => void;
  generateOffer: () => Promise<void>;
  acceptOffer: (offer: string) => Promise<void>;
  acceptAnswer: (answer: string) => Promise<void>;
  exportToken: (token: string) => Promise<void>;
  importToken: () => Promise<string | undefined>;
  addLog: (message: string) => void;
  setStatus: (status: TunnelStatus) => void;
}

export const useTunnelStore = create<TunnelState>((set, get) => ({
  status: "disconnected",
  logs: [],
  offerToken: "",
  answerToken: "",
  mcServerAddress: "localhost:25565",
  proxyPort: "25565",

  setMcServerAddress: (address) => set({ mcServerAddress: address }),
  setProxyPort: (port) => set({ proxyPort: port }),

  generateOffer: async () => {
    set({ status: "connecting", logs: [], offerToken: "" });
    try {
      const token = await CreateOffer();
      set({ status: "waiting-for-answer", offerToken: token });
      get().addLog("Offer token generated successfully");
    } catch (err: any) {
      set({ status: "error" });
      get().addLog(`Error: ${err.message || err}`);
      useToastStore.getState().addToast({
        title: "Failed to generate offer",
        description: err.message || "An unknown error occurred",
        variant: "destructive",
      });
    }
  },

  acceptOffer: async (offer) => {
    set({ status: "connecting", logs: [] });
    try {
      const answer = await AcceptOffer(offer);
      set({ status: "waiting-for-host", answerToken: answer });
      get().addLog("Answer generated - share this with host");
    } catch (err: any) {
      set({ status: "error" });
      get().addLog(`Error: ${err.message || err}`);
      useToastStore.getState().addToast({
        title: "Failed to accept offer",
        description: err.message || "An unknown error occurred",
        variant: "destructive",
      });
    }
  },

  acceptAnswer: async (answer) => {
    try {
      await AcceptAnswer(answer);
      set({ status: "connected" });
      get().addLog("Tunnel established!");
    } catch (err: any) {
      set({ status: "error" });
      get().addLog(`Error: ${err.message || err}`);
      useToastStore.getState().addToast({
        title: "Failed to accept answer",
        description: err.message || "An unknown error occurred",
        variant: "destructive",
      });
    }
  },

  exportToken: async (token) => {
    try {
      const path = await (window as any).runtime.SaveFileDialog();
      if (path) {
        await ExportToFile(token, path);
        get().addLog(`Token exported to ${path}`);
      }
    } catch (err: any) {
      get().addLog(`Error exporting: ${err.message || err}`);
      useToastStore.getState().addToast({
        title: "Failed to export token",
        description: err.message || "An unknown error occurred",
        variant: "destructive",
      });
    }
  },

  importToken: async () => {
    try {
      const path = await (window as any).runtime.OpenFileDialog();
      if (path) {
        const token = await ImportFromFile(path);
        get().addLog(`Token imported from ${path}`);
        return token;
      }
    } catch (err: any) {
      get().addLog(`Error importing: ${err.message || err}`);
      useToastStore.getState().addToast({
        title: "Failed to import token",
        description: err.message || "An unknown error occurred",
        variant: "destructive",
      });
    }
  },

  addLog: (message) =>
    set((state) => ({ logs: [...state.logs, message] })),
  setStatus: (status) => set({ status }),
}));
