import { create } from "zustand";
import {
  CreateOffer,
  AcceptOffer,
  AcceptAnswer,
  StartHostProxy,
  StartJoinerProxy,
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
  exportToken: (token: string, tokenType: 'offer' | 'answer') => Promise<void>;
  importToken: (file: File) => Promise<string | undefined>;
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
    console.log("[FRONTEND] generateOffer called");
    set({ status: "connecting", logs: [], offerToken: "" });
    try {
      console.log("[FRONTEND] Calling CreateOffer()...");
      const token = await CreateOffer();
      console.log("[FRONTEND] CreateOffer returned, token length:", token?.length);
      console.log("[FRONTEND] Token preview:", token?.substring(0, 50) + "...");
      set({ status: "waiting-for-answer", offerToken: token });
      get().addLog("Offer token generated successfully");
      console.log("[FRONTEND] State updated to waiting-for-answer");
    } catch (err: any) {
      console.error("[FRONTEND] CreateOffer error:", err);
      console.error("[FRONTEND] Error message:", err?.message);
      console.error("[FRONTEND] Error stack:", err?.stack);
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
    console.log("[FRONTEND] acceptOffer called, offer length:", offer?.length);
    set({ status: "connecting", logs: [] });
    try {
      console.log("[FRONTEND] Calling AcceptOffer()...");
      const answer = await AcceptOffer(offer);
      console.log("[FRONTEND] AcceptOffer returned, answer length:", answer?.length);
      set({ status: "waiting-for-host", answerToken: answer });
      get().addLog("Answer generated - share this with host");
    } catch (err: any) {
      console.error("[FRONTEND] AcceptOffer error:", err);
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
    console.log("[FRONTEND] acceptAnswer called, answer length:", answer?.length);
    try {
      console.log("[FRONTEND] Calling AcceptAnswer()...");
      await AcceptAnswer(answer);
      console.log("[FRONTEND] AcceptAnswer returned successfully");
      set({ status: "connected" });
      get().addLog("Tunnel established!");
    } catch (err: any) {
      console.error("[FRONTEND] AcceptAnswer error:", err);
      set({ status: "error" });
      get().addLog(`Error: ${err.message || err}`);
      useToastStore.getState().addToast({
        title: "Failed to accept answer",
        description: err.message || "An unknown error occurred",
        variant: "destructive",
      });
    }
  },

  exportToken: async (token, tokenType) => {
    try {
      // Create a blob and download it as a file
      const blob = new Blob([token], { type: "text/plain" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `covenant-${tokenType}.txt`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
      get().addLog(`Token exported to file`);
      useToastStore.getState().addToast({
        title: "Token exported successfully",
        description: "Saved to your browser's Downloads folder",
        variant: "default",
      });
    } catch (err: any) {
      get().addLog(`Error exporting: ${err.message || err}`);
      useToastStore.getState().addToast({
        title: "Failed to export token",
        description: err.message || "An unknown error occurred",
        variant: "destructive",
      });
    }
  },

  importToken: async (file: File) => {
    try {
      if (!file) {
        get().addLog(`No file selected for import`);
        return undefined;
      }
      const text = await file.text();
      get().addLog(`Token imported from ${file.name}`);
      useToastStore.getState().addToast({
        title: "Token imported successfully",
        description: `Loaded from ${file.name}`,
        variant: "default",
      });
      return text;
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
