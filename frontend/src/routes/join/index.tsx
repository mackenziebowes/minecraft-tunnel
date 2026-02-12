import React, { useEffect, useRef, useState } from "react";
import { useAppStore } from "@/lib/store";
import { useTunnelStore } from "@/lib/tunnelStore";
import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime";
import { TokenCard } from "@/components/custom/token-card";
import Sigil from "@/components/custom/sigil";

import {
  ArrowLeft,
  Activity,
  Terminal,
  FileUp,
  Link as LinkIcon,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";

export const JoinView = () => {
  const { setRoute } = useAppStore();
  const { status, logs, answerToken, acceptOffer, addLog, setStatus, importToken, exportToken } =
    useTunnelStore();

  const scrollRef = useRef<HTMLDivElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [offerInput, setOfferInput] = useState("");

  useEffect(() => {
    EventsOn("log", addLog);
    EventsOn("status-change", (newStatus: string) =>
      setStatus(newStatus as any),
    );
    return () => {
      EventsOff("log");
      EventsOff("status-change");
    };
  }, [addLog, setStatus]);

  useEffect(() => {
    scrollRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [logs]);

  const handlePasteOffer = async () => {
    try {
      await acceptOffer(offerInput);
      setOfferInput("");
    } catch (err: any) {
      addLog(`Error: ${err.message || err}`);
    }
  };

  const handleImportFile = async () => {
    if (fileInputRef.current) {
      fileInputRef.current.click();
    }
  };

  const handleFileSelected = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      const token = await importToken(file);
      if (token) {
        setOfferInput(token);
      }
    }
  };

  const statusColor =
    {
      disconnected: "border-slate-200",
      connecting: "border-yellow-200 animate-pulse",
      "waiting-for-answer": "border-blue-200",
      "waiting-for-host": "border-blue-200",
      connected: "border-green-200",
      error: "border-red-200",
    }[status] || "";

  return (
    <div className="flex flex-col items-center justify-center min-h-screen min-w-screen p-6 font-sans">
      <Sigil scale={0.25} rotating />
      <Card className="w-full shadow-xl">
        <CardHeader className="pb-4">
          <div className="flex justify-between items-start">
            <div className="space-y-1">
              <CardTitle className="text-xl font-bold tracking-tight">
                Join Tunnel
              </CardTitle>
              <CardDescription>
                Connect to a friend's Minecraft server.
              </CardDescription>
            </div>
            <Badge
              variant="outline"
              className={`${statusColor} capitalize px-3 py-1`}
            >
              {status === "connected" && <Activity className="w-3 h-3 mr-1" />}
              {status.replace(/-/g, " ")}
            </Badge>
          </div>
        </CardHeader>

        <CardContent className="space-y-6">
          {/* Offer Input Section */}
          {status === "disconnected" && (
            <div className="space-y-2">
              <Label className="text-sm font-medium">
                Paste Offer Token from your friend:
              </Label>
              <div className="flex gap-2">
                <Input
                  placeholder="Paste offer token here..."
                  value={offerInput}
                  onChange={(e) => setOfferInput(e.target.value)}
                  className="font-mono text-xs flex-1"
                />
                <Button onClick={handlePasteOffer} disabled={!offerInput}>
                  Connect
                </Button>
              </div>
              <div className="flex gap-2">
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".txt"
                  onChange={handleFileSelected}
                  className="hidden"
                />
                <Button variant="outline" size="sm" onClick={handleImportFile}>
                  <FileUp className="w-4 h-4 mr-2" />
                  Import File
                </Button>
                <Button variant="outline" size="sm" onClick={() => {
                  navigator.clipboard.readText().then(text => setOfferInput(text));
                }}>
                  <LinkIcon className="w-4 h-4 mr-2" />
                  Paste from Clipboard
                </Button>
              </div>
            </div>
          )}

          {/* Answer Token Section */}
          {(status === "waiting-for-host" || status === "connected") && (
            <div className="space-y-2">
              <Label className="text-sm font-medium">
                Share this Answer Token back to your friend:
              </Label>
              <TokenCard token={answerToken} type="answer" onExport={(token, type) => exportToken(token, type)} />
              <div className="text-sm text-slate-600 bg-yellow-50 border border-yellow-200 rounded p-3">
                <strong>⚠️ Important:</strong> Copy and send this back to the
                host!
              </div>
            </div>
          )}

          {/* Logs */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label className="text-sm font-medium flex items-center gap-2">
                <Terminal className="w-4 h-4" />
                Live Logs
              </Label>
              <span className="text-xs">{logs.length} events</span>
            </div>
            <div className="rounded-lg border p-4 shadow-inner">
              <ScrollArea className="h-64 w-full pr-4">
                <div className="flex flex-col gap-1 font-mono text-xs">
                  {logs.length === 0 && (
                    <div className="italic select-none py-10 text-center">
                      Waiting for connection...
                    </div>
                  )}
                  {logs.map((log, i) => (
                    <div
                      key={i}
                      className="break-all border-l-2 border-transparent pl-2 hover:border-slate-700 hover:bg-slate-900/50 transition-colors"
                    >
                      <span className="mr-2 text-slate-500">
                        {new Date().toLocaleTimeString([], { hour12: false })}
                      </span>
                      {log}
                    </div>
                  ))}
                  <div ref={scrollRef} />
                </div>
              </ScrollArea>
            </div>
          </div>
        </CardContent>

        <CardFooter className="flex justify-between pt-2 border-t border-slate-100">
          <Button
            variant="ghost"
            onClick={() => setRoute("/")}
            disabled={status !== "disconnected"}
          >
            <ArrowLeft className="w-4 h-4 mr-2" />
            Back
          </Button>

          {status === "connected" && (
            <Badge className="bg-green-100 text-green-800">
              <Activity className="w-3 h-3 mr-1" />
              Connected
            </Badge>
          )}
        </CardFooter>
      </Card>
    </div>
  );
};
